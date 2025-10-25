# Postman API Testing Guide

This guide shows how to use the Postman collection to test the Podling API (Master + Worker).

## Setup

### 1. Import Collection

1. Open Postman
2. Click **Import** button
3. Select `docs/postman/Podling.postman_collection.json`
4. Select `docs/postman/Podling.postman_environment.json`
5. Click **Import**

### 2. Select Environment

1. Click the environment dropdown (top right)
2. Select **Podling - Local**
3. Ensure the master is running on `http://localhost:8080`
4. Ensure the worker is running on `http://localhost:8081`

## Testing Workflow

### Step 1: Health Checks

Test that both master and worker are running:

**Master Health:**
```
GET /health (Master - Health Check)
```

**Worker Health:**
```
GET /health (Worker - Health Check)
```

Expected responses:
```json
{
  "status": "ok",
  "service": "podling-master" // or "podling-worker"
}
```

### Step 2: Register a Worker Node

Register the worker node with the master:

```
POST /api/v1/nodes/register (Master - Nodes > Register Node)
```

Request body:
```json
{
  "hostname": "localhost",
  "port": 8081,
  "capacity": 10
}
```

Response includes `nodeId` - **copy this to the `node_id` environment variable**.

### Step 3: Create a Task

Submit a task for execution:

```
POST /api/v1/tasks (Master - Tasks > Create Task)
```

Request body:
```json
{
  "name": "nginx-task",
  "image": "nginx:alpine",
  "env": {
    "PORT": "8080"
  }
}
```

Response includes:
- `taskId` - **copy this to the `task_id` environment variable**
- `status` - should be "scheduled" if node available
- `nodeId` - the node assigned to run this task

### Step 4: Worker Executes Task (Optional Manual Test)

If you want to manually trigger task execution on the worker:

```
POST /api/v1/tasks/:taskId/execute (Worker - Tasks > Execute Task)
```

This is normally called by the master's scheduler. The worker will:
1. Pull the Docker image
2. Create and start the container
3. Report status back to master

### Step 5: List All Tasks

View all tasks:

```
GET /api/v1/tasks (Master - Tasks > List Tasks)
```

### Step 6: Get Specific Task

Get task details:

```
GET /api/v1/tasks/:taskId (Master - Tasks > Get Task)
```

The `{{task_id}}` variable will be used automatically.

### Step 7: Check Worker Task Status

Check task status on the worker:

```
GET /api/v1/tasks/:taskId/status (Worker - Tasks > Get Task Status)
```

### Step 8: Update Task Status

Simulate worker updating task status (normally done automatically):

**Mark as Running:**
```
PUT /api/v1/tasks/:taskId/status (Master - Tasks > Update Task Status)
```

Body:
```json
{
  "status": "running",
  "containerId": "docker-container-id-123"
}
```

**Mark as Completed:**
```json
{
  "status": "completed"
}
```

**Mark as Failed:**
```json
{
  "status": "failed",
  "error": "Container crashed"
}
```

### Step 9: Send Heartbeat

Keep node alive with heartbeat:

```
POST /api/v1/nodes/:nodeId/heartbeat (Master - Nodes > Node Heartbeat)
```

The `{{node_id}}` variable will be used automatically.

### Step 10: List All Nodes

View all registered nodes:

```
GET /api/v1/nodes (Master - Nodes > List Nodes)
```

## Complete Testing Scenario

1. **Start Services**: Run master on 8080, worker on 8081
2. **Health Checks**: Verify both are running
3. **Register Worker**: Get node ID, save to environment
4. **Create Task**: Get task ID, save to environment
5. **Worker Executes**: Task runs in Docker container
6. **Monitor Progress**: Check task status on master and worker
7. **Heartbeat**: Worker sends periodic heartbeats
8. **View Results**: List all tasks and nodes

## Task Status Flow

Tasks progress through these states:

```
pending → scheduled → running → completed/failed
```

- **pending**: Task created, awaiting scheduling
- **scheduled**: Task assigned to a worker node
- **running**: Task is executing on a worker
- **completed**: Task finished successfully
- **failed**: Task execution failed

## Path Variables

The collection uses path variables for dynamic IDs:

- `:taskId` - Replace with actual task ID (e.g., `20250119123456-abc12345`)
- `:nodeId` - Replace with actual node ID (e.g., `20250119123456-xyz98765`)

To update path variables in Postman:
1. Open the request
2. Click **Params** tab
3. Update the value in the **Path Variables** section

## Collection Organization

The collection is organized into folders:

- **Health Check** - Server status
- **Tasks** - Task lifecycle management
  - Create Task
  - List Tasks
  - Get Task
  - Update Task Status (Running, Completed, Failed)
- **Nodes** - Worker node management
  - Register Node
  - Node Heartbeat
  - List Nodes

## Environment Variables

The environment includes:

- `base_url` - Master server URL (default: `http://localhost:8080`)

To change the server URL:
1. Click environment dropdown
2. Click edit (pencil icon)
3. Update `base_url` value
4. Save

## Tips

1. **Save Responses**: Click "Save Response" → "Save as Example" to keep track of successful responses
2. **Tests Tab**: Add test scripts to validate responses automatically
3. **Pre-request Scripts**: Add scripts to generate dynamic data
4. **Run Collection**: Use Collection Runner to execute all requests sequentially

## Example Testing Sequence

1. ✓ Health Check - Verify server is running
2. ✓ Register Node - Create worker node
3. ✓ Create Task - Submit nginx task
4. ✓ List Tasks - Verify task is scheduled
5. ✓ Update Status to Running - Simulate worker starting container
6. ✓ Node Heartbeat - Keep worker alive
7. ✓ Update Status to Completed - Simulate task completion
8. ✓ Get Task - Verify final status
9. ✓ List Nodes - Check node status

## Troubleshooting

**Connection Error**:
- Ensure master is running: `./bin/podling-master`
- Check `base_url` in environment matches your server

**404 Not Found**:
- Verify the endpoint path is correct
- Check path variables are set

**400 Bad Request**:
- Verify request body JSON is valid
- Check all required fields are present

**500 Internal Server Error**:
- Check master logs for errors
- Verify data types match expected format

---

For API documentation, see [README.md](../README.md#api-documentation)
