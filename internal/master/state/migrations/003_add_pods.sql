-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS pods (
    pod_id VARCHAR(255) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    namespace VARCHAR(255) DEFAULT 'default',
    labels JSONB,
    annotations JSONB,
    containers JSONB NOT NULL,
    status VARCHAR(50) NOT NULL,
    node_id VARCHAR(255),
    restart_policy VARCHAR(50),
    created_at TIMESTAMP NOT NULL,
    scheduled_at TIMESTAMP,
    started_at TIMESTAMP,
    finished_at TIMESTAMP,
    message TEXT,
    reason VARCHAR(255)
);

CREATE INDEX IF NOT EXISTS idx_pods_status ON pods(status);
CREATE INDEX IF NOT EXISTS idx_pods_node_id ON pods(node_id);
CREATE INDEX IF NOT EXISTS idx_pods_namespace ON pods(namespace);
CREATE INDEX IF NOT EXISTS idx_pods_labels ON pods USING GIN(labels);
CREATE INDEX IF NOT EXISTS idx_pods_created_at ON pods(created_at DESC);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_pods_created_at;
DROP INDEX IF EXISTS idx_pods_labels;
DROP INDEX IF EXISTS idx_pods_namespace;
DROP INDEX IF EXISTS idx_pods_node_id;
DROP INDEX IF EXISTS idx_pods_status;
DROP TABLE IF EXISTS pods;
-- +goose StatementEnd
