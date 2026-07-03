.PHONY: help build test test-verbose lint vet fmt-check run-stdio run-http docker-build docker-run clean tidy

BINARY := mcp-server
IMAGE := mcp-server-microservices-design:dev

help: ## Show this help message
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

tidy: ## go mod tidy
	go mod tidy

build: ## Build the server binary
	go build -buildvcs=false -o $(BINARY) ./cmd/server

test: ## Run all tests
	go test ./...

test-verbose: ## Run all tests with verbose output
	go test -v ./...

test-boundary: ## Run just the boundary service tests (the new code in this session)
	go test -v ./internal/services/boundary/

vet: ## Run go vet
	go vet ./...

fmt-check: ## Check formatting
	@if [ -n "$$(gofmt -l .)" ]; then \
		echo "Files need formatting:"; \
		gofmt -l .; \
		exit 1; \
	fi

lint: vet fmt-check ## Run all static checks

run-stdio: build ## Run the server in stdio mode (for local MCP clients)
	./$(BINARY)

run-http: build ## Run the server in streamable HTTP mode on port 8080
	MCP_TRANSPORT=streamablehttp PORT=8080 ./$(BINARY)

docker-build: ## Build the container image
	docker build -t $(IMAGE) .

docker-run: docker-build ## Build and run the container in stdio mode
	docker run --rm -i $(IMAGE)

docker-run-http: docker-build ## Build and run the container in HTTP mode on port 8080
	docker run --rm -p 8080:8080 -e MCP_TRANSPORT=streamablehttp -e PORT=8080 $(IMAGE)

clean: ## Remove build artifacts
	rm -f $(BINARY)
	go clean -testcache

verify-all: tidy build test lint ## Full verification: tidy, build, test, lint
	@echo "All verification checks passed."
