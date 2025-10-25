.PHONY: help build run test test-coverage clean dev install-tools fmt lint

# Default target
help:
	@echo "Podling - Container Orchestrator"
	@echo ""
	@echo "Available targets:"
	@echo "  make build          - Build all binaries"
	@echo "  make run            - Run the master controller"
	@echo "  make dev            - Run with hot reloading (Air)"
	@echo "  make test           - Run tests"
	@echo "  make test-coverage  - Run tests with coverage"
	@echo "  make test-race      - Run tests with race detector"
	@echo "  make fmt            - Format code"
	@echo "  make lint           - Run linter"
	@echo "  make clean          - Clean build artifacts"
	@echo "  make install-tools  - Install development tools"

# Build all binaries
build:
	@echo "Building master..."
	@go build -o bin/podling-master ./cmd/master
	@echo "Building worker..."
	@go build -o bin/podling-worker ./cmd/worker
	@echo "Build complete!"

# Run master controller
run:
	@go run ./cmd/master

# Development mode with hot reloading
dev:
	@air

# Run all tests
test:
	@go test -v ./...

# Run tests with coverage
test-coverage:
	@go test -v -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Run tests with race detector
test-race:
	@go test -race -v ./...

# Format code
fmt:
	@gofmt -s -w .
	@go mod tidy

# Run linter (requires golangci-lint)
lint:
	@golangci-lint run ./...

# Clean build artifacts
clean:
	@rm -rf bin/ tmp/ coverage.out coverage.html
	@echo "Cleaned build artifacts"

# Install development tools
install-tools:
	@echo "Installing development tools..."
	@go install github.com/air-verse/air@latest
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@echo "Tools installed successfully!"

# Download dependencies
deps:
	@go mod download
	@go mod verify
