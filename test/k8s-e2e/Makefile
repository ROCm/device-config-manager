.DEFAULT: all
.PHONY: all lint

TEST_ARGS = ""
ifdef OPENSHIFT
	TEST_ARGS += "-openshift"
endif

ifdef SIM_ENABLE
	TEST_ARGS += "-simEnable"
endif

ifdef CI_ENV
	TEST_ARGS += "-ciEnv"
endif

lint:
	@go fmt ./...
	@goimports -w ./
	@go vet ./...

all:
	@TOP_DIR=$(TOP_DIR) go test -failfast -helmchart $(TOP_DIR)/helm-charts/ -test.timeout=30m -v $(TEST_ARGS); 
