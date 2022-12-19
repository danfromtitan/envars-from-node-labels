/*
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

   http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"context"
	"fmt"
	"log"
	"regexp"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Map the node labels to environment variables. First we find the node object using the name found in the pods/binding
// target, then we retrieve all node labels, then we map these to env vars. For the environment variable name we use
// the node label key, then we replace all special characters with underscore, then turn it to uppercase. Also,
// prepend env var name with NODE_ to avoid possible overlapping with existing env var names. The service account
// that runs the admission controller deployment must have read access to the cluster nodes.
func createEnvVarsFromNodeLabels(binding corev1.Binding) map[string]string {
	ks := kubeClient()

	// Get the node object
	nodeName := binding.Target.Name
	node, getErr := ks.CoreV1().Nodes().Get(context.TODO(), nodeName, metav1.GetOptions{})
	if errors.IsNotFound(getErr) {
		log.Printf("node %s not found", nodeName)
	} else if statusError, isStatus := getErr.(*errors.StatusError); isStatus {
		log.Printf("error getting node: %v", statusError.ErrStatus.Message)
	} else if getErr != nil {
		log.Fatal(getErr)
	} else {
		log.Printf("found node %s", nodeName)
	}

	// Create env vars mapping
	envars := make(map[string]string)
	reg, err := regexp.Compile("[^A-Za-z0-9]+")
	if err != nil {
		log.Fatal(err)
	}
	for k, v := range node.Labels {
		envars[strings.ToUpper(reg.ReplaceAllString("node."+k, "_"))] = v
	}

	return envars
}

// Create the secret with container environment variables from node labels. The secret name was created dynamically
// during the pod creation event, and it was attached to a pod's metadata label. That means, before we can create
// the secret, we need to get the pod object from the binding request and retrieve the metadata label of the secret name.
// Secret is then created using env vars mapped by createEnvVarsFromNodeLabels. The service account that runs
// the admission controller deployment must have create/delete access to secrets and read access to pods in the namespace.
func createSecret(binding corev1.Binding) error {
	ks := kubeClient()
	var err error

	// Get pod object
	podInterface := ks.CoreV1().Pods(binding.Namespace)
	pod, err := podInterface.Get(context.TODO(), binding.Name, metav1.GetOptions{})
	if errors.IsNotFound(err) {
		log.Fatalf("pod %s in namespace %s not found", binding.Name, binding.Namespace)
	} else if statusError, isStatus := err.(*errors.StatusError); isStatus {
		log.Fatalf("error getting pod %s in namespace %s: %v", binding.Name, binding.Namespace, statusError.ErrStatus.Message)
	} else if err != nil {
		panic(fmt.Errorf("failed to get pod %s: %v", binding.Name, err))
	}

	// Get secret name from pod metadata label
	secretName := pod.Labels["envars-secret-name"]

	// Create secret with env vars
	if len(secretName) > 0 {
		secret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: pod.Namespace,
				Name:      secretName,
			},
			StringData: createEnvVarsFromNodeLabels(binding),
			Type:       corev1.SecretTypeOpaque,
		}

		createdSecret, err := ks.CoreV1().Secrets(secret.Namespace).Create(context.TODO(), secret, metav1.CreateOptions{})
		if statusError, isStatus := err.(*errors.StatusError); isStatus {
			log.Fatalf("error creating secret %s in namespace %s: %v", secretName, pod.Namespace, statusError.ErrStatus.Message)
		} else if err != nil {
			panic(fmt.Errorf("failed to create secret %s: %v", secretName, err))
		} else {
			log.Printf("secret %s created successfully", createdSecret.Name)
		}
	}

	return err
}
