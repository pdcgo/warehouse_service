-- +goose Up
-- +goose StatementBegin
-- Legacy-compatible: create the warehouses table only if it does not already exist
-- (the legacy system may already own it). Column set matches shared/db_models.Warehouse
-- (id is client-supplied, not auto-increment).
CREATE TABLE IF NOT EXISTS warehouses (
    id             BIGINT PRIMARY KEY,
    name           TEXT,
    is_full        BOOLEAN NOT NULL DEFAULT false,
    use_fixed_fee  BOOLEAN NOT NULL DEFAULT false,
    fee_fix        DOUBLE PRECISION NOT NULL DEFAULT 0,
    fee_percent    REAL NOT NULL DEFAULT 0,
    max_fee        DOUBLE PRECISION NOT NULL DEFAULT 0,
    "desc"         TEXT,
    address        TEXT,
    open_time      TIMESTAMPTZ,
    close_time     TIMESTAMPTZ,
    close_order    TIMESTAMPTZ,
    is_closed      BOOLEAN NOT NULL DEFAULT false,
    created        TIMESTAMPTZ,
    deleted        BOOLEAN NOT NULL DEFAULT false,
    rack_count     BIGINT NOT NULL DEFAULT 0,
    order_count    BIGINT NOT NULL DEFAULT 0,
    capacity       BIGINT NOT NULL DEFAULT 0,
    max_capacity   BIGINT NOT NULL DEFAULT 0,
    product_count  BIGINT NOT NULL DEFAULT 0
);

CREATE INDEX IF NOT EXISTS idx_warehouses_deleted ON warehouses (deleted);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
-- No-op: never drop the (potentially legacy-owned) warehouses table on rollback.
SELECT 1;
-- +goose StatementEnd
