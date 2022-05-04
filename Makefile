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

SHELL = /bin/bash
CMD ?= $(notdir $(wildcard ./cmd/*))

# Go-related variables
export CGO_ENABLED = 0
export GOFLAGS ?= -mod=readonly -trimpath
GOOS ?= $(shell go env GOOS)
GOBUILDFLAGS ?= -v
LDFLAGS += -extldflags '-static'
LDFLAGS_EXTRA=-w
BUILD_DEST ?= _build
GOTOOLFLAGS ?= $(GOBUILDFLAGS) -ldflags '$(LDFLAGS_EXTRA) $(LDFLAGS)' $(GOTOOLFLAGS_EXTRA)

# Docker-related variables
REPO = quay.io/kubermatic/telemetry-agent
TAGS ?= $(shell git describe --tags --always)
DOCKER_BUILD_FLAG += $(foreach tag, $(TAGS), -t $(REPO):$(tag))

# -----------------
# Compile
# -----------------
.PHONY: build
build: $(CMD)

.PHONY: $(CMD)
$(CMD): %: _build/%

_build/%: cmd/%
	GOOS=$(GOOS) go build $(GOTOOLFLAGS) -o $@ ./cmd/$*

# ---------------
# Code Generators
# ---------------
.PHONY: generate
generate:
	@hack/update-codegen.sh

.PHONY: docker-build
docker-build: build
	docker build $(DOCKER_BUILD_FLAG) .

# ------------
# Test Runners
# ------------
.PHONY: test
test:
	CGO_ENABLED=1 go test -race -v ./pkg/...

# -------------
# Util Commands
# -------------
.PHONY: clean
clean:
	@rm -rf _build/

.PHONY: tidy
tidy:
	go mod tidy

.PHONY: verify
verify:
	hack/verify-boilerplate.sh
	hack/verify-codegen.sh
	golangci-lint run --verbose ./...
