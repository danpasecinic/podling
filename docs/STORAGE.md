# Storage Configuration

Podling supports two storage backends for persisting task and node state.

## Storage Options

### 1. In-Memory Store (Default)

**Use case:** Development, testing, learning

**Characteristics:**

- Fast and simple
- No external dependencies
- Data is lost on restart
- Thread-safe with mutex-based concurrency control

**Configuration:**

```bash
export STORE_TYPE=memory
# Or simply omit STORE_TYPE (defaults to memory)
```

### 2. PostgreSQL Store

**Use case:** Production, persistent data, multiple master instances (future)

**Characteristics:**

- Data persists across restarts
- ACID guarantees
- Supports complex queries
- Production-ready

**Configuration:**

```bash
export STORE_TYPE=postgres
export DATABASE_URL="postgres://username:password@host:port/database?sslmode=disable"
```

## Quick Start with PostgreSQL

### Using Docker Compose

1. **Start PostgreSQL:**

```bash
docker-compose up -d
```

2. **Verify it's running:**

```bash
docker-compose ps
# Should show podling-postgres as "Up"
```

3. **Configure and run master:**

```bash
export STORE_TYPE=postgres
export DATABASE_URL="postgres://podling:podling123@localhost:5432/podling?sslmode=disable"
./bin/podling-master
```

The database schema will be automatically created on first startup.

### Manual PostgreSQL Setup

1. **Install PostgreSQL** (version 12 or later)

2. **Create database and user:**

```sql
CREATE
DATABASE podling;
CREATE
USER podling WITH PASSWORD 'your_password';
GRANT ALL PRIVILEGES ON DATABASE
podling TO podling;
```

3. **Configure connection:**

```bash
export DATABASE_URL="postgres://podling:your_password@localhost:5432/podling?sslmode=disable"
export STORE_TYPE=postgres
```

## Database Schema

The PostgreSQL store uses two main tables:

### Tasks Table

```sql
CREATE TABLE tasks
(
    task_id      VARCHAR(255) PRIMARY KEY,
    name         VARCHAR(255) NOT NULL,
    image        VARCHAR(255) NOT NULL,
    env          JSONB,
    status       VARCHAR(50)  NOT NULL,
    node_id      VARCHAR(255),
    container_id VARCHAR(255),
    created_at   TIMESTAMP    NOT NULL,
    started_at   TIMESTAMP,
    finished_at  TIMESTAMP,
    error        TEXT
);
```

### Nodes Table

```sql
CREATE TABLE nodes
(
    node_id        VARCHAR(255) PRIMARY KEY,
    hostname       VARCHAR(255) NOT NULL,
    port           INTEGER      NOT NULL,
    status         VARCHAR(50)  NOT NULL,
    capacity       INTEGER      NOT NULL,
    running_tasks  INTEGER      NOT NULL DEFAULT 0,
    last_heartbeat TIMESTAMP    NOT NULL
);
```

### Indexes

- `idx_tasks_status` - Fast filtering by task status
- `idx_tasks_node_id` - Fast lookup of tasks by node
- `idx_nodes_status` - Fast filtering of online/offline nodes
- `idx_nodes_last_heartbeat` - Efficient heartbeat monitoring

## Environment Variables

| Variable            | Required          | Default  | Description                             |
|---------------------|-------------------|----------|-----------------------------------------|
| `STORE_TYPE`        | No                | `memory` | Storage backend: `memory` or `postgres` |
| `DATABASE_URL`      | Yes (if postgres) | -        | PostgreSQL connection string            |
| `TEST_DATABASE_URL` | No                | -        | Database URL for running tests          |

## Connection String Format

```
postgres://username:password@host:port/database?options
```

**Example:**

```
postgres://podling:podling123@localhost:5432/podling?sslmode=disable
```

**SSL Modes:**

- `disable` - No SSL (development only)
- `require` - Require SSL
- `verify-ca` - Verify certificate
- `verify-full` - Full verification (production)

## Testing

### Running PostgreSQL Tests

```bash
# Start PostgreSQL
docker-compose up -d

# Set test database URL
export TEST_DATABASE_URL="postgres://podling:podling123@localhost:5432/podling?sslmode=disable"

# Run PostgreSQL-specific tests
go test -v ./internal/master/state/ -run TestPostgres
```

Tests automatically:

- Skip if `TEST_DATABASE_URL` is not set
- Clean up test data before and after each test
- Verify CRUD operations and edge cases

## Troubleshooting

### Connection Issues

**Error:** `failed to connect to database: connection refused`

**Solution:** Ensure PostgreSQL is running:

```bash
docker-compose ps
# Or for manual install:
sudo systemctl status postgresql
```

**Error:** `FATAL: password authentication failed`

**Solution:** Check credentials in `DATABASE_URL` match PostgreSQL configuration

**Error:** `database "podling" does not exist`

**Solution:** Create the database:

```bash
docker-compose exec postgres psql -U podling -c "CREATE DATABASE podling;"
```

### Migration Issues

**Error:** `relation "tasks" already exists`

**Solution:** This is normal - migrations use `CREATE TABLE IF NOT EXISTS`

**To reset database:**

```bash
docker-compose down -v  # Removes volume with data
docker-compose up -d    # Fresh start
```

## Performance Considerations

### In-Memory Store

- **Read latency:** <1μs
- **Write latency:** <1μs
- **Concurrency:** Read/write mutexes, good for moderate load

### PostgreSQL Store

- **Read latency:** 1-5ms (local), 10-50ms (network)
- **Write latency:** 2-10ms (local), 20-100ms (network)
- **Concurrency:** MVCC, excellent for high concurrent load
- **Scalability:** Can handle millions of tasks

### Recommendations

- **Development:** Use in-memory for fastest iteration
- **Testing:** Use PostgreSQL to catch production issues
- **Production:** Always use PostgreSQL for data persistence
- **CI/CD:** Use in-memory for fast test runs, PostgreSQL for integration tests

## Future Enhancements

Potential storage improvements:

1. **Connection pooling** - Reuse database connections
2. **Read replicas** - Scale read operations
3. **Caching layer** - Redis for frequently accessed data
4. **Time-series tables** - For metrics and audit logs
5. **Partitioning** - Partition tasks table by date for better performance
6. **Backup/restore** - Automated backup scripts

## Migration Between Stores

Currently, there's no built-in migration tool between storage backends.

**To migrate from in-memory to PostgreSQL:**

1. Export tasks via API:

```bash
curl http://localhost:8080/api/v1/tasks > tasks.json
```

2. Stop master, configure PostgreSQL, restart

3. Re-create tasks via API:

```bash
# Parse tasks.json and POST each task
```

Future versions may include a migration CLI tool.
