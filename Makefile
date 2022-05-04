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

export CGO_ENABLED = 0
export GOFLAGS ?= -mod=readonly -trimpath
SHELL = /bin/bash
CMD ?= $(notdir $(wildcard ./cmd/*))
GOOS ?= $(shell go env GOOS)
GOBUILDFLAGS ?= -v
LDFLAGS += -extldflags '-static'
LDFLAGS_EXTRA=-w
BUILD_DEST ?= _build
GOTOOLFLAGS ?= $(GOBUILDFLAGS) -ldflags '$(LDFLAGS_EXTRA) $(LDFLAGS)' $(GOTOOLFLAGS_EXTRA)

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
	golangci-lint run --verbose ./...
