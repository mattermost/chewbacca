################################################################################
##                             VERSION PARAMS                                 ##
################################################################################

## Docker Build Versions
DOCKER_BUILD_IMAGE = golang:1.22.8
DOCKER_BASE_IMAGE = alpine:3.19

################################################################################

GO ?= $(shell command -v go 2> /dev/null)
CHEWBACCA_IMAGE ?= mattermost/chewbacca-bot:test
MACHINE = $(shell uname -m)
GOFLAGS ?= $(GOFLAGS:)
BUILD_TIME := $(shell date -u +%Y%m%d.%H%M%S)

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
.PHONY: lint
lint: $(GOLANGCILINT)
	@echo Running golangci-lint
	$(GOLANGCILINT) run
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
	@echo Building Chewbacca
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 $(GO) build -gcflags all=-trimpath=$(PWD) -asmflags all=-trimpath=$(PWD) -a -installsuffix cgo -o build/chewbacca ./cmd

build-image:  ## Build the image for Chewbacca
	@echo Building Chewbacca Image
	docker build \
	--build-arg DOCKER_BUILD_IMAGE=$(DOCKER_BUILD_IMAGE) \
	--build-arg DOCKER_BASE_IMAGE=$(DOCKER_BASE_IMAGE) \
	. -f Dockerfile -t $(CHEWBACCA_IMAGE) \
	--no-cache

.PHONY: install
install: build
	go install ./...

.PHONY: test
test:
	go test ./... -v

$(GOLANGCILINT): ## Build golangci-lint
	BINDIR=$(TOOLS_BIN_DIR) TAG=$(GOLANGCILINT_VER) $(ENSURE_GOLANGCI_LINT)