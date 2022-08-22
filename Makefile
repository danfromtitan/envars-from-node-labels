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

AWS_ACCOUNT_ID = $$(aws sts get-caller-identity --query Account --output text)
AWS_REGION = $$(aws configure get region)
IMAGE_NAME = $$(basename `pwd`)

IMAGE_URL ?= "$(AWS_ACCOUNT_ID).dkr.ecr.$(AWS_REGION).amazonaws.com/$(IMAGE_NAME):latest"
NAMESPACE ?= "envhook"

.DEFAULT_GOAL := image

deps:
	TMPDIR=/var/tmp GO111MODULE=on go get -v ./...
	go mod tidy

envars-webhook: deps
	TMPDIR=/var/tmp CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o $@ ./cmd/envars-webhook

image: envars-webhook
	docker rmi $(AWS_ACCOUNT_ID).dkr.ecr.$(AWS_REGION).amazonaws.com/$(IMAGE_NAME):latest || true
	docker build --no-cache -t $(AWS_ACCOUNT_ID).dkr.ecr.$(AWS_REGION).amazonaws.com/$(IMAGE_NAME):latest .

push:
	aws ecr get-login-password --region $(AWS_REGION) | docker login --username AWS $(AWS_ACCOUNT_ID).dkr.ecr.$(AWS_REGION).amazonaws.com --password-stdin
	aws ecr batch-delete-image --repository-name $(IMAGE_NAME) --image-ids imageDigest=$$(aws ecr list-images --repository-name $(IMAGE_NAME) | jq -r ' .imageIds[] | [ .imageDigest ] | @tsv ')
	docker push $(AWS_ACCOUNT_ID).dkr.ecr.$(AWS_REGION).amazonaws.com/$(IMAGE_NAME):latest

tls:
	NAMESPACE=${NAMESPACE} deploy/tls.sh

deploy: undeploy
	NAMESPACE=${NAMESPACE} IMAGE_URL=${IMAGE_URL} deploy/deploy.sh create

undeploy:
	NAMESPACE=${NAMESPACE} IMAGE_URL=${IMAGE_URL} deploy/deploy.sh delete

sample:
	kubectl apply -f test/namespace.yaml
	kubectl apply -f test/env-configmap.yaml
	kubectl apply -f test/env-secrets.yaml

unsample:
	kubectl delete -f test/ || true

.PHONY: image push tls deploy undeploy sample unsample
