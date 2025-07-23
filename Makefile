-include dev.env

## Set all the environment variables here
# Docker Registry
DOCKER_REGISTRY ?= docker.io/rocm

# Build Container environment
DOCKER_BUILDER_TAG ?= v1.0
BUILD_BASE_IMAGE ?= ubuntu:22.04
CUR_USER:=$(shell whoami)
CUR_TIME:=$(shell date +%Y-%m-%d_%H.%M.%S)
CONTAINER_NAME:=${CUR_USER}_dcm-bld
BUILD_CONTAINER ?= $(DOCKER_REGISTRY)/device-config-manager-build:$(DOCKER_BUILDER_TAG)
CONTAINER_WORKDIR := /usr/src/github.com/ROCm/device-config-manager

# Dcm container environment
DCM_IMAGE_TAG ?= latest
DCM_IMAGE_NAME ?= device-config-manager
RHEL_BASE_MIN_IMAGE ?= registry.access.redhat.com/ubi9/ubi-minimal:9.4
BUILD_DATE ?= $(shell date   +%Y-%m-%dT%H:%M:%S%z)
GIT_COMMIT ?= $(shell git rev-list -1 HEAD --abbrev-commit)
VERSION ?=$(RELEASE)

RHEL_BASE_IMAGE ?= registry.access.redhat.com/ubi9/ubi:9.4
# RHEL BaseOS, AppStream, CRB repository base image
RHEL_REPO_URL ?= https://cdn.redhat.com

# export environment variables used across project
export DOCKER_REGISTRY
export BUILD_CONTAINER
export BUILD_BASE_IMAGE
export DCM_IMAGE_NAME
export DCM_IMAGE_TAG
export RHEL_BASE_IMAGE
export RHEL_BASE_MIN_IMAGE
export RHEL_REPO_URL

TOP_DIR := $(PWD)
HELM_CHARTS_DIR := $(TOP_DIR)/helm-charts

RHEL_VERSION = rhel
RHEL_LIBDIR = RHEL9
# 22.04 - jammy
# 24.04 - noble
UBUNTU_VERSION ?= jammy
UBUNTU_VERSION_NUMBER = 22.04
UBUNTU_LIBDIR = UBUNTU22
ifeq (${UBUNTU_VERSION}, noble)
UBUNTU_VERSION_NUMBER = 24.04
UBUNTU_LIBDIR = UBUNTU24
endif

# External repo builders
AMDSMI_BUILDER_IMAGE ?= ${DOCKER_REGISTRY}/amdsmi-builder-dcm:rhel9
AMDSMI_BUILDER_UB22_IMAGE ?= ${DOCKER_REGISTRY}/amdsmi-builder-dcm:ub22
AMDSMI_BUILDER_UB24_IMAGE ?= ${DOCKER_REGISTRY}/amdsmi-builder-dcm:ub24
AMDSMI_BUILDER_AZURE_IMAGE ?= ${DOCKER_REGISTRY}/amdsmi-builder-dcm:azure

#Builder images can be built using targets make amdsmi-build-ub22, 24 etc
#Push them to internal registry as changes are needed
#These builder images are used rather than BASE images to prevent building the same image redundantly
AMDSMI_BUILDER_IMAGE ?= amdsmi-builder-dcm:rhel9
AMDSMI_BUILDER_UB22_IMAGE ?= amdsmi-builder-dcm:ub22
AMDSMI_BUILDER_UB24_IMAGE ?= amdsmi-builder-dcm:ub24
AMDSMI_BUILDER_AZURE_IMAGE ?= amdsmi-builder-dcm:azure

# amdsmi builder base images and tags
export AMDSMI_BASE_IMAGE
export AMDSMI_BASE_UBUNTU22
export AMDSMI_BASE_UBUNTU24
export AMDSMI_BASE_AZURE

# AMD SMI builder base images and tags
export AMDSMI_BUILDER_IMAGE
export AMDSMI_BUILDER_UB22_IMAGE
export AMDSMI_BUILDER_UB24_IMAGE
export AMDSMI_BUILDER_AZURE_IMAGE

# docs build settings
DOCS_DIR := ${TOP_DIR}/docs
BUILD_DIR := $(DOCS_DIR)/_build
HTML_DIR := $(BUILD_DIR)/html

# library branch to build amdsmi libraries
AMDSMI_BRANCH ?= amd-mainline
AMDSMI_COMMIT ?= 61ea0f2fb86b337d0efaef4337e95bc24df2a599

EXCLUDE_PATTERN := "libamdsmi"
GO_PKG := $(shell go list ./...  2>/dev/null | grep github.com/ROCm/device-config-manager | egrep -v ${EXCLUDE_PATTERN})

export ${AMDSMI_BRANCH}
export ${AMDSMI_COMMIT}

include Makefile.build
include Makefile.compile

##################
# Makefile targets
#
##@ QuickStart
.PHONY: default
default: build-dev-container ## Quick start to build everything from docker shell container
	${MAKE} docker-compile

.PHONY: docker-shell
docker-shell:
	docker run --rm -it --privileged \
		--name ${CONTAINER_NAME} \
		-e "USER_NAME=$(shell whoami)" \
		-e "USER_UID=$(shell id -u)" \
		-e "USER_GID=$(shell id -g)" \
		-e "GIT_COMMIT=${GIT_COMMIT}" \
		-e "GIT_VERSION=${GIT_VERSION}" \
		-e "BUILD_DATE=${BUILD_DATE}" \
		-v $(CURDIR):$(CONTAINER_WORKDIR) \
		-v $(HOME)/.ssh:/home/$(shell whoami)/.ssh \
		-w $(CONTAINER_WORKDIR) \
		$(BUILD_CONTAINER) \
		bash -c "cd $(CONTAINER_WORKDIR) && git config --global --add safe.directory $(CONTAINER_WORKDIR) && bash"

.PHONY: docker-compile
docker-compile:
	docker run --rm -it --privileged \
		--name ${CONTAINER_NAME} \
		-e "USER_NAME=$(shell whoami)" \
		-e "USER_UID=$(shell id -u)" \
		-e "USER_GID=$(shell id -g)" \
		-e "GIT_COMMIT=${GIT_COMMIT}" \
		-e "GIT_VERSION=${GIT_VERSION}" \
		-e "BUILD_DATE=${BUILD_DATE}" \
		-v $(CURDIR):$(CONTAINER_WORKDIR) \
		-v $(HOME)/.ssh:/home/$(shell whoami)/.ssh \
		-w $(CONTAINER_WORKDIR) \
		$(BUILD_CONTAINER) \
		bash -c "cd $(CONTAINER_WORKDIR) && source ~/.bashrc && git config --global --add safe.directory $(CONTAINER_WORKDIR) && make amdsmi-build-rhel && make all"

# create development build container only if there is changes done on
# tools/base-image/Dockerfile
.PHONY: build-dev-container
build-dev-container:
	${MAKE} -C tools/base-image all INSECURE_REGISTRY=$(INSECURE_REGISTRY)

.PHONY:clean
clean:
	rm -rf pkg/configmanager/bin

.PHONY: dcm
dcm:
	${MAKE} -C cmd/deviceconfigmanager build run ARGS="-k" TOP_DIR=$(TOP_DIR) UBUNTU_VERSION=$(RHEL_VERSION) UBUNTU_LIBDIR=$(RHEL_LIBDIR) GIT_COMMIT=$(GIT_COMMIT) VERSION=$(VERSION) BUILD_DATE=$(BUILD_DATE)

# debian case supported for the next release
.PHONY: dcm-st
dcm-st:
	${MAKE} -C cmd/deviceconfigmanager build-st run-st ARGS="-d" TOP_DIR=$(TOP_DIR) UBUNTU_VERSION=$(UBUNTU_VERSION) UBUNTU_LIBDIR=$(UBUNTU_LIBDIR) GIT_COMMIT=$(GIT_COMMIT) VERSION=$(VERSION) BUILD_DATE=$(BUILD_DATE)

.PHONY: dcm-docker
dcm-docker:
	${MAKE} -C docker TOP_DIR=$(TOP_DIR) UBUNTU_VERSION=$(RHEL_VERSION) UBUNTU_LIBDIR=$(RHEL_LIBDIR)

.PHONY: docker-publish
docker-publish:
	${MAKE} -C docker docker-publish TOP_DIR=$(TOP_DIR) UBUNTU_VERSION=$(RHEL_VERSION) UBUNTU_LIBDIR=$(RHEL_LIBDIR)

.PHONY:all
all:
	${MAKE} dcm
	${MAKE} dcm-docker

copyrights:
	GOFLAGS=-mod=mod go run tools/build/copyright/main.go && ${MAKE} fmt && ./tools/build/check-local-files.sh

.PHONY: helm-lint
helm-lint:
	cd $(HELM_CHARTS_DIR); helm lint

.PHONY: helm-build
helm-build: helm-lint
	helm package helm-charts/ --destination ./helm-charts

.PHONY: helm-install
helm-install: helm-build
	cd $(HELM_CHARTS_DIR); helm install amd-gpu-operator ./device-config-manager-charts-v1.3.1.tgz -n kube-amd-gpu --create-namespace -f values.yaml

.PHONY: helm-uninstall
helm-uninstall:
	helm uninstall amd-gpu-operator -n kube-amd-gpu

.PHONY: helm-list
helm-list:
	helm list --all-namespaces

GOLANGCI_LINT = $(shell pwd)/bin/golangci-lint
.PHONY: golangci-lint
golangci-lint: ## Download golangci-lint locally if necessary.
	$(call go-get-tool,$(GOLANGCI_LINT),github.com/golangci/golangci-lint/cmd/golangci-lint@v1.53.1)

# go-get-tool will 'go install' any package $2 and install it to $1.
PROJECT_DIR := $(shell dirname $(abspath $(lastword $(MAKEFILE_LIST))))
define go-get-tool
@[ -f $(1) ] || { \
set -e ;\
echo "Downloading $(2)" ;\
GOBIN=$(PROJECT_DIR)/bin go install $(2) ;\
}
endef

GOFILES_NO_VENDOR = $(shell find . -type f -name '*.go' -not -path "./vendor/*")
.PHONY: lint
lint: golangci-lint ## Run golangci-lint against code.
	@if [ `gofmt -l $(GOFILES_NO_VENDOR) | wc -l` -ne 0 ]; then \
		echo There are some malformed files, please make sure to run \'make fmt\'; \
		gofmt -l $(GOFILES_NO_VENDOR); \
		exit 1; \
	fi
	$(GOLANGCI_LINT) run -v --timeout 5m0s

.PHONY: fmt
fmt: ## Run go fmt against code.
	go fmt $(GO_PKG)

.PHONY: vet
vet: ## Run go vet against code.
	$(info +++ govet sources)
	go vet -source $(GO_PKG)

.PHONY:loadgpu
loadgpu:
	sudo modprobe amdgpu

.PHONY:mod
mod:
	@echo "setting up go mod packages"
	@go mod tidy
	@go mod vendor

.PHONY:checks
checks: fmt

.PHONY: e2e
e2e:
	${MAKE} -C test/k8s-e2e all TOP_DIR=$(TOP_DIR)

.PHONY: update-submodules
update-submodules:
	git submodule update --remote --recursive

.PHONY: build-all
build-all: 
	${MAKE} amdsmi-compile-rhel amdsmi-compile-ub22 amdsmi-compile-ub24 amdsmi-compile-azure
	@echo "Docker image build is available under docker/ directory"

.PHONY: docs clean-docs dep-docs
dep-docs:
	pip install -r $(DOCS_DIR)/sphinx/requirements.txt

docs: dep-docs
	sphinx-build -b html $(DOCS_DIR) $(HTML_DIR)
	@echo "Docs built at $(HTML_DIR)/index.html"

clean-docs:
	rm -rf $(BUILD_DIR)

.PHONY: gopkglist
gopkglist:
	go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.34.2
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.5.1
	go install go.uber.org/mock/mockgen@v0.5.0
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.64.8
	go install golang.org/x/tools/cmd/goimports@latest
	go install github.com/alta/protopatch/cmd/protoc-gen-go-patch@latest

.PHONY: gen
gen: gopkglist
	${MAKE} -C proto/ all
