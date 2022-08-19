### Install/upgrade

```bash
cd charts/envars-webhook
NAMESPACE=webtest
helm upgrade --install \
  --namespace $NAMESPACE \
  --create-namespace \
  envars-webhook . \
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

# change