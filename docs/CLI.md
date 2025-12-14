# CLI Reference

The `podling` CLI provides a user-friendly interface to interact with the Podling orchestrator.

## Installation

```bash
make build
cp bin/podling /usr/local/bin/
```

## Configuration

The CLI can be configured via:

1. **Command-line flags**: `--master http://localhost:8070`
2. **Environment variables**: `PODLING_MASTER_URL=http://localhost:8070`
3. **Config file**: `~/.podling.yaml` (future)

## Global Flags

All commands support these flags:

| Flag            | Description           | Default                 |
|-----------------|-----------------------|-------------------------|
| `--master`      | Master API URL        | `http://localhost:8070` |
| `--verbose, -v` | Enable verbose output | false                   |
| `--config`      | Config file location  | `$HOME/.podling.yaml`   |
| `--help, -h`    | Show help             | -                       |

## Task Commands

### Run Task

Submit a new task to run a single container.

```bash
podling run <name> --image <image> [flags]
```

**Flags:**

- `--image`: Container image (required)
- `--env`: Environment variables (repeatable)

**Examples:**

```bash
podling run my-nginx --image nginx:latest

podling run my-redis --image redis:latest --env PORT=6379 --env MODE=standalone

podling --master http://production:8070 run my-task --image alpine:latest
```

### List Tasks

```bash
podling ps [flags]
```

**Flags:**

- `--task, -t`: Get detailed info for a specific task

**Examples:**

```bash
podling ps

podling ps --task <task-id>
podling ps -t <task-id>
```

### View Logs

```bash
podling logs <task-id> [flags]
```

**Flags:**

- `--tail`: Number of lines to show (default: 100)

**Examples:**

```bash
podling logs <task-id>

podling logs <task-id> --tail 50
```

## Pod Commands

### Create Pod

Create a multi-container pod.

```bash
podling pod create <name> [flags]
```

**Flags:**

- `--container`: Container spec (repeatable, format: `name:image[:env1=val1,env2=val2]`)
- `--namespace`: Pod namespace (default: "default")
- `--label`: Labels (repeatable, format: `key=value`)

**Examples:**

```bash
podling pod create my-web --container nginx:nginx:latest

podling pod create my-app \
  --container app:myapp:1.0:PORT=8080,ENV=prod \
  --container sidecar:nginx:latest \
  --namespace production \
  --label app=myapp \
  --label version=1.0
```

**Container Specification Format:**

```
name:image[:env1=val1,env2=val2]
```

| Example                               | Description                |
|---------------------------------------|----------------------------|
| `nginx:nginx:latest`                  | Simple container           |
| `app:myapp:1.0:PORT=8080,DB=postgres` | With environment variables |

### List Pods

```bash
podling pod list
```

### Get Pod

```bash
podling pod get <pod-id>
```

Shows detailed pod information including all container statuses.

### Delete Pod

```bash
podling pod delete <pod-id>
```

## Service Commands

### Create Service

```bash
podling service create <name> [flags]
```

**Flags:**

- `--selector`: Pod selector labels (repeatable, format: `key=value`)
- `--port`: Service ports (repeatable, format: `[name:]port[:targetPort]`)
- `--namespace`: Service namespace (default: "default")
- `--label`: Labels (repeatable)

**Examples:**

```bash
podling service create web --selector app=nginx --port 80

podling service create api \
  --selector app=backend,tier=api \
  --port http:8080:80 \
  --port metrics:9090 \
  --namespace production
```

**Port Format:**

| Format         | Description                        |
|----------------|------------------------------------|
| `80`           | Port 80, target 80                 |
| `8080:80`      | Port 8080, target 80               |
| `http:8080`    | Named port "http" on 8080          |
| `http:8080:80` | Named "http", port 8080, target 80 |

### List Services

```bash
podling service list [flags]
```

**Flags:**

- `--namespace`: Filter by namespace

### Get Service

```bash
podling service get <service-id>
```

### Delete Service

```bash
podling service delete <service-id>
```

## Node Commands

### List Nodes

```bash
podling nodes [flags]
```

**Flags:**

- `--verbose`: Show additional details

**Output:**

```
ID          HOSTNAME    PORT   STATUS   CAPACITY   TASKS   LAST HEARTBEAT
worker-1    localhost   8081   online   10         2       30s ago
worker-2    localhost   8082   online   10         1       25s ago
```

## Examples

### Deploy a Web Application

```bash
podling pod create web-app \
  --container app:myapp:latest:PORT=8080 \
  --container nginx:nginx:alpine \
  --label app=web \
  --label tier=frontend

podling service create web \
  --selector app=web \
  --port 80:8080
```

### Monitor Cluster

```bash
podling nodes
podling pod list
podling service list
```

### Debug a Task

```bash
podling ps -t <task-id>
podling logs <task-id> --tail 200
```
