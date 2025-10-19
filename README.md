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

- Go 1.25 or later
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

## API Documentation

The master controller exposes a RESTful API for managing tasks and worker nodes.

### Running the Master

```bash
# Build and run the master
go build -o bin/podling-master ./cmd/master
./bin/podling-master

# Or use Make
make build && ./bin/podling-master
```

The master will start on `http://localhost:8080` with the following endpoints:

### Endpoints

#### Health Check

```bash
GET /health

curl http://localhost:8080/health
```

#### Task Management

**Create Task** - Submit a new task for execution

```bash
POST /api/v1/tasks
Content-Type: application/json

{
  "name": "my-nginx-task",
  "image": "nginx:latest",
  "env": {
    "PORT": "8080"
  }
}

# Example
curl -X POST http://localhost:8080/api/v1/tasks \
  -H "Content-Type: application/json" \
  -d '{"name":"nginx-task","image":"nginx:latest"}'
```

**List Tasks** - Get all tasks

```bash
GET /api/v1/tasks

curl http://localhost:8080/api/v1/tasks
```

**Get Task** - Get specific task details

```bash
GET /api/v1/tasks/{taskId}

curl http://localhost:8080/api/v1/tasks/20250119123456-abc12345
```

**Update Task Status** - Update task execution status (typically called by workers)

```bash
PUT /api/v1/tasks/{taskId}/status
Content-Type: application/json

{
  "status": "running",
  "containerId": "docker-container-id"
}

# Example
curl -X PUT http://localhost:8080/api/v1/tasks/20250119123456-abc12345/status \
  -H "Content-Type: application/json" \
  -d '{"status":"running","containerId":"abc123"}'
```

#### Node Management

**Register Node** - Register a worker node

```bash
POST /api/v1/nodes/register
Content-Type: application/json

{
  "hostname": "worker-1",
  "port": 8081,
  "capacity": 10
}

# Example
curl -X POST http://localhost:8080/api/v1/nodes/register \
  -H "Content-Type: application/json" \
  -d '{"hostname":"worker-1","port":8081,"capacity":10}'
```

**Node Heartbeat** - Update node heartbeat

```bash
POST /api/v1/nodes/{nodeId}/heartbeat

curl -X POST http://localhost:8080/api/v1/nodes/20250119123456-xyz98765/heartbeat
```

**List Nodes** - Get all registered nodes

```bash
GET /api/v1/nodes

curl http://localhost:8080/api/v1/nodes
```

### Task Status Flow

Tasks progress through the following states:

```
pending → scheduled → running → completed/failed
```

- **pending**: Task created, awaiting scheduling
- **scheduled**: Task assigned to a worker node
- **running**: Task is executing on a worker
- **completed**: Task finished successfully
- **failed**: Task execution failed

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

### Phase 2: Master Controller ✅ COMPLETE

- [x] Build Echo-based HTTP API server
- [x] Implement all REST API endpoints (7 total)
- [x] Build round-robin scheduler
- [x] Add automatic task scheduling on creation
- [x] Write comprehensive tests (91.9% API coverage, 100% scheduler coverage)
- [x] Add godoc comments for all exported types
- [x] Validate with race detector

### Phase 3: Worker Agent (Next)

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

**Current Test Coverage**:

- State management: 94.7%
- API handlers: 91.9%
- Scheduler: 100%
- Overall: >90%

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
