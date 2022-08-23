# Usage

Follow the instructions below to deploy the admission controller.

### Helm requirements

```bash
helm repo add envars-webhook https://danfromtitan.github.io/envars-from-node-labels/
helm repo update
helm search repo envars-webhook
```


### Install/upgrade

Update values as needed and deploy the webhook.

```bash
NAMESPACE=webtest
helm upgrade --install \
  --namespace $NAMESPACE \
  --create-namespace \
  envars-webhook envars-webhook/envars-webhook \
  -f values.yaml
```


### Verification

```bash
NAMESPACE=webtest
kubectl get secret -n $NAMESPACE envars-webhook-tls -o 'go-template={{index .data "tls.crt"}}' | base64 -d | openssl x509 -text -noout
kubectl get pods -n $NAMESPACE
kubectl logs -f -n $NAMESPACE pod-name
kubectl get mutatingwebhookconfigurations envars-webhook -o yaml
```


### Uninstall

```bash
NAMESPACE=webtest
helm uninstall -n $NAMESPACE envars-webhook
```
