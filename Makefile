.PHONY: help preview apply test vet hooks-install

GOCACHE ?= $(CURDIR)/.cache/go-build

help: ## Show available commands
	@awk 'BEGIN {FS = ":.*## "; print "Commands:"} /^[a-zA-Z_-]+:.*## / {printf "  %-14s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

preview: ## Show skill installs and replacements without changing files
	GOCACHE=$(GOCACHE) go run ./cmd/agentic-sdd

apply: ## Back up existing skills and install source skills
	GOCACHE=$(GOCACHE) go run ./cmd/agentic-sdd --apply

test: ## Run Go tests
	GOCACHE=$(GOCACHE) go test ./...

vet: ## Run Go static analysis
	GOCACHE=$(GOCACHE) go vet ./...

hooks-install: ## Install Lefthook Git hooks locally
	lefthook install
