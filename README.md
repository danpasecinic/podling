# Podling

[![CI](https://github.com/danpasecinic/podling/actions/workflows/ci.yml/badge.svg)](https://github.com/danpasecinic/podling/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/danpasecinic/podling)](https://goreportcard.com/report/github.com/danpasecinic/podling)
[![codecov](https://codecov.io/gh/danpasecinic/podling/branch/main/graph/badge.svg)](https://codecov.io/gh/danpasecinic/podling)

A lightweight, educational container orchestrator built from scratch in Go. Inspired by Kubernetes, it features a master
controller with REST API, worker agents that manage containers via Docker, and a CLI tool.

## Features

- **Master-Worker Architecture**: Distributed container management
- **REST API**: Echo-based HTTP server for control plane
- **State Management**: Thread-safe in-memory state store
- **Hot Reloading**: Air integration for rapid development
- **Production Patterns**: Following golang-standards/project-layout

## Quick Start

### Prerequisites

- Go 1.25+
- Docker Engine
- Make (optional, for convenience)

### Installation

```bash
# Clone the repository
git clone https://github.com/danpasecinic/podling.git
cd podling

# Install dependencies
go mod download

# Install development tools (Air, linters)
make install-tools
```

### Development

```bash
# Run with hot reloading
make dev

# Or run directly
make run

# Run tests
make test

# Run tests with coverage
make test-coverage

# Run tests with race detector
make test-race
```

## Project Structure

Following the [golang-standards/project-layout](https://github.com/golang-standards/project-layout):

```
podling/
├── cmd/                    # Main applications
│   ├── master/            # Master controller entry point
│   ├── worker/            # Worker agent entry point
│   └── podling/           # CLI tool entry point
├── internal/               # Private application code
│   ├── types/             # Core data models
│   │   ├── task.go        # Task model and status
│   │   └── node.go        # Node model and status
│   ├── master/            # Master controller internals
│   │   ├── api/           # HTTP API handlers (Echo)
│   │   ├── scheduler/     # Task scheduling logic
│   │   └── state/         # State management
│   │       ├── store.go       # In-memory state store
│   │       └── store_test.go  # State store tests
│   └── worker/            # Worker agent internals
│       ├── agent/         # Worker agent logic
│       └── docker/        # Docker SDK integration
├── .air.toml              # Air hot reload configuration
├── Makefile               # Development commands
└── go.mod                 # Go module definition
```

## Architecture

### Components

| Component  | Responsibility                                    | Port  |
|------------|---------------------------------------------------|-------|
| **Master** | Task scheduling, API server, state management     | 8080  |
| **Worker** | Container execution, status reporting, heartbeats | 8081+ |
| **CLI**    | User interface for task submission and monitoring | -     |

### Technology Stack

- **Language**: Go 1.25
- **Web Framework**: [Echo v4](https://echo.labstack.com/) - High performance, minimalist
- **Container Runtime**: [Docker Engine API](https://docs.docker.com/engine/api/)
- **Hot Reload**: [Air](https://github.com/air-verse/air) - Live reload for Go apps
- **Testing**: Go's built-in testing with race detector

## Development Progress

### Phase 1: Foundation ✅ COMPLETE

- [x] Set up Go project following golang-standards/project-layout
- [x] Define core data models (Task and Node)
- [x] Implement thread-safe in-memory state store with mutex protection
- [x] Write comprehensive unit tests (94.7% coverage)
- [x] Validate concurrent access with race detector
- [x] Add Echo framework for HTTP server
- [x] Configure Air for hot reloading
- [x] Create Makefile for development workflow

### Phase 2: Master Controller (Next)

- [ ] Build Echo-based HTTP API server
- [ ] Implement task submission endpoint (`POST /api/v1/tasks`)
- [ ] Implement worker registration endpoint (`POST /api/v1/nodes/register`)
- [ ] Build round-robin scheduler
- [ ] Add structured logging

### Phase 3: Worker Agent

- [ ] Build worker HTTP server
- [ ] Integrate Docker SDK for container management
- [ ] Implement task execution logic
- [ ] Add heartbeat mechanism
- [ ] Implement status reporting

### Phase 4: CLI Tool

- [ ] Build CLI using cobra or similar
- [ ] Implement `podling run` command
- [ ] Implement `podling ps` command
- [ ] Implement `podling nodes` command

## Testing

```bash
# Run all tests
make test

# Run with coverage report (generates coverage.html)
make test-coverage

# Run with race detector
make test-race

# Or use go directly
go test ./...
go test -race ./...
go test -coverprofile=coverage.out ./...
```

**Current Test Coverage**: 94.7% for state management

## Code Quality

```bash
# Format code
make fmt

# Run linter (requires golangci-lint)
make lint
```

## Building

```bash
# Build all binaries
make build

# Binaries will be in bin/
# - bin/podling-master
# - bin/podling-worker
# - bin/podling
```

## Contributing

This is an educational project. Contributions are welcome! Please:

1. Follow Go best practices
2. Maintain test coverage above 60%
3. Run `make fmt` before committing
4. Ensure all tests pass with `make test-race`

## License

See [LICENSE](LICENSE) file for details.

## Acknowledgments

- Inspired by [Kubernetes](https://kubernetes.io/)
- Project structure follows [golang-standards/project-layout](https://github.com/golang-standards/project-layout)
- Built with [Echo](https://echo.labstack.com/) framework
- Development powered by [Air](https://github.com/air-verse/air)
