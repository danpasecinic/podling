# Podling

[![CI](https://github.com/danpasecinic/podling/actions/workflows/ci.yml/badge.svg)](https://github.com/danpasecinic/podling/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/danpasecinic/podling)](https://goreportcard.com/report/github.com/danpasecinic/podling)
[![codecov](https://codecov.io/gh/danpasecinic/podling/branch/main/graph/badge.svg)](https://codecov.io/gh/danpasecinic/podling)

A lightweight, educational container orchestrator built from scratch in Go. Inspired by Kubernetes, it features a master
controller with REST API, worker agents that manage containers via Docker, and a CLI tool.

## Features

- **Master-Worker Architecture**: Distributed container management
- **REST API**: Echo-based HTTP server for control plane
- **Persistent Storage**: PostgreSQL or in-memory state store
- **Hot Reloading**: Air integration for rapid development
- **Production Patterns**: Following golang-standards/project-layout

## Quick Start

### Prerequisites

- Go 1.25 or later
- Docker Engine
- Make (optional, for convenience)
- PostgreSQL 12+ (optional, for persistent storage)

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

### Running with PostgreSQL (Recommended)

```bash
# Start PostgreSQL using docker-compose
docker-compose up -d

# Configure via .env file (easiest method)
cp .env.example .env
# Edit .env to set STORE_TYPE=postgres
make run
```

The master automatically loads `.env` if present, so no need to export variables manually.

### Running with In-Memory Store (Development)

```bash
# Default mode - no configuration needed
make dev

# Or explicitly set
export STORE_TYPE=memory
make run
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

# Run PostgreSQL tests (requires running database)
export TEST_DATABASE_URL="postgres://podling:podling123@localhost:5432/podling?sslmode=disable"
go test ./internal/master/state/
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
│   └── worker/            # Worker agent internals
│       ├── agent/         # Worker agent logic
│       └── docker/        # Docker SDK integration
├── docs/                  # Documentation
│   ├── postman/           # Postman collection for API testing
│   ├── POSTMAN_GUIDE.md   # API testing guide
│   └── SESSION_STATE.md   # Development session tracking
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

### Testing with Postman

Import the provided Postman collection to test all endpoints:

1. Import `docs/postman/Podling.postman_collection.json` into Postman
2. Import `docs/postman/Podling.postman_environment.json` for local environment
3. Select "Podling - Local" environment
4. Start testing the API

See [Postman Guide](docs/POSTMAN_GUIDE.md) for detailed testing workflow.

### Running the Worker

```bash
# Build and run a worker node
go build -o bin/podling-worker ./cmd/worker
./bin/podling-worker -node-id=worker-1 -port=8081

# Or use Make
make build && ./bin/podling-worker -node-id=worker-1

# Worker configuration options:
# -node-id: Unique worker identifier (required)
# -hostname: Worker hostname (default: localhost)
# -port: Worker port (default: 8081)
# -master-url: Master API URL (default: http://localhost:8080)
# -heartbeat-interval: Heartbeat interval (default: 30s)
# -shutdown-timeout: Graceful shutdown timeout (default: 30s)
```

The worker will:

- Connect to the master and send periodic heartbeats
- Execute tasks in Docker containers
- Report task status back to master
- Stream container logs via API
- Handle graceful shutdown with task cleanup

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

#### Worker Endpoints

**Execute Task** - Execute a task on worker (called by master)

```bash
POST /api/v1/tasks/:id/execute
Content-Type: application/json

{
  "task": {
    "taskId": "task-id",
    "name": "my-task",
    "image": "nginx:latest",
    "env": {"PORT": "8080"}
  }
}
```

**Get Task Status** - Get task execution status

```bash
GET /api/v1/tasks/:id/status

curl http://localhost:8081/api/v1/tasks/task-id/status
```

**Get Task Logs** - Stream container logs

```bash
GET /api/v1/tasks/:id/logs?tail=100

curl http://localhost:8081/api/v1/tasks/task-id/logs?tail=100
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

## CLI Usage

The `podling` CLI provides a user-friendly interface to interact with the Podling orchestrator.

### Installation

```bash
# Build the CLI
make build

# The binary will be at bin/podling
# Optionally, add it to your PATH
cp bin/podling /usr/local/bin/
```

### Configuration

The CLI can be configured via:

1. **Command-line flags**: `--master http://localhost:8080`
2. **Environment variables**: `PODLING_MASTER_URL=http://localhost:8080`
3. **Config file**: `~/.podling.yaml` (future enhancement)

### Commands

#### Run a Task

Submit a new task to run a container:

```bash
# Basic usage
podling run my-nginx --image nginx:latest

# With environment variables
podling run my-redis --image redis:latest --env PORT=6379 --env MODE=standalone

# With custom master URL
podling --master http://production:8080 run my-task --image alpine:latest
```

#### List Tasks

View all tasks or get details of a specific task:

```bash
# List all tasks
podling ps

# Get detailed info for a specific task
podling ps --task <task-id>
podling ps -t <task-id>
```

Output example:

```
ID                      NAME       IMAGE          STATUS     NODE        CREATED
20250119123456-abc123   my-nginx   nginx:latest   running    worker-1    2m
20250119123457-def456   my-redis   redis:latest   completed  worker-2    5m
```

#### List Worker Nodes

View all registered worker nodes:

```bash
# List all nodes
podling nodes

# With verbose output
podling nodes --verbose
```

Output example:

```
ID          HOSTNAME    PORT   STATUS   CAPACITY   TASKS   LAST HEARTBEAT
worker-1    localhost   8081   online   10         2       30s ago
worker-2    localhost   8082   online   10         1       25s ago
```

#### View Task Logs

Fetch container logs for a running task:

```bash
# Get logs (last 100 lines by default)
podling logs <task-id>

# Limit output
podling logs <task-id> --tail 50
```

### Global Flags

All commands support these global flags:

- `--master string`: Master API URL (default "http://localhost:8080")
- `--verbose, -v`: Enable verbose output
- `--config string`: Config file location (default "$HOME/.podling.yaml")
- `--help, -h`: Show help for any command

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
- Worker agent: 86.2%
- Docker client: 73.6%
- Overall: >85%

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

## License

See [LICENSE](LICENSE) file for details.

## Acknowledgments

- Inspired by [Kubernetes](https://kubernetes.io/)
- Project structure follows [golang-standards/project-layout](https://github.com/golang-standards/project-layout)
- Built with [Echo](https://echo.labstack.com/) framework
- Development powered by [Air](https://github.com/air-verse/air)
