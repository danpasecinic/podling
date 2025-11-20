.PHONY: help build run test test-coverage clean dev install-tools fmt lint install uninstall run-cluster stop-cluster

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
	@echo "  make install        - Install binaries to /usr/local/bin"
	@echo "  make uninstall      - Remove installed binaries"
	@echo "  make run-cluster    - Run master + 2 workers in background"
	@echo "  make stop-cluster   - Stop all running podling processes"

# Build all binaries
build:
	@echo "Building master..."
	@go build -o bin/podling-master ./cmd/master
	@echo "Building worker..."
	@go build -o bin/podling-worker ./cmd/worker
	@echo "Building CLI..."
	@go build -o bin/podling ./cmd/podling
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

# Install binaries to /usr/local/bin
install: build
	@echo "Installing binaries to /usr/local/bin..."
	@sudo cp bin/podling /usr/local/bin/podling
	@sudo cp bin/podling-master /usr/local/bin/podling-master
	@sudo cp bin/podling-worker /usr/local/bin/podling-worker
	@echo "✓ Installed successfully!"
	@echo "You can now use: podling, podling-master, podling-worker"

# Remove installed binaries
uninstall:
	@echo "Removing installed binaries..."
	@sudo rm -f /usr/local/bin/podling
	@sudo rm -f /usr/local/bin/podling-master
	@sudo rm -f /usr/local/bin/podling-worker
	@echo "✓ Uninstalled successfully!"

# Run master and workers together
run-cluster: build
	@echo "Starting Podling cluster..."
	@./bin/podling-master > /tmp/podling-master.log 2>&1 & echo $$! > /tmp/podling-master.pid
	@sleep 2
	@./bin/podling-worker -node-id=worker-1 -port=8071 > /tmp/podling-worker-1.log 2>&1 & echo $$! > /tmp/podling-worker-1.pid
	@./bin/podling-worker -node-id=worker-2 -port=8072 > /tmp/podling-worker-2.log 2>&1 & echo $$! > /tmp/podling-worker-2.pid
	@sleep 1
	@echo "✓ Cluster started!"
	@echo "  Master:   http://localhost:8070"
	@echo "  Worker 1: http://localhost:8071"
	@echo "  Worker 2: http://localhost:8072"
	@echo ""
	@echo "Logs:"
	@echo "  Master:   tail -f /tmp/podling-master.log"
	@echo "  Worker 1: tail -f /tmp/podling-worker-1.log"
	@echo "  Worker 2: tail -f /tmp/podling-worker-2.log"
	@echo ""
	@echo "Stop with: make stop-cluster"

# Stop all podling processes
stop-cluster:
	@echo "Stopping Podling cluster..."
	@if [ -f /tmp/podling-master.pid ]; then kill $$(cat /tmp/podling-master.pid) 2>/dev/null || true; rm /tmp/podling-master.pid; fi
	@if [ -f /tmp/podling-worker-1.pid ]; then kill $$(cat /tmp/podling-worker-1.pid) 2>/dev/null || true; rm /tmp/podling-worker-1.pid; fi
	@if [ -f /tmp/podling-worker-2.pid ]; then kill $$(cat /tmp/podling-worker-2.pid) 2>/dev/null || true; rm /tmp/podling-worker-2.pid; fi
	@pkill -f podling-master || true
	@pkill -f podling-worker || true
	@echo "✓ Cluster stopped!"
