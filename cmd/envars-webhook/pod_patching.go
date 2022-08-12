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

// Add secret reference next to existing sources. The idea here is to preserve configuration coming from an upstream
// manifest (i.e. Helm chart) and append our secret reference to sources that already exist (if any).
// The secret name is created dynamically and stored in pod metadata for further use (i.e. create and delete).
//
// There is a bug in K8s API that inserts different envSources content in the new object during UPDATE events:
// - if there is an existing envSources from upstream (i.e. Helm chart), our env vars secret is not found in the new
//   object, so we need to re-patch envSources during an UPDATE event.
// - when envSources comes empty from upstream (i.e. Helm chart), the new object does have our env vars secret in the
//   new object, and we need to prevent re-patching, otherwise the envSources would have the same secret twice.
func containerEnvFromSource(envSources []corev1.EnvFromSource, secretName string) []corev1.EnvFromSource {
	secretSource := &corev1.EnvFromSource{
		SecretRef: &corev1.SecretEnvSource{
			LocalObjectReference: corev1.LocalObjectReference{
				Name: secretName}}}

	appendSecret := true
	for _, envSrc := range envSources {
		if (envSrc.SecretRef != nil) && (envSrc.SecretRef.Name == secretName) {
			appendSecret = false
		}
	}
	if appendSecret {
		envSources = append(envSources, *secretSource)
	}

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

// Retrieve the pod name or the generated prefix (i.e. deployments, statefulsets)
func podName(pod corev1.Pod) string {
	if len(pod.Name) > 0 {
		return pod.Name
	}
	return pod.GetGenerateName()
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

	// Loop through the list of (init)containers and create a list of envFromSource patches
	for i, container := range pod.Spec.InitContainers {
		log.Printf(">>> looking at INIT container %s in pod %s", container.Name, pod.Name)
		addSecretLabel, patches = patchContainer(container, podName(pod), i, addSecretLabel, secretName, patches, "initContainers")
	}
	for i, container := range pod.Spec.Containers {
		log.Printf(">>> looking at container %s in pod %s", container.Name, pod.Name)
		addSecretLabel, patches = patchContainer(container, podName(pod), i, addSecretLabel, secretName, patches, "containers")
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

// Container patching helper
func patchContainer(container corev1.Container, podName string, index int, addSecretLabel bool, secretName string, patches []patchOperation, containerType string) (bool, []patchOperation) {
	if !config.ContainersAllowed[container.Name] {
		log.Printf("%s container patching not allowed", container.Name)
	} else {
		addSecretLabel = true
		containerEnvSource := containerEnvFromSource(container.EnvFrom, secretName)
		patches = append(patches, patchOperation{
			Op:    "replace",
			Path:  fmt.Sprintf("/spec/%v/%d/envFrom", containerType, index),
			Value: containerEnvSource,
		})
		log.Printf("patched envFrom source for container %s in pod %s", container.Name, podName)
	}
	return addSecretLabel, patches
}
