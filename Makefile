# Copyright 2021 The Telemetry Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

SHELL=/bin/bash
.SHELLFLAGS=-euo pipefail -c
COMPONENTS = kubernetes-agent kubermatic-agent reporter
IMAGE_ORG = quay.io/kubermatic
VERSION = v0.1.0
KIND_CLUSTER ?= telemetry
URL ?= https://telemetry.k8c.io/api/v1

export CGO_ENABLED:=0

# Dev Image to use
# Always bump this version, when changing ANY component version below.
DEV_IMAGE_TAG=v1
# Versions used to build DEV image:
export CONTROLLER_GEN_VERSION=0.4.0

ifdef CI
	# prow sets up GOPATH and we want to make sure it's in the PATH
	# https://github.com/kubernetes/test-infra/issues/9469
	# https://github.com/kubernetes/test-infra/blob/895df89b7e4238125063157842c191dac6f7e58f/prow/pod-utils/decorate/podspec.go#L474
	export PATH:=${PATH}:${GOPATH}/bin
endif

# -----------------
# Compile & Release
# -----------------
bin/linux_amd64/%: GOARGS = GOOS=linux GOARCH=amd64
bin/darwin_amd64/%: GOARGS = GOOS=darwin GOARCH=amd64
bin/windows_amd64/%: GOARGS = GOOS=windows GOARCH=amd64

bin/%:
	$(eval COMPONENT=$(shell basename $*))
	$(GOARGS) go build $(BUILD_ARGS) -o bin/$* cmd/$(COMPONENT)/main.go

release:
	goreleaser release --rm-dist

# ----------------
# Deployment and Installation
# ----------------
setup-cluster:
	@kind create cluster --name=${KIND_CLUSTER} --image=${KIND_NODE_IMAGE} || true

kind-deploy-agent: setup-cluster kind-load deploy-agent

deploy-agent:
	@kubectl create namespace telemetry-system || true  # ignore if exists
	@kustomize build config/agent/default | \
		sed "s|<URL_PLACEHOLDER>|${URL}|g" | \
		sed "s|<UUID_PLACEHOLDER>|$$(uuidgen | base64)|g" | \
		sed "s|quay.io/kubermatic/telemetry-reporter:v0.1.0|${IMAGE_ORG}/telemetry-reporter:${VERSION}|g" | \
		sed "s|quay.io/kubermatic/telemetry-kubernetes-agent:v0.1.0|${IMAGE_ORG}/telemetry-kubernetes-agent:${VERSION}|g" | \
		sed "s|quay.io/kubermatic/telemetry-kubermatic-agent:v0.1.0|${IMAGE_ORG}/telemetry-kubermatic-agent:${VERSION}|g" | \
		kubectl apply -f -

# ---------------
# Code Generators
# ---------------
generate:
ifdef CI
	@hack/codegen.sh
else
	@docker run --rm -e CI=true \
		-w /go/src/k8c.io/telemetry \
		-v $(PWD):/go/src/k8c.io/telemetry:delegated \
		--user "$(id -u):$(id -g)" \
		${IMAGE_ORG}/telemetry-dev:${DEV_IMAGE_TAG} \
		make generate
endif

# ------------
# Test Runners
# ------------
test:
	CGO_ENABLED=1 go test -race -v ./pkg/...
.PHONY: test

lint: pre-commit
	@hack/validate-directory-clean.sh
	golangci-lint run ./...  --deadline=15m

# -------------
# Util Commands
# -------------
fmt:
	go fmt ./...

vet:
	go vet ./...

tidy:
	go mod tidy

pre-commit:
	pre-commit run -a

require-docker:
	@docker ps > /dev/null 2>&1 || start-docker.sh || ./hack/start-docker.sh || (echo "cannot find running docker daemon nor can start new one" && false)
	@[[ -z "${QUAY_IO_USERNAME}" ]] || ( echo "logging in to ${QUAY_IO_USERNAME}" && docker login -u ${QUAY_IO_USERNAME} -p ${QUAY_IO_PASSWORD} quay.io )
.PHONY: require-docker

# ----------------
# Container Images
# ----------------
push-images: $(addprefix push-image-, $(COMPONENTS))

# build all container images except the test image
build-images: $(addprefix build-image-, $(COMPONENTS))

kind-load: $(addprefix kind-load-, $(COMPONENTS))

.SECONDEXPANSION:
build-image-%: bin/linux_amd64/$$* require-docker
	@mkdir -p bin/image/$*
	@mv bin/linux_amd64/$* bin/image/$*
	@cp -a config/dockerfiles/$*.Dockerfile bin/image/$*/Dockerfile
	@docker build -t ${IMAGE_ORG}/telemetry-$*:${VERSION} bin/image/$*

push-image-%: build-image-$$* require-docker
	@docker push ${IMAGE_ORG}/telemetry-$*:${VERSION}
	@echo pushed ${IMAGE_ORG}/telemetry-$*:${VERSION}

kind-load-%: build-image-$$*
	kind load docker-image ${IMAGE_ORG}/telemetry-$*:${VERSION} --name=${KIND_CLUSTER}

build-image-test: require-docker
	@mkdir -p bin/image/test
	@cp -a config/dockerfiles/test.Dockerfile bin/image/test/Dockerfile
	@cp -a .pre-commit-config.yaml bin/image/test
	@cp -a go.mod go.sum bin/image/test
	@cp -a hack/start-docker.sh bin/image/test
	@docker build -t ${IMAGE_ORG}/telemetry-test bin/image/test

push-image-test: build-image-test
	@docker push ${IMAGE_ORG}/telemetry-test
	@echo pushed ${IMAGE_ORG}/telemetry-test

build-image-dev: require-docker
	@mkdir -p bin/image/dev
	@cp -a config/dockerfiles/dev.Dockerfile bin/image/dev/Dockerfile
	@docker build -t ${IMAGE_ORG}/telemetry-dev:${DEV_IMAGE_TAG} bin/image/dev \
		--build-arg CONTROLLER_GEN_VERSION=${CONTROLLER_GEN_VERSION} \
		--build-arg APISERVER_BUILDER_VERSION=${APISERVER_BUILDER_VERSION} \
		--build-arg PROTOBUF_VERSION=${PROTOBUF_VERSION}

push-image-dev: build-image-dev
	@docker push ${IMAGE_ORG}/telemetry-dev:${DEV_IMAGE_TAG}
	@echo pushed ${IMAGE_ORG}/telemetry-dev:${DEV_IMAGE_TAG}

# -------
# Cleanup
# -------
clean:
	@rm -rf bin/$*
	@kind delete cluster --name=${KIND_CLUSTER}
.PHONY: clean
