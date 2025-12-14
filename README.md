# Podling

[![CI](https://github.com/danpasecinic/podling/actions/workflows/ci.yml/badge.svg)](https://github.com/danpasecinic/podling/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/danpasecinic/podling)](https://goreportcard.com/report/github.com/danpasecinic/podling)

A lightweight, educational container orchestrator built from scratch in Go. Inspired by Kubernetes, it features a master
controller with REST API, worker agents that manage containers via Docker, and a CLI tool.

## Features

- **Master-Worker Architecture**: Distributed container management
- **Multi-Container Pods**: Kubernetes-style pods with shared lifecycle and networking
- **Service Discovery**: ClusterIP allocation with DNS resolution
- **REST API**: Echo-based HTTP server for control plane
- **Web Dashboard**: Modern React UI with real-time monitoring
- **Health Checks**: Liveness and readiness probes (HTTP, TCP, Exec)
- **Persistent Storage**: PostgreSQL or in-memory state store

## Quick Start

### Prerequisites

- Go 1.25+, Docker Engine, Node.js 18+ (for dashboard)

### Run

```bash
git clone https://github.com/danpasecinic/podling.git && cd podling

# Start PostgreSQL
docker-compose up -d

# Configure and run master
cp .env.example .env
make run

# In another terminal, start a worker
make build
./bin/podling-worker -node-id=worker-1
```

### Basic Usage

```bash
# Run a container
./bin/podling run my-nginx --image nginx:latest

# Create a multi-container pod
./bin/podling pod create my-app \
  --container app:myapp:1.0:PORT=8080 \
  --container sidecar:nginx:latest

# Create a service
./bin/podling service create web --selector app=nginx --port 80

# View status
./bin/podling ps
./bin/podling pod list
./bin/podling nodes
```

## Architecture

| Component | Description                              | Port  |
|-----------|------------------------------------------|-------|
| Master    | API server, scheduling, state management | 8070  |
| Worker    | Container execution, heartbeats          | 8081+ |
| DNS       | Service discovery resolution             | 5353  |
| Dashboard | Web UI                                   | 5173  |

```
podling/
├── cmd/                    # Main applications (master, worker, CLI)
├── internal/
│   ├── types/             # Core data models (Task, Pod, Node, Service)
│   ├── master/            # API handlers, scheduler, state store
│   └── worker/            # Docker client, health checks, agent
├── web/                   # React dashboard
└── docs/                  # Documentation
```

<img width="3828" height="2350" alt="Dashboard" src="https://github.com/user-attachments/assets/b146f06a-6e8a-47e0-9a68-2652e105953a" />

## Documentation

| Document                                 | Description              |
|------------------------------------------|--------------------------|
| [API Reference](docs/API.md)             | REST API endpoints       |
| [CLI Reference](docs/CLI.md)             | Command-line interface   |
| [Architecture](docs/ARCHITECTURE.md)     | System diagrams          |
| [Pod Networking](docs/POD_NETWORKING.md) | Shared network namespace |
| [Storage](docs/STORAGE.md)               | PostgreSQL configuration |
| [Security](docs/SECURITY.md)             | SSRF prevention          |
| [Postman Guide](docs/POSTMAN_GUIDE.md)   | API testing              |

## Development

```bash
make dev          # Run with hot reload
make test         # Run tests
make test-coverage # Generate coverage report
make lint         # Run linter
make build        # Build all binaries
```

## License

See [LICENSE](LICENSE) file for details.

## Acknowledgments

- Inspired by [Kubernetes](https://kubernetes.io/)
- Built with [Echo](https://echo.labstack.com/), [React](https://react.dev/), [shadcn/ui](https://ui.shadcn.com/)
