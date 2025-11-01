-- +goose Up
-- +goose StatementBegin
ALTER TABLE tasks ADD COLUMN IF NOT EXISTS liveness_probe JSONB;
ALTER TABLE tasks ADD COLUMN IF NOT EXISTS readiness_probe JSONB;
ALTER TABLE tasks ADD COLUMN IF NOT EXISTS restart_policy VARCHAR(50);
ALTER TABLE tasks ADD COLUMN IF NOT EXISTS health_status VARCHAR(50);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE tasks DROP COLUMN IF EXISTS health_status;
ALTER TABLE tasks DROP COLUMN IF EXISTS restart_policy;
ALTER TABLE tasks DROP COLUMN IF EXISTS readiness_probe;
ALTER TABLE tasks DROP COLUMN IF EXISTS liveness_probe;
-- +goose StatementEnd
