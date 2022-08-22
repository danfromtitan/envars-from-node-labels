# Usage

Follow the instructions below to deploy envars-from-node-labels webhook.


### Helm requirements

```bash
helm repo add envars-webhook https://danfromtitan.github.io/envars-from-node-labels/
helm repo update
helm search repo envars-webhook
```

On a freshly cloned work directory, the chart dependency will be missing, so you need to update it.

```bash
cd setup/external-dns
helm dep update
helm dep list
```


### Install/upgrade

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
