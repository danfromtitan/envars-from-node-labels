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

# deploy.sh
#
# Sets up the environment for the admission controller webhook demo in the active cluster.

set -euo pipefail

: ${1?'missing kubectl action'}

basedir="$(dirname "$0")"

# Read the PEM-encoded CA certificate from secret, base64 decode it and replace the `${CA_PEM_B64}` placeholder 
# in the YAML template, then create the Kubernetes admission controller resources
CA_PEM_B64="$(kubectl get secret -n webhook envars-webhook-tls -o 'go-template={{index .data "ca.crt"}}')"
IMAGE_NAME="$(aws sts get-caller-identity --query Account --output text).dkr.ecr.$(aws configure get region).amazonaws.com/$(basename `pwd`):latest"
sed -e 's@${CA_PEM_B64}@'"${CA_PEM_B64}"'@g' \
  -e 's@${IMAGE_NAME}@'"${IMAGE_NAME}"'@g' < "${basedir}/deployment.yaml.template" \
  | kubectl $1 -f - || true

echo "The webhook server has been $1d"
