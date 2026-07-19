PACKAGE_IMAGE_TARGET := package-image
TARGETS := $(filter-out $(PACKAGE_IMAGE_TARGET),$(shell ls scripts))
DAPPER_IMAGE ?= pasturestack-node-agent-dapper:ubuntu26
DAPPER_SOURCE ?= /go/src/github.com/PastureStack/node-agent

.dapper-image: Dockerfile.dapper
	docker build \
		--network "$${DOCKER_BUILD_NETWORK:-host}" \
		--build-arg DAPPER_HOST_ARCH=$${DAPPER_HOST_ARCH:-amd64} \
		-t $(DAPPER_IMAGE) \
		-f Dockerfile.dapper .

$(TARGETS): .dapper-image
	docker run --rm \
		--privileged \
		--cgroupns=host \
		-v $(CURDIR):$(DAPPER_SOURCE) \
		-e DAPPER_UID=$$(id -u) \
		-e DAPPER_GID=$$(id -g) \
		-e ARCH=$${ARCH:-amd64} \
		-e DOCKER_BUILD_NETWORK \
		-e TAG \
		-e REPO \
		-e CROSS \
		-e VERSION_OVERRIDE \
		-e NO_TEST \
		-e TEST_BUSYBOX_IMAGE \
		-e TEST_BUSYBOX_ALIAS \
		$(DAPPER_IMAGE) $@

$(PACKAGE_IMAGE_TARGET): .dapper-image
	docker run --rm \
		-v $(CURDIR):$(DAPPER_SOURCE) \
		-v /var/run/docker.sock:/var/run/docker.sock \
		-e DAPPER_UID=$$(id -u) \
		-e DAPPER_GID=$$(id -g) \
		-e ARCH=$${ARCH:-amd64} \
		-e DOCKER_BUILD_NETWORK \
		-e TAG \
		-e IMAGE_NAME \
		-e VERSION_OVERRIDE \
		$(DAPPER_IMAGE) $@

deps:
	@echo "Dependencies are vendored; no external dependency fetch is required."

.DEFAULT_GOAL := ci

.PHONY: $(TARGETS) deps
