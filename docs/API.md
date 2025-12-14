# API Reference

The master controller exposes a RESTful API on port `8070` for managing tasks, pods, services, and worker nodes.

## Authentication

When authentication is enabled, include the JWT token in the Authorization header:

```bash
Authorization: Bearer <token>
```

## Health Check

```bash
GET /health

curl http://localhost:8070/health
```

Returns service status and authentication state.

## Task Endpoints

### Create Task

Submit a new task for execution.

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
```

Example:

```bash
curl -X POST http://localhost:8070/api/v1/tasks \
  -H "Content-Type: application/json" \
  -d '{"name":"nginx-task","image":"nginx:latest"}'
```

### List Tasks

```bash
GET /api/v1/tasks

curl http://localhost:8070/api/v1/tasks
```

### Get Task

```bash
GET /api/v1/tasks/{taskId}

curl http://localhost:8070/api/v1/tasks/20250119123456-abc12345
```

### Update Task Status

Update task execution status (typically called by workers).

```bash
PUT /api/v1/tasks/{taskId}/status
Content-Type: application/json

{
  "status": "running",
  "containerId": "docker-container-id"
}
```

### Task Status Flow

```
pending → scheduled → running → completed/failed
```

| Status    | Description                       |
|-----------|-----------------------------------|
| pending   | Task created, awaiting scheduling |
| scheduled | Task assigned to a worker node    |
| running   | Task is executing on a worker     |
| completed | Task finished successfully        |
| failed    | Task execution failed             |

## Pod Endpoints

Pods are groups of one or more containers with shared lifecycle.

### Create Pod

```bash
POST /api/v1/pods
Content-Type: application/json

{
  "name": "my-web-app",
  "namespace": "production",
  "labels": {
    "app": "web",
    "version": "1.0"
  },
  "containers": [
    {
      "name": "app",
      "image": "myapp:1.0",
      "env": {"PORT": "8080"}
    },
    {
      "name": "sidecar",
      "image": "nginx:latest"
    }
  ]
}
```

Example:

```bash
curl -X POST http://localhost:8070/api/v1/pods \
  -H "Content-Type: application/json" \
  -d '{"name":"web-pod","containers":[{"name":"nginx","image":"nginx:latest"}]}'
```

### List Pods

```bash
GET /api/v1/pods

curl http://localhost:8070/api/v1/pods
```

### Get Pod

```bash
GET /api/v1/pods/{podId}

curl http://localhost:8070/api/v1/pods/20250119123456-pod123
```

### Delete Pod

```bash
DELETE /api/v1/pods/{podId}

curl -X DELETE http://localhost:8070/api/v1/pods/20250119123456-pod123
```

### Pod Status Flow

```
pending → scheduled → running → succeeded/failed
```

| Status    | Description                       |
|-----------|-----------------------------------|
| pending   | Pod created, awaiting scheduling  |
| scheduled | Pod assigned to a worker node     |
| running   | All containers in pod are running |
| succeeded | All containers exited with code 0 |
| failed    | One or more containers failed     |

## Service Endpoints

Services provide stable endpoints for discovering and accessing pods.

### Create Service

```bash
POST /api/v1/services
Content-Type: application/json

{
  "name": "web-service",
  "namespace": "default",
  "selector": {
    "app": "nginx"
  },
  "ports": [
    {
      "name": "http",
      "port": 80,
      "targetPort": 8080
    }
  ]
}
```

### List Services

```bash
GET /api/v1/services
GET /api/v1/services?namespace=production

curl http://localhost:8070/api/v1/services
```

### Get Service

```bash
GET /api/v1/services/{serviceId}

curl http://localhost:8070/api/v1/services/20250119123456-svc123
```

### Get Service Endpoints

```bash
GET /api/v1/services/{serviceId}/endpoints

curl http://localhost:8070/api/v1/services/20250119123456-svc123/endpoints
```

### Delete Service

```bash
DELETE /api/v1/services/{serviceId}

curl -X DELETE http://localhost:8070/api/v1/services/20250119123456-svc123
```

## Node Endpoints

### Register Node

```bash
POST /api/v1/nodes/register
Content-Type: application/json

{
  "hostname": "worker-1",
  "port": 8081,
  "capacity": 10
}
```

### Node Heartbeat

```bash
POST /api/v1/nodes/{nodeId}/heartbeat

curl -X POST http://localhost:8070/api/v1/nodes/worker-1/heartbeat
```

### List Nodes

```bash
GET /api/v1/nodes

curl http://localhost:8070/api/v1/nodes
```

## Worker Endpoints

These endpoints are exposed by worker nodes (port 8081+).

### Execute Task

Called by master to execute a task on the worker.

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

### Get Task Status

```bash
GET /api/v1/tasks/:id/status

curl http://localhost:8081/api/v1/tasks/task-id/status
```

### Get Task Logs

```bash
GET /api/v1/tasks/:id/logs?tail=100

curl http://localhost:8081/api/v1/tasks/task-id/logs?tail=100
```

## Testing with Postman

Import the provided Postman collection to test all endpoints:

1. Import `docs/postman/Podling.postman_collection.json` into Postman
2. Import `docs/postman/Podling.postman_environment.json` for local environment
3. Select "Podling - Local" environment
4. Start testing the API

See [POSTMAN_GUIDE.md](POSTMAN_GUIDE.md) for detailed testing workflow.
