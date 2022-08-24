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
  -n $NAMESPACE \
  --create-namespace \
  envars-webhook envars-webhook/envars-webhook \
  --set webhook.namespaceSelector=samples \
  --set webhook.verboseLogs=false \
  --set webhook.containersAllowed.ingester=true,webhook.containersAllowed.prober=true,webhook.containersAllowed.store-gateway=true
```


### Verification

Follow the notes in Helm deployment output to verify the deployment.


### Uninstall

```bash
NAMESPACE=webtest
helm uninstall -n $NAMESPACE envars-webhook
```
