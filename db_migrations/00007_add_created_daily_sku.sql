-- +goose Up
-- +goose StatementBegin
ALTER TABLE daily_sku_histories
    ADD COLUMN created_at TIMESTAMPTZ NOT NULL DEFAULT NOW();
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE daily_sku_histories
    DROP COLUMN IF EXISTS created_at;
-- +goose StatementEnd