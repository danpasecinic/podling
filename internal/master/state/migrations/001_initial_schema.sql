-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS tasks (
    task_id VARCHAR(255) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    image VARCHAR(255) NOT NULL,
    env JSONB,
    status VARCHAR(50) NOT NULL,
    node_id VARCHAR(255),
    container_id VARCHAR(255),
    created_at TIMESTAMP NOT NULL,
    started_at TIMESTAMP,
    finished_at TIMESTAMP,
    error TEXT
);

CREATE TABLE IF NOT EXISTS nodes (
    node_id VARCHAR(255) PRIMARY KEY,
    hostname VARCHAR(255) NOT NULL,
    port INTEGER NOT NULL,
    status VARCHAR(50) NOT NULL,
    capacity INTEGER NOT NULL,
    running_tasks INTEGER NOT NULL DEFAULT 0,
    last_heartbeat TIMESTAMP NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_tasks_status ON tasks(status);
CREATE INDEX IF NOT EXISTS idx_tasks_node_id ON tasks(node_id);
CREATE INDEX IF NOT EXISTS idx_nodes_status ON nodes(status);
CREATE INDEX IF NOT EXISTS idx_nodes_last_heartbeat ON nodes(last_heartbeat);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_nodes_last_heartbeat;
DROP INDEX IF EXISTS idx_nodes_status;
DROP INDEX IF EXISTS idx_tasks_node_id;
DROP INDEX IF EXISTS idx_tasks_status;
DROP TABLE IF EXISTS nodes;
DROP TABLE IF EXISTS tasks;
-- +goose StatementEnd
