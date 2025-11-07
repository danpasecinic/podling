-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS services (
    service_id VARCHAR(255) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    namespace VARCHAR(255) DEFAULT 'default',
    type VARCHAR(50) NOT NULL DEFAULT 'ClusterIP',
    cluster_ip VARCHAR(50),
    selector JSONB,
    ports JSONB NOT NULL,
    labels JSONB,
    annotations JSONB,
    session_affinity VARCHAR(50),
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_services_namespace ON services(namespace);
CREATE INDEX IF NOT EXISTS idx_services_name ON services(name);
CREATE INDEX IF NOT EXISTS idx_services_namespace_name ON services(namespace, name);
CREATE INDEX IF NOT EXISTS idx_services_labels ON services USING GIN(labels);
CREATE INDEX IF NOT EXISTS idx_services_created_at ON services(created_at DESC);

CREATE TABLE IF NOT EXISTS endpoints (
    service_id VARCHAR(255) PRIMARY KEY,
    service_name VARCHAR(255) NOT NULL,
    namespace VARCHAR(255) DEFAULT 'default',
    subsets JSONB NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    FOREIGN KEY (service_id) REFERENCES services(service_id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_endpoints_namespace ON endpoints(namespace);
CREATE INDEX IF NOT EXISTS idx_endpoints_service_name ON endpoints(service_name);
CREATE INDEX IF NOT EXISTS idx_endpoints_namespace_name ON endpoints(namespace, service_name);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_endpoints_namespace_name;
DROP INDEX IF EXISTS idx_endpoints_service_name;
DROP INDEX IF EXISTS idx_endpoints_namespace;
DROP TABLE IF EXISTS endpoints;

DROP INDEX IF EXISTS idx_services_created_at;
DROP INDEX IF EXISTS idx_services_labels;
DROP INDEX IF EXISTS idx_services_namespace_name;
DROP INDEX IF EXISTS idx_services_name;
DROP INDEX IF EXISTS idx_services_namespace;
DROP TABLE IF EXISTS services;
-- +goose StatementEnd
