#!/usr/bin/env bash
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#    http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# tls.sh
#
# Generate a (self-signed) CA certificate and a certificate and private key to be used by the webhook demo server.
# The certificate will be issued for the Common Name (CN) of `envars-webhook.webhook.svc`, which is the
# cluster-internal DNS name for the service.

set -euo pipefail

# Generate keys into a temporary directory.
echo "Generating TLS keys ..."
key_dir="$(mktemp -d)"
trap 'rm -rf -- "${key_dir}"' EXIT
chmod 0700 "${key_dir}"
cd "${key_dir}"

cat > csr.conf <<EOF
[req]
req_extensions = v3_req
distinguished_name = req_distinguished_name
prompt = no
[req_distinguished_name]
CN = envars-webhook.${NAMESPACE}.svc
[ v3_req ]
basicConstraints = CA:FALSE
keyUsage = nonRepudiation, digitalSignature, keyEncipherment
extendedKeyUsage = serverAuth
subjectAltName = @alt_names
[alt_names]
DNS.1 = envars-webhook
DNS.2 = envars-webhook.${NAMESPACE}
DNS.3 = envars-webhook.${NAMESPACE}.svc
DNS.4 = envars-webhook.${NAMESPACE}.svc.cluster.local
EOF

# Generate the CA cert and private key
openssl req -nodes -new -x509 -keyout ca.key -out ca.crt -subj "/CN=Admission Controller Webhook CA"

# Generate the private key for the webhook server
openssl genrsa -out envars-webhook-tls.key 2048

# Generate a Certificate Signing Request (CSR) for the private key, and sign it with the private key of the CA.
openssl req -new -key envars-webhook-tls.key -subj "/CN=envars-webhook.${NAMESPACE}.svc" -config csr.conf \
    | openssl x509 -req -CA ca.crt -CAkey ca.key -CAcreateserial -out envars-webhook-tls.crt -extensions v3_req -extfile csr.conf

# Create the TLS secret for the keys that were generated
cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: Namespace
metadata:
  name: ${NAMESPACE}
  labels:
    name: ${NAMESPACE}
---
apiVersion: v1
kind: Secret
metadata:
  name: envars-webhook-tls
  namespace: ${NAMESPACE}
type: kubernetes.io/tls
data:
  tls.key: $(cat ${key_dir}/envars-webhook-tls.key | base64 -w 0)
  tls.crt: $(cat ${key_dir}/envars-webhook-tls.crt | base64 -w 0)
  ca.crt: $(cat ${key_dir}/ca.crt | base64 -w 0)
EOF
