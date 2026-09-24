.PHONY: help preview apply test

GOCACHE ?= /private/tmp/agentic-sdd-go-cache

help: ## Show available commands
	@awk 'BEGIN {FS = ":.*## "; print "Commands:"} /^[a-zA-Z_-]+:.*## / {printf "  %-12s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

preview: ## Show skill installs and replacements without changing files
	GOCACHE=$(GOCACHE) go run ./cmd/agentic-sdd

apply: ## Back up existing skills and install source skills
	GOCACHE=$(GOCACHE) go run ./cmd/agentic-sdd --apply

test: ## Run Go tests
	GOCACHE=$(GOCACHE) go test ./...
