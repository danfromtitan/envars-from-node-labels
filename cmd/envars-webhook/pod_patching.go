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
	"github.com/google/uuid"
	corev1 "k8s.io/api/core/v1"
	"log"
)

// Inject env vars from node labels only for selected containers
var allowedContainerNames = map[string]bool{
	"ingester":      true,
	"store-gateway": true,
}

// Add secret reference next to existing sources. The idea here is to preserve configuration coming from an upstream
// manifest (i.e. Helm chart) and append our secret reference to sources that already exist (if any).
// The secret name is created dynamically and stored in pod metadata for further use (i.e. create and delete).
func containerEnvFromSource(envSources []corev1.EnvFromSource, secretName string) []corev1.EnvFromSource {
	secretSource := &corev1.EnvFromSource{
		SecretRef: &corev1.SecretEnvSource{
			LocalObjectReference: corev1.LocalObjectReference{
				Name: secretName}}}
	envSources = append(envSources, *secretSource)

	return envSources
}

// Create event: make a random secret name
// Update event: retrieve secret name from pod label
func createSecretNameIfEmpty(secretName string) string {
	if len(secretName) == 0 {
		return fmt.Sprintf("envars-%s", uuid.New())
	}
	return secretName
}

// Create event: secret reference is added next to existing sources and secret name is stored in pod label
// Update event: in case of pod update, kubectl apply will stumble upon the secret reference source we've inserted
// because that doesn't exist with the outside manifest. At the same time pod is not recreated, only the container gets
// restarted. That means we can keep the same secret reference since the pod stays on the same node and only have
// to re-patch the env sources with the same patches we did during pod create event.
// That is not an issue with deployment updates, in that case pods are simply deleted and recreated.
func patchPod(pod corev1.Pod) []patchOperation {
	var patches []patchOperation
	var addSecretLabel bool

	// Create event: make a random secret name
	// Update event: retrieve secret name from pod label
	secretName := createSecretNameIfEmpty(pod.Labels["envars-secret-name"])

	// Loop through the list of containers and create a list of envFromSource patches
	for i := range pod.Spec.Containers {
		container := pod.Spec.Containers[i]
		if !allowedContainerNames[container.Name] {
			log.Printf("%s container patching not allowed", container.Name)
		} else {
			addSecretLabel = true
			containerEnvSource := containerEnvFromSource(container.EnvFrom, secretName)
			patches = append(patches, patchOperation{
				Op:    "replace",
				Path:  fmt.Sprintf("/spec/containers/%d/envFrom", i),
				Value: containerEnvSource,
			})
			log.Printf("patched envFrom source for container %s of pod %s", container.Name, pod.Name)
		}
	}

	// Store secret name in the pod's metadata label if at least one container is allowed to receive the env vars.
	// The add operation from JSON patch will add or replace the secret label value
	// (https://datatracker.ietf.org/doc/html/rfc6902#section-4.1)
	if addSecretLabel {
		patches = append(patches, patchOperation{
			Op:    "add",
			Path:  "/metadata/labels/envars-secret-name",
			Value: secretName,
		})
	}

	return patches
}
