-- +goose Up
CREATE TABLE transaction_items (
    id BIGSERIAL PRIMARY KEY,
    transaction_id BIGINT NOT NULL,
    sku_id TEXT NOT NULL,
    owned BOOLEAN NOT NULL DEFAULT FALSE,
    count INTEGER NOT NULL,
    product_price DOUBLE PRECISION NOT NULL DEFAULT 0,
    inbound_cost DOUBLE PRECISION NOT NULL DEFAULT 0,
    broken_count INTEGER NOT NULL DEFAULT 0,
    lost_count INTEGER NOT NULL DEFAULT 0,
    total DOUBLE PRECISION NOT NULL,

    CONSTRAINT fk_transaction_items_inv_transactions
        FOREIGN KEY (transaction_id)
        REFERENCES inv_transactions(id)
        ON DELETE CASCADE
);

-- Optional indexes (recommended)
CREATE INDEX idx_transaction_items_tx_id ON transaction_items(transaction_id);
CREATE INDEX idx_transaction_items_sku_id ON transaction_items(sku_id);


-- +goose Down
DROP TABLE IF EXISTS transaction_items;