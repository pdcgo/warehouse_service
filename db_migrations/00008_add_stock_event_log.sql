-- +goose Up
CREATE TABLE stock_event_logs (
    id         TEXT PRIMARY KEY,
    raw        BYTEA,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- +goose Down
DROP TABLE IF EXISTS stock_event_logs;