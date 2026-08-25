-include dev.env

## Set all the environment variables here
# Docker Registry
DOCKER_REGISTRY ?= docker.io/rocm

# helm environment variables
HELM_DCM_IMAGE := $(DOCKER_REGISTRY)/device-config-manager

# Build Container environment
BUILD_BASE_IMAGE ?= ubuntu:22.04
CUR_USER:=$(shell whoami)
CUR_TIME:=$(shell date +%Y-%m-%d_%H.%M.%S)
CONTAINER_NAME:=${CUR_USER}_dcm-bld

# Dcm container environment
DCM_IMAGE_TAG ?= latest
DCM_IMAGE_NAME ?= device-config-manager
# Label stamped on the runtime image; CI passes HOURLY_TAG_LABEL=$(RELEASE).
HOURLY_TAG_LABEL ?= latest
RHEL_BASE_MIN_IMAGE ?= registry.access.redhat.com/ubi9/ubi-minimal:9.8
BUILD_DATE ?= $(shell date   +%Y-%m-%dT%H:%M:%S%z)
GIT_COMMIT ?= $(shell git rev-list -1 HEAD --abbrev-commit)
VERSION ?=$(RELEASE)

RHEL_BASE_IMAGE ?= registry.access.redhat.com/ubi9/ubi:9.8
# RHEL BaseOS, AppStream, CRB repository base image
RHEL_REPO_URL ?= https://cdn.redhat.com

# export environment variables used across project
export DOCKER_REGISTRY
export BUILD_BASE_IMAGE
export DCM_IMAGE_NAME
export DCM_IMAGE_TAG
export RHEL_BASE_IMAGE
export RHEL_BASE_MIN_IMAGE
export RHEL_REPO_URL
export REGISTRY

TOP_DIR := $(PWD)
HELM_CHARTS_DIR := $(TOP_DIR)/helm-charts
PKG_LIB_PATH := ${TOP_DIR}/debian/usr/local/configs/
PKG_CONFIG_PATH := ${TOP_DIR}/debian/etc/dcm/
ASSETS_PATH :=${TOP_DIR}/assets

# Canonical build target selector. One of: rhel9 (default) | ub22 | ub24.
DISTRO ?= rhel9

# DISTRO -> asset libdir / ubuntu version number / ubuntu codename
ifeq ($(DISTRO),rhel9)
  LIBDIR = RHEL9
  UBUNTU_VERSION_NUMBER =
  CODENAME =
else ifeq ($(DISTRO),ub22)
  LIBDIR = UBUNTU22
  UBUNTU_VERSION_NUMBER = 22.04
  CODENAME = jammy
else ifeq ($(DISTRO),ub24)
  LIBDIR = UBUNTU24
  UBUNTU_VERSION_NUMBER = 24.04
  CODENAME = noble
else
  $(error unsupported DISTRO '$(DISTRO)' - use rhel9, ub22, or ub24)
endif

# Thin build image for the cgo go build. Public default; override GOLANG_IMAGE
# in dev.env for a mirror.
GOLANG_IMAGE ?= golang:1.25-bookworm
DCM_BUILD_IMAGE ?= dcm-build:local

# Use the legacy builder for image builds. BuildKit ignores the daemon's
# insecure-registries setting, so pulling the mirrored golang/ubi bases over
# plain HTTP (as in the box CI environment) fails with "server gave HTTP
# response to HTTPS client". The legacy builder honors insecure-registries.
DOCKER_BUILD ?= DOCKER_BUILDKIT=0 docker build

ifneq (,$(findstring collab-,$(RELEASE)))
#remove collab- prefix from tag
DEBIAN_VERSION := $(shell echo "$(RELEASE)" | sed 's/^collab-//')
else ifneq (,$(findstring dcm-,$(RELEASE)))
#remove dcm-v prefix from tag
DEBIAN_VERSION := $(shell echo "$(RELEASE)" | sed 's/^dcm-v//')
else ifneq (,$(findstring v,$(RELEASE)))
#remove prefix for release tag
DEBIAN_VERSION := $(shell echo "$(RELEASE)" | sed 's/^v//')
else
DEBIAN_VERSION := 1.0.0
endif

BUILD_PKG_PATH = ${TOP_DIR}/build/${LIBDIR}
DEBIAN_CONTROL = ${TOP_DIR}/debian/DEBIAN/control
BUILD_VER_ENV = ${DEBIAN_VERSION}~$(UBUNTU_VERSION_NUMBER)

# Staged amdsmi libs for packaging — populated by .stage-amdsmi (tarball or, with
# AMDSMI_FROM_TARBALL=0, regenerated committed assets). No longer read from the
# committed assets/ tree, which was removed.
AMD_SMI_LIBS := ${TOP_DIR}/build/assets
PKG_PATH := ${TOP_DIR}/debian/usr/local/bin
PROTOS_PATH := $(TOP_DIR)/proto

# External repo builders
AMDSMI_BASE_IMAGE ?= registry.access.redhat.com/ubi9/ubi:9.4
AMDSMI_BASE_UBUNTU22 ?= ubuntu:22.04
AMDSMI_BASE_UBUNTU24 ?= ubuntu:24.04
AMDSMi_BASE_AZURE ?= mcr.microsoft.com/azurelinux/base/core:3.0

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
DOCS_MARKDOWNLINTCONFIG ?= docs/.markdownlint.yaml
DOCS_MD_GLOB ?= "**/*.md"
DOCS_SPELLCHECK_CONFIG ?= .spellcheck.yaml

# library branch to build amdsmi libraries
AMDSMI_REPO   ?= https://github.com/ROCm/rocm-systems.git
AMDSMI_BRANCH ?= release/therock-10.0
AMDSMI_COMMIT ?= 6ccde5dbc2714a95f4fa92a7f2d3aa185ed2174b
AMDSMI_SUBDIR ?= projects/amdsmi
PROJECT_VERSION ?= "1.5.2"

EXCLUDE_PATTERN := "libamdsmi"
GO_PKG := $(shell go list ./...  2>/dev/null | grep github.com/ROCm/device-config-manager | egrep -v ${EXCLUDE_PATTERN})

ROCM_TARBALL_URL ?= https://rocm.prereleases.amd.com/tarball-multi-arch/therock-dist-linux-multiarch-10.0.0rc4.tar.gz
ROCM_VERSION ?= 10.0.0rc4

# 1 = extract amdsmi.h + libamd_smi.so + rocm_sysdeps from the therock tarball
#     (ROCM_TARBALL_URL) at build time (default; committed assets were removed).
# 0 = stage from committed assets/amd_smi_lib libs — these are no longer checked
#     in, so `make amdsmi-update` must regenerate them first.
AMDSMI_FROM_TARBALL ?= 1

export AMDSMI_REPO
export AMDSMI_BRANCH
export AMDSMI_COMMIT
export AMDSMI_SUBDIR
export ROCM_TARBALL_URL
export ROCM_VERSION
export AMDSMI_FROM_TARBALL

include Makefile.build
include Makefile.compile

##################
# Makefile targets
#
##@ QuickStart
.PHONY: default
default: all ## Quick start: build binary + runtime image (Docker only)

.PHONY:clean
clean:
	rm -rf pkg/configmanager/bin
	rm -r $(TOP_DIR)/build/assets

# Unified DCM Build Targets
# Selector: DISTRO=rhel9 (default) | ub22 | ub24

.PHONY: .dcm-build-image
.dcm-build-image:
	@$(DOCKER_BUILD) -f docker/Dockerfile.build \
		--build-arg GOLANG_IMAGE=$(GOLANG_IMAGE) \
		-t $(DCM_BUILD_IMAGE) docker/ >/dev/null

# Stage amdsmi (libamd_smi.so + rocm_sysdeps + amdsmi.h) into build/assets/ for the
# cgo build. AMDSMI_FROM_TARBALL=0 copies the committed assets/ libs (default);
# =1 extracts them from the therock tarball (ROCM_TARBALL_URL) at build time.
.PHONY: .stage-amdsmi
.stage-amdsmi:
	@rm -rf build/assets && mkdir -p build/assets/amd_smi
ifeq ($(AMDSMI_FROM_TARBALL),1)
	@echo "Staging amdsmi from tarball $(ROCM_TARBALL_URL)"
	@rm -rf build/smi && mkdir -p build/smi
	@curl -fSL "$(ROCM_TARBALL_URL)" | tar -xz -C build/smi --wildcards --no-anchored \
		'amdsmi.h' 'libamd_smi.so*' 'librocm_sysdeps_*.so*'
	@cp -a build/smi/lib/libamd_smi.so* build/assets/
	@cp -a build/smi/lib/rocm_sysdeps/lib/librocm_sysdeps_*.so* build/assets/
	@cp -a build/smi/include/amd_smi/amdsmi.h build/assets/amd_smi/
	@# recreate the libdrm/libdrm_amdgpu linker symlinks the cgo LDFLAGS
	@# (-ldrm -ldrm_amdgpu) expect; the tarball ships them only as renamed
	@# librocm_sysdeps_* reals.
	@cd build/assets && \
		ln -sf librocm_sysdeps_drm.so.2 libdrm.so.2 && ln -sf libdrm.so.2 libdrm.so && \
		ln -sf librocm_sysdeps_drm_amdgpu.so.1 libdrm_amdgpu.so.1 && ln -sf libdrm_amdgpu.so.1 libdrm_amdgpu.so
	@rm -rf build/smi
else
	@echo "Staging amdsmi from committed assets (LIBDIR=$(LIBDIR))"
	@if [ ! -f assets/amd_smi_lib/x86_64/$(LIBDIR)/lib/amdsmi.h ]; then \
		echo "ERROR: committed amdsmi assets were removed; run 'make amdsmi-update'"; \
		echo "       to regenerate them, or build with AMDSMI_FROM_TARBALL=1 (default)."; \
		exit 1; \
	fi
	@cp -r assets/amd_smi_lib/x86_64/$(LIBDIR)/lib/* build/assets/
	@cp assets/amd_smi_lib/x86_64/$(LIBDIR)/lib/amdsmi.h build/assets/amd_smi/
endif

.PHONY: dcm
dcm: .dcm-build-image .stage-amdsmi
	@echo "Building DCM binary (DISTRO=$(DISTRO), libs=$(LIBDIR))"
	@docker run --rm \
		-v $(CURDIR):$(CURDIR) -w $(CURDIR) \
		--user $$(id -u):$$(id -g) \
		-e HOME=$(CURDIR) \
		-e GOCACHE=$(CURDIR)/.cache/go \
		-e GOPATH=$(CURDIR)/.cache/gopath \
		-e CGO_ENABLED=1 -e GOOS=linux -e GOARCH=amd64 -e GOFLAGS=-mod=vendor \
		$(DCM_BUILD_IMAGE) \
		bash -c 'set -e; \
			mkdir -p bin; \
			go build -ldflags "-s -w -X main.Version=$(VERSION) -X main.GitCommit=$(GIT_COMMIT) -X main.BuildDate=$(BUILD_DATE)" \
				-o bin/device-config-manager-$(DISTRO) cmd/deviceconfigmanager/main.go'
	@echo "Built bin/device-config-manager-$(DISTRO)"

.PHONY: dcm-rhel9 dcm-ub22 dcm-ub24
dcm-rhel9:
	@$(MAKE) dcm DISTRO=rhel9
dcm-ub22:
	@$(MAKE) dcm DISTRO=ub22
dcm-ub24:
	@$(MAKE) dcm DISTRO=ub24

# The runtime image is always RHEL9 (ubi-minimal base). Force a rhel9 binary
# build so `dcm-docker DISTRO=ub22` can't embed an Ubuntu-targeted binary
# alongside the RHEL9 libs/UBI base.
.PHONY: dcm-docker
dcm-docker:
	@$(MAKE) dcm DISTRO=rhel9
	@echo "Building DCM runtime image $(HELM_DCM_IMAGE):$(DCM_IMAGE_TAG)"
	@rm -rf docker/smilib docker/device-config-manager
	@# stage the LD_PRELOADed libs from build/assets so they match whatever the
	@# cgo build linked against (committed assets or tarball, per AMDSMI_FROM_TARBALL).
	@cp -r build/assets docker/smilib
	@cp bin/device-config-manager-rhel9 docker/device-config-manager
	@$(DOCKER_BUILD) -f docker/Dockerfile \
		--build-arg RHEL_BASE_MIN_IMAGE=$(RHEL_BASE_MIN_IMAGE) \
		--build-arg ROCM_TARBALL_URL=$(ROCM_TARBALL_URL) \
		--build-arg ROCM_VERSION=$(ROCM_VERSION) \
		--build-arg AMDSMI_FROM_TARBALL=$(AMDSMI_FROM_TARBALL) \
		--label HOURLY_TAG=$(HOURLY_TAG_LABEL) \
		-t $(HELM_DCM_IMAGE):$(DCM_IMAGE_TAG) docker/
	@rm -rf docker/smilib docker/device-config-manager
	@rm -rf docker/obj && mkdir -p docker/obj
	@docker save $(HELM_DCM_IMAGE):$(DCM_IMAGE_TAG) | gzip > docker/obj/config-manager-ubi9-latest.tgz
	@echo "Built image $(HELM_DCM_IMAGE):$(DCM_IMAGE_TAG) and tarball docker/obj/config-manager-ubi9-latest.tgz"

.PHONY: docker-publish
docker-publish: dcm-docker
	@echo "Publishing DCM runtime image $(HELM_DCM_IMAGE):$(DCM_IMAGE_TAG)"
	@docker push $(HELM_DCM_IMAGE):$(DCM_IMAGE_TAG)

.PHONY: all
all: dcm dcm-docker

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
	cd $(HELM_CHARTS_DIR); helm install amd-gpu-operator ./device-config-manager-charts-v1.5.2.tgz -n kube-amd-gpu --create-namespace -f values.yaml

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

pkg-clean:
	rm -rf ${TOP_DIR}/bin/*.deb

.PHONY: pkg
pkg: pkg-clean
	@if [ "$(DISTRO)" = "rhel9" ]; then echo "pkg requires DISTRO=ub22 or ub24"; exit 1; fi
	${MAKE} dcm DISTRO=$(DISTRO)
	@echo "Building debian for $(BUILD_VER_ENV)"
	@echo "Build path ${BUILD_PKG_PATH}"
	#copy precompiled libs
	mkdir -p ${PKG_LIB_PATH}
	cp -rvf ${AMD_SMI_LIBS}/ ${PKG_LIB_PATH}
	mkdir -p ${PKG_PATH}
	cp -vf $(TOP_DIR)/bin/device-config-manager-$(DISTRO) ${PKG_PATH}/
	#strip the dcm gobin to reduce the debian package size
	strip ${PKG_PATH}/device-config-manager-$(DISTRO)
	cd ${TOP_DIR}
	sed -i "s/BUILD_VER_ENV/$(BUILD_VER_ENV)/g" $(DEBIAN_CONTROL)
	sed -i "s/UBUNTU_VERSION_PLACEHOLDER/$(DISTRO)/g" debian/usr/lib/systemd/system/amd-config-manager.service
	dpkg-deb -Zxz --build debian ${TOP_DIR}/bin
	#remove copied files
	rm -rf ${PKG_LIB_PATH}
	# revert the dynamic version set file
	git checkout $(DEBIAN_CONTROL)
	git checkout debian/usr/lib/systemd/system/amd-config-manager.service
	# rename for internal build
	mv -vf ${TOP_DIR}/bin/amdgpu-configmanager_*~${UBUNTU_VERSION_NUMBER}_amd64.deb ${TOP_DIR}/bin/amdgpu-configmanager_${UBUNTU_VERSION_NUMBER}_amd64.deb

.PHONY: pkg-ainic
pkg-ainic: 
	@echo "AINIC and GPU packages are now unified - using same 'pkg' target"
	@$(MAKE) pkg

.PHONY: pkg-jammy
pkg-jammy:
	@echo "Building unified DCM package for Ubuntu 22.04 (jammy)"
	@$(MAKE) pkg DISTRO=ub22

.PHONY: pkg-noble
pkg-noble:
	@echo "Building unified DCM package for Ubuntu 24.04 (noble)"
	@$(MAKE) pkg DISTRO=ub24

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
	@go mod edit -go=1.25.11
	@go mod vendor

.PHONY:checks
checks: fmt

.PHONY: e2e test-ainic
e2e:
	${MAKE} -C test/k8s-e2e all TOP_DIR=$(TOP_DIR)

test-ainic:
	${MAKE} -C test/k8s-e2e test-ainic TOP_DIR=$(TOP_DIR)

.PHONY: update-submodules
update-submodules:
	git submodule update --remote --recursive

.PHONY: build-amdsmi-all
build-amdsmi-all:
	${MAKE} amdsmi-build-rhel amdsmi-build-ub22 amdsmi-build-ub24 amdsmi-build-azure
	@echo "Docker image build is available under docker/ directory"

.PHONY: compile-amdsmi-all
compile-amdsmi-all:
	${MAKE} amdsmi-compile-rhel amdsmi-compile-ub22 amdsmi-compile-ub24 amdsmi-compile-azure

.PHONY: amdsmi-update
amdsmi-update:
	@echo "Updating amdsmi libs from branch $(AMDSMI_BRANCH) and commit id $(AMDSMI_COMMIT)"
	${MAKE} update-submodules build-amdsmi-all compile-amdsmi-all
	mkdir -p assets/amd_smi_lib/x86_64/AZURE3/lib assets/amd_smi_lib/x86_64/RHEL9/lib \
		assets/amd_smi_lib/x86_64/UBUNTU22/lib assets/amd_smi_lib/x86_64/UBUNTU24/lib
	cp build/assets/AZURE3/dcmout/* assets/amd_smi_lib/x86_64/AZURE3/lib/
	cp build/assets/RHEL9/dcmout/* assets/amd_smi_lib/x86_64/RHEL9/lib/
	cp build/assets/UBUNTU22/dcmout/* assets/amd_smi_lib/x86_64/UBUNTU22/lib/
	cp build/assets/UBUNTU24/dcmout/* assets/amd_smi_lib/x86_64/UBUNTU24/lib/

.PHONY: docs clean-docs dep-docs
dep-docs:
	pip install -r $(DOCS_DIR)/sphinx/requirements.txt

docs: dep-docs
	sphinx-build -b html $(DOCS_DIR) $(HTML_DIR)
	@echo "Docs built at $(HTML_DIR)/index.html"

clean-docs:
	rm -rf $(BUILD_DIR)

.PHONY: docs-lint-markdown
docs-lint-markdown:
	markdownlint-cli2 $(DOCS_MD_GLOB) --config $(DOCS_MARKDOWNLINTCONFIG)

.PHONY: docs-lint-spelling
docs-lint-spelling:
	pyspelling -c $(DOCS_SPELLCHECK_CONFIG)

.PHONY: docs-lint
docs-lint: ## Run docs Markdown lint + spelling (full ROCm-style docs lint).
	${MAKE} docs-lint-markdown
	${MAKE} docs-lint-spelling

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

.PHONY: gen-ainic-proto
gen-ainic-proto: gopkglist
	@echo "building ainic proto"
	@protoc --proto_path=proto --go-grpc_out=. --go_out=. proto/ainic.proto

.PHONY: gen-gpu-proto
gen-gpu-proto: gopkglist
	@echo "building gpu/partition proto"
	@protoc --proto_path=proto --go-grpc_out=. --go_out=. proto/partition.proto

.PHONY: copy-assets-k8s
copy-assets-k8s: .stage-amdsmi
	@echo "amdsmi staged into build/assets (AMDSMI_FROM_TARBALL=$(AMDSMI_FROM_TARBALL))"

# cicd target to build helm chart - requires PROJECT_VERSION, DCM_IMAGE_TAG to be set
.PHONY: helm
helm: helm-lint
	@rm -rf helm-charts-k8s
	@yq eval -i '.image.repository = "$(HELM_DCM_IMAGE)"' helm-charts/values.yaml
	@yq eval -i '.image.tag = "$(DCM_IMAGE_TAG)"' helm-charts/values.yaml
	@mkdir -p helm-charts-k8s
	helm package helm-charts/ --destination ./helm-charts-k8s --app-version ${PROJECT_VERSION} --version ${PROJECT_VERSION}
	cp -vf helm-charts-k8s/device-config-manager-*.tgz helm-charts-k8s/device-config-manager-helm-k8s-${PROJECT_VERSION}.tgz
