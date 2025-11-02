-- +goose Up
-- +goose StatementBegin

ALTER TABLE nodes ADD COLUMN resources JSONB;

UPDATE nodes
SET resources = jsonb_build_object(
    'capacity', jsonb_build_object(
        'cpu', capacity * 1000,
        'memory', capacity * 1024 * 1024 * 1024
    ),
    'allocatable', jsonb_build_object(
        'cpu', capacity * 1000,
        'memory', capacity * 1024 * 1024 * 1024
    ),
    'used', jsonb_build_object(
        'cpu', 0,
        'memory', 0
    )
)
WHERE resources IS NULL;


ALTER TABLE nodes ALTER COLUMN resources SET NOT NULL;
ALTER TABLE nodes DROP COLUMN capacity;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE nodes ADD COLUMN capacity INTEGER;

UPDATE nodes
SET capacity = CAST((resources->'capacity'->>'cpu')::numeric / 1000 AS INTEGER)
WHERE capacity IS NULL;

ALTER TABLE nodes ALTER COLUMN capacity SET NOT NULL;
ALTER TABLE nodes DROP COLUMN resources;
-- +goose StatementEnd
