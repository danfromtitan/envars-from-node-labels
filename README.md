# Create container env vars from node labels

This repository contains a K8s admission controller that implements a [MutatingAdmissionWebhook](https://kubernetes.io/docs/reference/access-authn-authz/admission-controllers/#mutatingadmissionwebhook) 
which together with additional helper methods would expose node labels as environment variables in selected pod containers.

The original problem it solves was to expose the availability zone information from node labels to pod environment 
variables. Often times, highly available application need to be aware of the availability zone of their deployment and, 
unless you have dedicated deployments for each AZ, that configuration cannot be given to a container at deployment time.

The current implementation went beyond the initial scope and features:

- all node labels are exposed as environment variables to containers
- select the container names where the node labels would be exposed
- insert the env vars node labels next to existing env vars from other sources (i.e. configmap, secret)
- support for single pods as well as deployments and stateful sets

As with [any other admission controller](https://kubernetes.io/docs/reference/access-authn-authz/extensible-admission-controllers/#side-effects), 
you need to stay clean of unwanted results during pod creation and deletion events. The implementation comes with a 
number of risk mitigations factors:

- admission requests are limited to pods and pods/binding resources
- namespace where requests are accepted from must be explicitly matched
- avoids taking requests from kube-public and kube-system namespaces
- logs most acceptable errors and won't return an exception unless the intended functionality is seriously impeded.


## Design in a nutshell

The implementation combines the JSON patching feature of the mutating admission controller with in-cluster client-go calls 
to K8s API for selected events. That is needed because, on one side, env variables are immutable resources, and on the 
other side, node labels are unknown to a pod until the pod is scheduled for binding to a node.

The sequence of events is:

- before pod is created, the webhook receives a pod admission request
- webhook mutates the request by adding env vars from a TBD secret source
- at the same time, the secret name is patched to a pod label for later use (i.e. secret create and delete)
- the next event is pods/binding when we learn the node where the pod will be deployed
- that means node labels could now be determined and turned into key-value pairs for the secret
- at that time, before pod is started, the webhook creates the secret with the env vars from node labels
- additional consideration is given to pod updates, EnvFromSource is re-patched with the existing secret to avoid an immutable resource exception
- when pod is deleted, the webhook looks for the secret name label of the pod and if found, it deletes the secret.

For client-go request to work with in-cluster API calls, the service account running the webhook pod needs permissions 
on selected cluster resources. These are configured as part of the webhook deployment.  


## Prerequisites

- Kubernetes cluster ver 1.9.0 or above
- ideally, for scripts to work out of the box, you'd want to use an EKS cluster. In lack of that you'd need to modify 
  deployment script and provide equivalent outcomes for the aws CLI commands.
- `admissionregistration.k8s.io/v1beta1` API enabled. In addition, the `MutatingAdmissionWebhook` admission controller 
  should be added and [listed in the admission-control](https://docs.aws.amazon.com/eks/latest/userguide/platform-versions.html) 
  flag of `kube-apiserver`.
- [GNU make](https://www.gnu.org/software/make/), [Go](https://golang.org) and [Docker engine](https://docs.docker.com/engine/install/) 
  are required to build the image


## Build

- Create and push the image

```bash
make
make push
```

## Deployment

- Verify you have `admissionregistration.k8s.io/v1beta1` installed in your cluster

```bash
kubectl api-versions | grep admissionregistration.k8s.io/v1beta1
```

- Create TLS self-signed certificate for the webhook, package cert, key and CAcert in a secret

```bash
make tls
```


- Enable debug logging in the deployment manifest to see the JSON request and response in server logs.

```yaml
      containers:
        - name: server
          image: ${IMAGE_NAME}
          imagePullPolicy: Always
          env:
            - name: GODEBUG
              value: webhook2debug=1
```


- Deploy the webhook. Note that deploy target will also try to do a cleanup.

```bash
make deploy
```

- Run the undeploy target if you want to do a permanent cleanup.

```bash
make undeploy
```


## Verification

- The `envars-webhook` pod in the `webhook` namespace should be running, logs show admission controller activity

```bash
kubectl get pods -n webhook
k logs -f -n webhook envars-webhook-12345678-abcde
```


- Mutating webhook `envars-webhook` should exist

```bash
kubectl get mutatingwebhookconfigurations
```


- Create a test namespace with a pre-existing configmap and secret. Note the unsample target will clean up all resources 
  under the sample directory. 

```bash
make sample
```


- Create a pod with a single container that has a pre-existing configmap. The container's env should return the key-value 
  pairs from pre-existing configmap along with the env vars created from node labels.

```bash
kubectl apply -f samples/pod-allowed.yaml
kubectl logs -n samples pod-allowed
```


- Create a pod with mixed containers, some allowed and some not allowed to take env vars from noe labels. The containers' 
  env should return the key-value pairs from pre-existing configmap and secret along with the env vars created from node 
  labels where container was allowed.

```bash
kubectl apply -f samples/pod-mixed.yaml
kubectl logs -n samples pod-mixed ingester
kubectl logs -n samples pod-mixed store-gateway
kubectl logs -n samples pod-mixed compactor
```


- Create a pod with a container that is not allowed to receive env vars from node labels. The container's env should only 
  return the key-value pairs from pre-existing configmap and secret.

```bash
kubectl apply -f samples/pod-excluded.yaml
kubectl logs -n samples pod-excluded
```


- Create a deployment with a container that has a pre-existing configmap and secret and is allowed to receive env vars 
  from node labels. The containers' env should return the key-value pairs from pre-existing configmap and secret, along 
  with the env vars created from node labels.

```bash
kubectl apply -f samples/deployment.yaml
kubectl exec -it -n samples deployment-12345abcde-abcde -- env
```


- Create a statefulset with a container that has a pre-existing configmap and secret and is allowed to receive env vars
  from node labels. The containers' env should return the key-value pairs from pre-existing configmap and secret, along
  with the env vars created from node labels.

```bash
kubectl apply -f samples/statefulset.yaml
kubectl exec -it -n samples statefulset-0 -- env
```


- Cleanup verification resources

```bash
make unsample
```


## Credits

The admission controller implementation was inspired from the outstanding 
[admission-controller-webhook-demo](https://github.com/stackrox/admission-controller-webhook-demo) project.
