################################################################################
##                             VERSION PARAMS                                 ##
################################################################################

## Docker Build Versions
DOCKER_BUILD_IMAGE = golang:1.22.8
DOCKER_BASE_IMAGE = alpine:3.20

################################################################################

GO ?= $(shell command -v go 2> /dev/null)
CHEWBACCA_IMAGE ?= mattermost/chewbacca-bot:test
CHEWBACCA_IMAGE_REPO ?= mattermost/chewbacca-bot
MACHINE = $(shell uname -m)
GOFLAGS ?= $(GOFLAGS:)
BUILD_TIME := $(shell date -u +%Y%m%d.%H%M%S)
ARCH ?= amd64

################################################################################

TOOLS_BIN_DIR := $(abspath bin)

ENSURE_GOLANGCI_LINT = ./scripts/ensure_golangci-lint.sh

GOLANGCILINT_VER := v1.53.3
GOLANGCILINT_BIN := golangci-lint
GOLANGCILINT := $(TOOLS_BIN_DIR)/$(GOLANGCILINT_BIN)

################################################################################

export GO111MODULE=on

## Checks the code style, tests, builds and bundles.
all: check-style dist

## Runs govet and gofmt against all packages.
.PHONY: check-style
check-style: govet lint
	@echo Checking for style guide compliance

## Runs lint against all packages.
lint: $(GOPATH)/bin/golangci-lint
	@echo Running golangci-lint
	golangci-lint run ./...
	@echo lint success

## Runs lint against all packages for changes only
lint-changes: $(GOPATH)/bin/golangci-lint
	@echo Running golangci-lint over changes only
	golangci-lint run -n
	@echo lint success


## Runs govet against all packages.
.PHONY: vet
govet:
	@echo Running govet
	$(GO) vet ./...
	@echo Govet success

## Builds and that's all :)
.PHONY: dist
dist:	build

.PHONY: build
build: ## Build the Chewbacca
	@echo Building Chewbacca for ARCH=$(ARCH)
	@if [ "$(ARCH)" = "amd64" ]; then \
		export GOARCH="amd64"; \
	elif [ "$(ARCH)" = "arm64" ]; then \
		export GOARCH="arm64"; \
	elif [ "$(ARCH)" = "arm" ]; then \
		export GOARCH="arm"; \
	else \
		echo "Unknown architecture $(ARCH)"; \
		exit 1; \
	fi; \
	GOOS=linux CGO_ENABLED=0 $(GO) build -gcflags -buildvcs=false all=-trimpath=$(PWD) -asmflags all=-trimpath=$(PWD) -a -installsuffix cgo -o build/chewbacca ./cmd

.PHONY: build-image
build-image:  ## Build the image for Chewbacca
	@echo Building Chewbacca Image
	@if [ -z "$(DOCKER_USERNAME)" ] || [ -z "$(DOCKER_PASSWORD)" ]; then \
		echo "DOCKER_USERNAME and/or DOCKER_PASSWORD not set. Skipping Docker login."; \
	else \
		echo $(DOCKER_PASSWORD) | docker login --username $(DOCKER_USERNAME) --password-stdin; \
	fi
	docker buildx build \
    --platform linux/arm64,linux/amd64 \
	--build-arg DOCKER_BUILD_IMAGE=$(DOCKER_BUILD_IMAGE) \
	--build-arg DOCKER_BASE_IMAGE=$(DOCKER_BASE_IMAGE) \
	. -f Dockerfile -t $(CHEWBACCA_IMAGE) \
	--no-cache \
	--push

.PHONY: build-image-with-tag
build-image-with-tag:  ## Build the image for Chewbacca
	@echo Building Chewbacca Image
	@if [ -z "$(DOCKER_USERNAME)" ] || [ -z "$(DOCKER_PASSWORD)" ]; then \
		echo "DOCKER_USERNAME and/or DOCKER_PASSWORD not set. Skipping Docker login."; \
	else \
		echo $(DOCKER_PASSWORD) | docker login --username $(DOCKER_USERNAME) --password-stdin; \
	fi
	docker buildx build \
    --platform linux/arm64,linux/amd64 \
	--build-arg DOCKER_BUILD_IMAGE=$(DOCKER_BUILD_IMAGE) \
	--build-arg DOCKER_BASE_IMAGE=$(DOCKER_BASE_IMAGE) \
	. -f Dockerfile -t $(CHEWBACCA_IMAGE) -t $(CHEWBACCA_IMAGE_REPO):${TAG} \
	--no-cache \
	--push

.PHONY: build-image-locally
build-image-locally:  ## Build the image for Chewbacca
	@echo Building Chewbacca Image
	@if [ -z "$(DOCKER_USERNAME)" ] || [ -z "$(DOCKER_PASSWORD)" ]; then \
		echo "DOCKER_USERNAME and/or DOCKER_PASSWORD not set. Skipping Docker login."; \
	else \
		echo $(DOCKER_PASSWORD) | docker login --username $(DOCKER_USERNAME) --password-stdin; \
	fi
	docker buildx build \
    --platform linux/arm64 \
	--build-arg DOCKER_BUILD_IMAGE=$(DOCKER_BUILD_IMAGE) \
	--build-arg DOCKER_BASE_IMAGE=$(DOCKER_BASE_IMAGE) \
	. -f Dockerfile -t $(CHEWBACCA_IMAGE) \
	--no-cache \
	--load

.PHONY: push-image-pr
push-image-pr:
	@echo Push Image PR
	./scripts/push-image-pr.sh

.PHONY: push-image
push-image:
	@echo Push Image
	./scripts/push-image.sh

.PHONY: install
install: build
	go install ./...

.PHONY: test
test:
	go test ./... -v

## --------------------------------------
## Tooling Binaries
## --------------------------------------

$(GOPATH)/bin/golangci-lint: ## Install golangci-lint
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@$(GOLANGCILINT_VER)
