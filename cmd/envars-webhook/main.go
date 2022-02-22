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
	"fmt"
	admission "k8s.io/api/admission/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

const (
	tlsDir      = `/run/secrets/tls`
	tlsCertFile = `tls.crt`
	tlsKeyFile  = `tls.key`
)

var (
	podResource = metav1.GroupVersionResource{Version: "v1", Resource: "pods"}
	VerboseLogs bool
)

// Enable debug logging
func init() {
	val := os.Getenv("GODEBUG")
	if strings.Contains(val, "webhook2debug=1") {
		VerboseLogs = true
	}
}

// In cluster config
func kubeClient() kubernetes.Clientset {
	// Create the in-cluster config
	config, err := rest.InClusterConfig()
	if err != nil {
		panic(err)
	}
	// Create the client set
	ks, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err)
	}

	return *ks
}

// applyEnvPatching adds environment variables from a secret that is created later in the pods/binding event.
// The admission controller will patch every pod that is created outside of Kubernetes namespaces with an envFrom
// reference that is appended to existing configurations, checking that the container is allowed to receive the updates.
// The secret name must be created upfront, and we store it in a pod's metadata label.
func mutateDispatch(req *admission.AdmissionRequest) ([]patchOperation, error) {
	// If this gets invoked on an object of a different kind, then log a message and return an empty patch, allowing the
	// request to pass through unchanged.
	if req.Resource != podResource {
		log.Printf("pass the buck on %s, only admit %s ATM.", req.Resource, podResource)
		return nil, nil
	}

	var err error
	var patches []patchOperation

	// Parse the request object and patch the pod with secret reference at pod creation event
	if (req.RequestKind.Kind == "Pod") && (req.Operation == "CREATE" || req.Operation == "UPDATE") {
		raw := req.Object.Raw
		pod := corev1.Pod{}
		if _, _, err := universalDeserializer.Decode(raw, nil, &pod); err != nil {
			return nil, fmt.Errorf("could not deserialize pod object: %v", err)
		}
		patches = patchPod(pod)
	}

	// Parse the request object and create secret that holds the environment variables at pod binding event
	if (req.RequestKind.Kind == "Binding") && (req.Operation == "CREATE") {
		raw := req.Object.Raw
		binding := corev1.Binding{}
		if _, _, err := universalDeserializer.Decode(raw, nil, &binding); err != nil {
			return nil, fmt.Errorf("could not deserialize binding object: %v", err)
		}
		err = createSecret(binding)
		if err != nil {
			return nil, err
		}
	}

	// Parse the request old object and delete envs secret at pod delete event
	if (req.RequestKind.Kind == "Pod") && (req.Operation == "DELETE") {
		pod := corev1.Pod{}
		raw := req.OldObject.Raw
		if _, _, err := universalDeserializer.Decode(raw, nil, &pod); err != nil {
			return nil, fmt.Errorf("could not deserialize pod object: %v", err)
		}
		err = deleteSecret(pod)
		if err != nil {
			return nil, err
		}
	}

	return patches, err
}

func main() {
	certPath := filepath.Join(tlsDir, tlsCertFile)
	keyPath := filepath.Join(tlsDir, tlsKeyFile)

	log.Println("starting admission controller ...")
	mux := http.NewServeMux()
	mux.Handle("/mutate", admitFuncHandler(mutateDispatch))
	server := &http.Server{
		Addr:    ":8443",
		Handler: mux,
	}
	log.Fatal(server.ListenAndServeTLS(certPath, keyPath))
}
