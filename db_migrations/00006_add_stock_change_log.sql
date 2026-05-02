-- +goose Up
-- +goose StatementBegin
CREATE TABLE stock_change_logs (
    id             BIGSERIAL        PRIMARY KEY,
    sku_id         VARCHAR(255)     NOT NULL,
    warehouse_id   BIGINT           NOT NULL CHECK (warehouse_id > 0),
    actor_id       BIGINT           NOT NULL CHECK (actor_id > 0),
    transaction_id BIGINT           NOT NULL CHECK (transaction_id > 0),
    change_count   INTEGER          NOT NULL CHECK (change_count <> 0),
    change_amount  DOUBLE PRECISION NOT NULL CHECK (change_amount <> 0),
    transaction_at TIMESTAMPTZ      NOT NULL,
    type           INTEGER          NOT NULL DEFAULT 0,
    created_at     TIMESTAMPTZ      NOT NULL DEFAULT NOW()
);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE INDEX idx_stock_change_logs_sku_id         ON stock_change_logs (sku_id);
CREATE INDEX idx_stock_change_logs_warehouse_id   ON stock_change_logs (warehouse_id);
CREATE INDEX idx_stock_change_logs_transaction_id ON stock_change_logs (transaction_id);
CREATE INDEX idx_stock_change_logs_transaction_at ON stock_change_logs (transaction_at);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS stock_change_logs;
-- +goose StatementEnd