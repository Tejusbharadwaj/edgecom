.PHONY: proto build test test-integration run docker-build docker-run docker-test clean docs install-tools

# Variables
PROTO_PATH := proto
GO_OUT_PATH := internal/grpc
GOBIN := $(shell go env GOPATH)/bin
PKG := github.com/tejusbharadwaj/edgecom
BIN_NAME := time-series-service

# Proto generation
proto:
	@echo "Generating protobuf files..."
	protoc \
		--plugin=protoc-gen-go=$(GOBIN)/protoc-gen-go \
		--plugin=protoc-gen-go-grpc=$(GOBIN)/protoc-gen-go-grpc \
		--proto_path=$(PROTO_PATH) \
		--go_out=. \
		--go_opt=paths=source_relative \
		--go-grpc_out=. \
		--go-grpc_opt=paths=source_relative \
		$(PROTO_PATH)/timeseries.proto

# Build
build: proto
	@echo "Building $(BIN_NAME)..."
	go mod tidy
	go build -o bin/$(BIN_NAME) ./cmd/main.go

# Testing
test:
	@echo "Running unit tests..."
	go test ./... -tags=!integration

test-integration:
	@echo "Running integration tests..."
	docker compose up --build db test

# Local development
run: build
	@echo "Running $(BIN_NAME)..."
	./bin/$(BIN_NAME)

# Docker operations
docker-build:
	@echo "Building Docker images..."
	docker compose build

docker-run:
	@echo "Running service in Docker..."
	docker compose up --build app

docker-test:
	@echo "Running tests in Docker..."
	docker compose up --build db test

# Cleanup
clean:
	@echo "Cleaning up..."
	rm -rf bin docs
	docker compose down --volumes

# Documentation
docs: proto
	@echo "Starting documentation server..."
	@if ! command -v pkgsite >/dev/null 2>&1; then \
		echo "Installing pkgsite..."; \
		go install golang.org/x/pkgsite/cmd/pkgsite@latest; \
	fi
	@echo "Documentation available at http://localhost:6060/$(PKG)"
	@echo "Press Ctrl+C to stop the server"
	pkgsite -http=:6060

# Tool installation
install-tools:
	@echo "Installing required tools..."
	@echo "Installing protoc plugins..."
	@GOBIN=$(GOBIN) go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	@GOBIN=$(GOBIN) go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
	@echo "Installing documentation tools..."
	@GOBIN=$(GOBIN) go install golang.org/x/pkgsite/cmd/pkgsite@latest
	@echo "Tools installed to $(GOBIN)"
	@echo "Make sure $(GOBIN) is in your PATH"

# Development helpers
.PHONY: help
help:
	@echo "Available targets:"
	@echo "  build            - Build the service"
	@echo "  clean            - Clean up build artifacts and Docker volumes"
	@echo "  docker-build     - Build Docker images"
	@echo "  docker-run       - Run service in Docker"
	@echo "  docker-test      - Run tests in Docker"
	@echo "  docs             - Start documentation server"
	@echo "  install-tools    - Install required development tools"
	@echo "  proto            - Generate protobuf code"
	@echo "  run              - Run service locally"
	@echo "  test             - Run unit tests"
	@echo "  test-integration - Run integration tests"