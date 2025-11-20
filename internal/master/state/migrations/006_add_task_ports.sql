-- +goose Up
-- +goose StatementBegin
ALTER TABLE tasks ADD COLUMN IF NOT EXISTS ports JSONB;
ALTER TABLE tasks ADD COLUMN IF NOT EXISTS resources JSONB;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE tasks DROP COLUMN IF EXISTS resources;
ALTER TABLE tasks DROP COLUMN IF EXISTS ports;
-- +goose StatementEnd
