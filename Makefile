.PHONY: help preview apply test vet docker-e2e hooks-install

GOCACHE ?= $(CURDIR)/.cache/go-build

help: ## Show available commands
	@awk 'BEGIN {FS = ":.*## "; print "Commands:"} /^[a-zA-Z_-]+:.*## / {printf "  %-14s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

preview: ## Show skill installs and replacements without changing files
	GOCACHE=$(GOCACHE) go run ./cmd/agentic-sdd preview

apply: ## Back up existing skills and install source skills
	GOCACHE=$(GOCACHE) go run ./cmd/agentic-sdd apply

test: ## Run Go tests
	GOCACHE=$(GOCACHE) go test ./...

vet: ## Run Go static analysis
	GOCACHE=$(GOCACHE) go vet ./...

docker-e2e: ## Exercise preview and apply in a Docker container with an isolated home
	sh tests/docker-e2e.sh

hooks-install: ## Install Lefthook Git hooks locally
	lefthook install
