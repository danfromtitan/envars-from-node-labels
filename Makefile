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

# Makefile for building the Admission Controller webhook demo server and docker image.

IMAGE_NAME="envars-webhook"
NAMESPACE = "envars-webhook"

.DEFAULT_GOAL := image

deps:
	TMPDIR=/var/tmp GO111MODULE=on go get -v ./...

envars-webhook: $(shell find . -name '*.go')
	TMPDIR=/var/tmp CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o $@ ./cmd/envars-webhook

image: envars-webhook
	docker rmi ${IMAGE_NAME} || true
	docker build --no-cache -t ${IMAGE_NAME} .

push:
	docker push ${IMAGE_NAME}

tls:
	NAMESPACE=${NAMESPACE} deploy/tls.sh 

deploy: 
	NAMESPACE=${NAMESPACE} IMAGE_NAME=${IMAGE_NAME} deploy/deploy.sh apply

undeploy:
	NAMESPACE=${NAMESPACE} IMAGE_NAME=${IMAGE_NAME} deploy/deploy.sh delete

sample:
	kubectl apply -f samples/namespace.yaml
	kubectl apply -f samples/env-configmap.yaml
	kubectl apply -f samples/env-secrets.yaml

unsample:
	kubectl delete -f samples/ || true

.PHONY: image push tls deploy undeploy sample unsample
