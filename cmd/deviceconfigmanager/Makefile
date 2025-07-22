$(info DOCKER_REGISTRY is $(DOCKER_REGISTRY))
DCM_BUILD_DOCKER_IMG := ${DOCKER_REGISTRY}/dcm-build
ifeq ($(UBUNTU_VERSION), jammy)
    BASE_IMAGE=$(AMDSMI_BUILDER_UB22_IMAGE)
else ifeq ($(UBUNTU_VERSION), noble)
    BASE_IMAGE=$(AMDSMI_BUILDER_UB24_IMAGE)
else ifeq ($(UBUNTU_VERSION), rhel)
    BASE_IMAGE=$(AMDSMI_BUILDER_IMAGE)
endif

.PHONY:
build:
	@echo "buidling amd config manager binary"
	@docker build --build-arg DCM_BASE_IMAGE=$(BASE_IMAGE) \
                  --build-arg UBUNTU_LIBDIR=$(UBUNTU_LIBDIR) \
				  --build-arg UBUNTU_VERSION=$(UBUNTU_VERSION) \
				  --build-arg GIT_COMMIT=$(GIT_COMMIT) \
				  --build-arg VERSION=$(VERSION) \
				  --build-arg BUILD_DATE=$(BUILD_DATE) \
				  -t $(DCM_BUILD_DOCKER_IMG):$(UBUNTU_VERSION) . -f Dockerfile

.PHONY:
build-st:
	@echo "buidling amd config manager binary"
	@docker build --build-arg DCM_BASE_IMAGE=$(BASE_IMAGE) \
                  --build-arg UBUNTU_LIBDIR=$(UBUNTU_LIBDIR) \
				  --build-arg UBUNTU_VERSION=$(UBUNTU_VERSION) \
				  --build-arg GIT_COMMIT=$(GIT_COMMIT) \
				  --build-arg VERSION=$(VERSION) \
				  --build-arg BUILD_DATE=$(BUILD_DATE) \
				  -t $(DCM_BUILD_DOCKER_IMG):$(UBUNTU_VERSION) . -f Dockerfile.standalone

.PHONY:
run:
	@echo "Running amd config manager docker"
	@docker run -e UBUNTU_LIBDIR=$(UBUNTU_LIBDIR) \
				-e UBUNTU_VERSION=$(UBUNTU_VERSION) \
				-e GIT_COMMIT=$(GIT_COMMIT) \
				-e VERSION=$(VERSION) \
				-e BUILD_DATE=$(BUILD_DATE) \
				-v $(TOP_DIR):/device-config-manager \
				-t $(DCM_BUILD_DOCKER_IMG):$(UBUNTU_VERSION) $(ARGS)

.PHONY:
run-st:
	@echo "Running amd config manager docker"
	@docker run -e UBUNTU_LIBDIR=$(UBUNTU_LIBDIR) \
				-e UBUNTU_VERSION=$(UBUNTU_VERSION) \
				-e GIT_COMMIT=$(GIT_COMMIT) \
				-e VERSION=$(VERSION) \
				-e BUILD_DATE=$(BUILD_DATE) \
				-v $(TOP_DIR):/device-config-manager \
				-t $(DCM_BUILD_DOCKER_IMG):$(UBUNTU_VERSION) $(ARGS)