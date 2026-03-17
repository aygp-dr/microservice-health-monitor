.PHONY: build run test lint clean help

BINARY := $(shell basename $(CURDIR))

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

build: ## Build the binary
	go build -o bin/$(BINARY) .

run: build ## Build and run
	./bin/$(BINARY)

test: ## Run tests
	go vet ./...

lint: ## Run go vet
	go vet ./...

clean: ## Clean build artifacts
	rm -rf bin/
