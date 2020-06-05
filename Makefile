# Copyright 2020 PingCAP, Inc.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# See the License for the specific language governing permissions and
# limitations under the License.

GO := go

ARCH ?= $(shell ${GO} env GOARCH)
OS ?= $(shell ${GO} env GOOS)
IMAGE_REPO ?= localhost:5000/tikv
IMAGE_TAG ?= latest

ALL_TARGETS := cmd/tikv-controller-manager cmd/pd-discovery
GIT_VERSION = $(shell ./hack/version.sh | awk -F': ' '/^GIT_VERSION:/ {print $$2}')

ifneq ($(VERSION),)
	LDFLAGS += -X k8s.io/component-base/version.gitVersion=${VERSION}
else
	LDFLAGS += -X k8s.io/component-base/version.gitVersion=${GIT_VERSION}
endif

all: build
.PHONY: all

verify: 
	./hack/verify-all.sh
.PHONY: verify

build: $(ALL_TARGETS)
.PHONY: all

$(ALL_TARGETS): GOOS = $(OS)
$(ALL_TARGETS): GOARCH = $(ARCH)
$(ALL_TARGETS):
	CGO_ENABLED=0 $(GO) build -ldflags "${LDFLAGS}" -o output/bin/$(GOOS)/$(GOARCH)/$@ ./$@
.PHONY: $(ALL_TARGETS)

test:
	${GO} test ./cmd/... ./pkg/...
.PHONY: test

# OS/ARCH for binary in image is hardcoded to linux/amd64
image: GOOS = linux
image: GOARCH = amd64
image: build
	docker build -t "${IMAGE_REPO}/tikv-operator:${IMAGE_TAG}" .
.PHONY: image

e2e-examples:
	hack/e2e-examples.sh
.PHONY: e2e-examples
