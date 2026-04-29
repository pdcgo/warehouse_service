-- +goose Up
CREATE TABLE IF NOT EXISTS daily_sku_histories (
    id                  BIGSERIAL       PRIMARY KEY,
    t                   TIMESTAMPTZ     NOT NULL,
    sku_id              VARCHAR(255)    NOT NULL,
    warehouse_id        BIGINT          NOT NULL,
    start_stock_count   BIGINT          NOT NULL DEFAULT 0,
    end_stock_count     BIGINT          NOT NULL DEFAULT 0,
    start_stock_amount  DOUBLE PRECISION NOT NULL DEFAULT 0,
    end_stock_amount    DOUBLE PRECISION NOT NULL DEFAULT 0,
    diff_stock_count    BIGINT          NOT NULL DEFAULT 0,
    diff_stock_amount   DOUBLE PRECISION NOT NULL DEFAULT 0,
    CONSTRAINT idx_daily_sku_histories_unique UNIQUE (t, sku_id, warehouse_id)
);

-- +goose Down
DROP TABLE IF EXISTS daily_sku_histories;
