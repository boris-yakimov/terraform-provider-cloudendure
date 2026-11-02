BINARY    := terraform-provider-cloudendure
VERSION   ?= 0.1.0
NAMESPACE := boris-yakimov
NAME      := cloudendure
OS_ARCH   := $(shell go env GOOS)_$(shell go env GOARCH)
PLUGIN_DIR := $(HOME)/.terraform.d/plugins/registry.terraform.io/$(NAMESPACE)/$(NAME)/$(VERSION)/$(OS_ARCH)

LDFLAGS := -s -w -X main.version=$(VERSION)

.DEFAULT_GOAL := build

.PHONY: build
build: ## Build the provider binary
	go build -trimpath -ldflags "$(LDFLAGS)" -o $(BINARY)

.PHONY: install
install: build ## Install the provider into the local Terraform plugin directory
	mkdir -p $(PLUGIN_DIR)
	mv $(BINARY) $(PLUGIN_DIR)/

.PHONY: test
test: ## Run unit tests (uses an in-memory fake API and the local terraform CLI)
	go test -race -count=1 -timeout=10m ./...

.PHONY: lint
lint: ## Run golangci-lint
	golangci-lint run ./...

.PHONY: fmt
fmt: ## Format Go code and Terraform examples
	gofmt -s -w .
	terraform fmt -recursive ./examples/

.PHONY: generate
generate: ## Regenerate docs/ from the provider schema and examples
	go generate ./...

.PHONY: debug
debug: ## Build without optimisations and start the provider in debug mode (for delve)
	go build -gcflags="all=-N -l" -o $(BINARY)
	./$(BINARY) -debug

.PHONY: help
help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-10s %s\n", $$1, $$2}'
