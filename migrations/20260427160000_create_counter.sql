-- +goose Up
CREATE TABLE counters (
    id integer PRIMARY KEY,
    value integer NOT NULL DEFAULT 0 CHECK (value >= 0),
    updated_at timestamptz NOT NULL DEFAULT now()
);

INSERT INTO counters (id, value)
VALUES (1, 0)
ON CONFLICT (id) DO NOTHING;

-- +goose Down
DROP TABLE counters;

