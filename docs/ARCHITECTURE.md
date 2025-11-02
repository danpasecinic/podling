# Podling Architecture Diagrams

## System Architecture

```mermaid
graph TB
    subgraph "User Interface"
        CLI[CLI Tool<br/>podling]
    end

    subgraph "Control Plane - Master :8080"
        API[REST API<br/>Echo Framework]
        Scheduler[Scheduler<br/>Task Assignment]
        State[State Manager<br/>Thread-Safe Store]
        API --> Scheduler
        API --> State
        Scheduler --> State
    end

    subgraph "Data Plane - Workers"
        W1[Worker Agent :8081<br/>Node Registration<br/>Heartbeats]
        W2[Worker Agent :8082<br/>Node Registration<br/>Heartbeats]
        W3[Worker Agent :8083<br/>Node Registration<br/>Heartbeats]
        D1[Docker Engine<br/>Container Runtime]
        D2[Docker Engine<br/>Container Runtime]
        D3[Docker Engine<br/>Container Runtime]
        W1 --> D1
        W2 --> D2
        W3 --> D3
    end

    CLI -->|HTTP| API
    Scheduler -->|Assign Tasks| W1
    Scheduler -->|Assign Tasks| W2
    Scheduler -->|Assign Tasks| W3
    W1 -.->|Heartbeat| State
    W2 -.->|Heartbeat| State
    W3 -.->|Heartbeat| State
    style CLI fill: #e1f5ff
    style API fill: #ffe1e1
    style Scheduler fill: #ffe1e1
    style State fill: #ffe1e1
    style W1 fill: #e1ffe1
    style W2 fill: #e1ffe1
    style W3 fill: #e1ffe1
```

## Task Lifecycle Flow

```mermaid
sequenceDiagram
    participant User
    participant CLI
    participant API
    participant State
    participant Scheduler
    participant Worker
    participant Docker

    User->>CLI: podling run nginx --image nginx:latest
    CLI->>API: POST /api/v1/tasks
    API->>State: Create Task (status: pending)
    State-->>API: Task ID
    API-->>CLI: 201 Created
    CLI-->>User: Task created: task-123

    loop Every 1 second
        Scheduler->>State: Get pending tasks
        State-->>Scheduler: [task-123, ...]
        Scheduler->>State: Get online nodes
        State-->>Scheduler: [worker-1, worker-2, ...]
        Scheduler->>Scheduler: Select best node
        Scheduler->>State: Update task (status: scheduled, node: worker-1)
    end

    Scheduler->>Worker: Assign task-123
    Worker->>State: Update task (status: running)
    Worker->>Docker: Create container
    Docker-->>Worker: Container ID
    Worker->>Docker: Start container
    Docker-->>Worker: Success
    Worker->>State: Update task (container_id, started_at)

    loop Container Running
        Worker->>Docker: Check container status
        Docker-->>Worker: Running
    end

    Docker->>Worker: Container exited
    Worker->>Docker: Get exit code & logs
    Docker-->>Worker: Exit code: 0
    Worker->>State: Update task (status: completed, finished_at)
    Worker->>Docker: Remove container
```

## Worker Heartbeat Mechanism

```mermaid
sequenceDiagram
    participant Worker
    participant Master
    participant State
    participant Scheduler

    Worker->>Master: POST /api/v1/heartbeat
    Note over Worker: Every 10 seconds<br/>{node_id, capacity, running_tasks}
    Master->>State: Update node heartbeat
    State-->>Master: OK
    Master-->>Worker: 200 OK

    alt Heartbeat Timeout (>30s)
        Scheduler->>State: Check node health
        State-->>Scheduler: worker-1 offline
        Scheduler->>State: Mark node as offline
        Scheduler->>State: Get tasks on worker-1
        State-->>Scheduler: [task-123, task-456]
        Scheduler->>State: Mark tasks as failed
        Note over Scheduler: Optionally reschedule
    end

    alt Worker Recovers
        Worker->>Master: POST /api/v1/heartbeat
        Master->>State: Update node (status: online)
        Worker->>Master: POST /api/v1/nodes (re-register)
    end
```

## CLI Command Flow

```mermaid
graph TD
    Start[User runs podling command]
    Start --> CMD{Command Type}

    CMD -->|run| RunFlow[podling run nginx --image nginx:latest]
    CMD -->|ps| PsFlow[podling ps]
    CMD -->|nodes| NodesFlow[podling nodes]
    CMD -->|logs| LogsFlow[podling logs task-123]

    RunFlow --> R1[Parse arguments]
    R1 --> R2[POST /api/v1/tasks]
    R2 --> R3[Display task ID & status]

    PsFlow --> P1{--task flag?}
    P1 -->|Yes| P2[GET /api/v1/tasks/:id]
    P1 -->|No| P3[GET /api/v1/tasks]
    P2 --> P4[Display detailed task info]
    P3 --> P5[Display task table]

    NodesFlow --> N1[GET /api/v1/nodes]
    N1 --> N2[Display node table]

    LogsFlow --> L1[GET /api/v1/tasks/:id]
    L1 --> L2[Find worker node]
    L2 --> L3[GET worker:8081/api/v1/tasks/:id/logs]
    L3 --> L4[Display logs to stdout]

    style RunFlow fill:#e1f5ff
    style PsFlow fill:#e1f5ff
    style NodesFlow fill:#e1f5ff
    style LogsFlow fill:#e1f5ff
```

## State Management

```mermaid
graph TB
    subgraph "State Manager (Thread-Safe)"
        Tasks[Tasks Map<br/>sync.RWMutex]
        Nodes[Nodes Map<br/>sync.RWMutex]
    end

    subgraph "Read Operations"
        GetTask[GetTask]
        ListTasks[ListTasks]
        GetNode[GetNode]
        ListNodes[ListNodes]
    end

    subgraph "Write Operations"
        CreateTask[CreateTask]
        UpdateTask[UpdateTask]
        RegisterNode[RegisterNode]
        UpdateNodeHeartbeat[UpdateNodeHeartbeat]
    end

    GetTask -.->|RLock| Tasks
    ListTasks -.->|RLock| Tasks
    GetNode -.->|RLock| Nodes
    ListNodes -.->|RLock| Nodes

    CreateTask -->|Lock| Tasks
    UpdateTask -->|Lock| Tasks
    RegisterNode -->|Lock| Nodes
    UpdateNodeHeartbeat -->|Lock| Nodes

    Tasks --> TaskData[Task Data:<br/>- ID, Name, Image<br/>- Status, NodeID<br/>- ContainerID<br/>- Timestamps<br/>- Error]
    Nodes --> NodeData[Node Data:<br/>- ID, Hostname, Port<br/>- Status, Capacity<br/>- RunningTasks<br/>- LastHeartbeat]

    style Tasks fill:#ffe1e1
    style Nodes fill:#ffe1e1
```

## Scheduling Algorithm

```mermaid
flowchart TD
    Start[Scheduler Loop<br/>Every 1 second]
    Start --> GetPending[Get Pending Tasks from State]
    GetPending --> CheckTasks{Any Pending?}
    CheckTasks -->|No| Sleep[Sleep 1s]
    CheckTasks -->|Yes| GetNodes[Get Online Nodes]
    
    GetNodes --> CheckNodes{Any Online Nodes?}
    CheckNodes -->|No| Sleep
    CheckNodes -->|Yes| SelectNode[Select Best Node]
    
    SelectNode --> Criteria{Selection Criteria}
    Criteria --> C1[1. Lowest Running Tasks]
    Criteria --> C2[2. Most Available Capacity]
    Criteria --> C3[3. Fastest Heartbeat]
    
    C1 --> BestNode[Best Node Selected]
    C2 --> BestNode
    C3 --> BestNode
    
    BestNode --> Assign[Assign Task to Node]
    Assign --> UpdateState[Update State:<br/>- task.Status = scheduled<br/>- task.NodeID = node_id]
    UpdateState --> NotifyWorker[Send Task to Worker<br/>POST worker/api/v1/tasks]
    NotifyWorker --> NextTask{More Tasks?}
    NextTask -->|Yes| SelectNode
    NextTask -->|No| Sleep
    
    Sleep --> Start

    style Start fill:#ffe1e1
    style SelectNode fill:#fff4e1
    style Assign fill:#e1ffe1
```

## Data Models

```mermaid
classDiagram
    class Task {
        +string TaskID
        +string Name
        +string Image
        +string Status
        +string NodeID
        +string ContainerID
        +map~string,string~ Env
        +time.Time CreatedAt
        +time.Time StartedAt
        +time.Time FinishedAt
        +string Error
    }

    class Pod {
        +string PodID
        +string Name
        +string Namespace
        +map~string,string~ Labels
        +Container[] Containers
        +string Status
        +string NodeID
        +RestartPolicy RestartPolicy
        +time.Time CreatedAt
        +time.Time ScheduledAt
        +time.Time StartedAt
        +time.Time FinishedAt
        +string Message
        +string Reason
    }

    class Container {
        +string Name
        +string Image
        +string[] Command
        +string[] Args
        +map~string,string~ Env
        +ContainerPort[] Ports
        +HealthCheck LivenessProbe
        +HealthCheck ReadinessProbe
        +string ContainerID
        +string Status
        +string HealthStatus
        +int ExitCode
        +string Error
    }

    class Node {
        +string NodeID
        +string Hostname
        +int Port
        +string Status
        +int Capacity
        +int RunningTasks
        +time.Time LastHeartbeat
    }

    class TaskStatus {
        <<enumeration>>
        Pending
        Scheduled
        Running
        Completed
        Failed
    }

    class PodStatus {
        <<enumeration>>
        Pending
        Scheduled
        Running
        Succeeded
        Failed
    }

    class ContainerStatus {
        <<enumeration>>
        Waiting
        Running
        Terminated
    }

    class NodeStatus {
        <<enumeration>>
        Online
        Offline
    }

    Task --> TaskStatus
    Pod --> PodStatus
    Pod --> Container
    Container --> ContainerStatus
    Node --> NodeStatus
```

## API Endpoints

```mermaid
graph LR
    subgraph "Master API :8080"
        direction TB
        T[Tasks]
        T1[POST /api/v1/tasks<br/>Create Task]
        T2[GET /api/v1/tasks<br/>List Tasks]
        T3[GET /api/v1/tasks/:id<br/>Get Task]

        P[Pods]
        P1[POST /api/v1/pods<br/>Create Pod]
        P2[GET /api/v1/pods<br/>List Pods]
        P3[GET /api/v1/pods/:id<br/>Get Pod]
        P4[PUT /api/v1/pods/:id/status<br/>Update Pod Status]
        P5[DELETE /api/v1/pods/:id<br/>Delete Pod]

        N[Nodes]
        N1[POST /api/v1/nodes<br/>Register Node]
        N2[GET /api/v1/nodes<br/>List Nodes]
        N3[POST /api/v1/heartbeat<br/>Node Heartbeat]
    end

    subgraph "Worker API :8081+"
        direction TB
        W[Worker Operations]
        W1[POST /api/v1/tasks<br/>Assign Task]
        W2[GET /api/v1/tasks/:id<br/>Get Task Status]
        W3[GET /api/v1/tasks/:id/logs<br/>Get Container Logs]
    end

    style T1 fill:#e1f5ff
    style T2 fill:#e1f5ff
    style T3 fill:#e1f5ff
    style P1 fill:#d4edff
    style P2 fill:#d4edff
    style P3 fill:#d4edff
    style P4 fill:#d4edff
    style P5 fill:#d4edff
    style N1 fill:#ffe1e1
    style N2 fill:#ffe1e1
    style N3 fill:#ffe1e1
    style W1 fill:#e1ffe1
    style W2 fill:#e1ffe1
    style W3 fill:#e1ffe1
```

## Pod Lifecycle Flow

```mermaid
sequenceDiagram
    participant User
    participant CLI
    participant API
    participant State
    participant Scheduler
    participant Worker
    participant Docker

    User->>CLI: podling pod create my-app --container app:myapp:1.0 --container sidecar:nginx:latest
    CLI->>API: POST /api/v1/pods
    API->>API: Validate containers<br/>(unique names, required fields)
    API->>State: Create Pod (status: pending)
    State-->>API: Pod ID

    API->>Scheduler: Schedule Pod
    Scheduler->>State: Get online nodes
    State-->>Scheduler: [worker-1, worker-2]
    Scheduler->>Scheduler: Select node (round-robin)
    Scheduler->>State: Update pod (status: scheduled, node: worker-1)
    State-->>API: Success
    API-->>CLI: 201 Created
    CLI-->>User: Pod created: pod-123

    Note over Worker: Worker receives pod assignment

    loop For each container in pod
        Worker->>Docker: Pull image (app:myapp:1.0)
        Docker-->>Worker: Image pulled
        Worker->>Docker: Pull image (nginx:latest)
        Docker-->>Worker: Image pulled
    end

    Worker->>State: Update pod (status: running)

    par Container 1: app
        Worker->>Docker: Create container (app)
        Docker-->>Worker: Container ID
        Worker->>Docker: Start container (app)
        Worker->>Docker: Start health checker (app)
        Worker->>Docker: Wait for container (app)
    and Container 2: sidecar
        Worker->>Docker: Create container (sidecar)
        Docker-->>Worker: Container ID
        Worker->>Docker: Start container (sidecar)
        Worker->>Docker: Start health checker (sidecar)
        Worker->>Docker: Wait for container (sidecar)
    end

    Note over Worker: All containers running

    alt All containers exit successfully
        Docker->>Worker: Container app exited (code 0)
        Docker->>Worker: Container sidecar exited (code 0)
        Worker->>State: Update pod (status: succeeded)
    else Any container fails
        Docker->>Worker: Container app exited (code 1)
        Worker->>State: Update pod (status: failed)
    end

    Worker->>Docker: Remove all containers
    Docker-->>Worker: Cleanup complete
```

## Error Handling & Recovery

### Task State Diagram

```mermaid
stateDiagram-v2
    [*] --> Pending: Task Created
    Pending --> Scheduled: Scheduler Assigns
    Scheduled --> Running: Worker Starts Container

    Running --> Completed: Container Exit 0
    Running --> Failed: Container Exit != 0

    Pending --> Failed: No Available Nodes (timeout)
    Scheduled --> Failed: Worker Offline
    Running --> Failed: Worker Lost Heartbeat

    Failed --> [*]: Task Terminal
    Completed --> [*]: Task Terminal

    note right of Failed
        Errors logged to task.Error field
        - Container execution errors
        - Worker communication errors
        - Resource constraints
    end note
```

### Pod State Diagram

```mermaid
stateDiagram-v2
    [*] --> Pending: Pod Created
    Pending --> Scheduled: Scheduler Assigns
    Scheduled --> Running: All Containers Starting

    Running --> Succeeded: All Containers Exit 0
    Running --> Failed: Any Container Exit != 0

    Pending --> Failed: No Available Nodes (timeout)
    Scheduled --> Failed: Worker Offline
    Running --> Failed: Worker Lost Heartbeat
    Running --> Failed: Image Pull Failed

    Failed --> [*]: Pod Terminal
    Succeeded --> [*]: Pod Terminal

    note right of Running
        Pod is "Running" when all containers are running
        Each container has independent status:
        - Waiting: Container not started yet
        - Running: Container executing
        - Terminated: Container finished
    end note

    note right of Failed
        Pod fails if ANY container fails
        Errors logged per container
        - Image pull errors
        - Container execution errors
        - Health check failures
        - Resource constraints
    end note
```

## Deployment Topology

```mermaid
graph TB
    subgraph "Development (localhost)"
        DevMaster[Master :8080]
        DevWorker1[Worker :8081]
        DevWorker2[Worker :8082]
        DevDocker[Docker Desktop]
        
        DevWorker1 --> DevDocker
        DevWorker2 --> DevDocker
    end

    subgraph "Production (distributed)"
        ProdMaster[Master :8080<br/>control-plane.internal]
        
        subgraph "Worker Pool"
            ProdW1[Worker :8081<br/>worker-1.internal]
            ProdW2[Worker :8081<br/>worker-2.internal]
            ProdW3[Worker :8081<br/>worker-3.internal]
        end
        
        ProdD1[Docker Engine<br/>worker-1]
        ProdD2[Docker Engine<br/>worker-2]
        ProdD3[Docker Engine<br/>worker-3]
        
        ProdW1 --> ProdD1
        ProdW2 --> ProdD2
        ProdW3 --> ProdD3
    end

    LB[Load Balancer]
    LB --> ProdMaster
    
    style DevMaster fill:#ffe1e1
    style ProdMaster fill:#ffe1e1
```
