BINARY_NAME := agentic-sdd
CMD_DIR := ./cmd/agentic-sdd

.PHONY: help preview apply test vet docker-e2e hooks-install mod-verify tidy-check build build-darwin verify tag release

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

mod-verify: ## Verify module dependencies match go.sum
	GOCACHE=$(GOCACHE) go mod verify

tidy-check: ## Fail if `go mod tidy` would change go.mod/go.sum
	GOCACHE=$(GOCACHE) go mod tidy
	git diff --exit-code -- go.mod go.sum

# Plain `go build` would target whatever OS the runner is on; agentic-sdd's
# release artifacts are darwin-only (see .goreleaser.yml), so cross-compile
# explicitly instead of relying on the host toolchain's default GOOS.
build-darwin: ## Cross-compile darwin amd64+arm64 binaries into bin/
	GOOS=darwin GOARCH=amd64 go build -trimpath -o bin/$(BINARY_NAME)_darwin_amd64 $(CMD_DIR)
	GOOS=darwin GOARCH=arm64 go build -trimpath -o bin/$(BINARY_NAME)_darwin_arm64 $(CMD_DIR)

build: ## Build a binary for the current OS/arch into bin/
	go build -trimpath -o bin/$(BINARY_NAME) $(CMD_DIR)

verify: mod-verify tidy-check vet test build-darwin ## Run the full local verification gate

tag: ## Create an annotated git tag (use V=v0.1.0 MSG="release message")
	@if [ -z "$(V)" ]; then echo "V is required (e.g. V=v0.1.0)"; exit 1; fi
	@if [ -z "$(MSG)" ]; then echo "MSG is required"; exit 1; fi
	git tag -a $(V) -m "$(MSG)"
	@echo "Tagged $(V) — push with: git push origin $(V)"

release: ## Dry-run GoReleaser snapshot locally (no publish, no tag required)
	GOVERSION=$(shell go version | awk '{print $$3}') goreleaser release --snapshot --clean
