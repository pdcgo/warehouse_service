package warehouse_models

import (
	"time"

	"github.com/pdcgo/schema/services/warehouse_iface/v1"
)

type StockChangeLog struct {
	ID            int64                           `db:"id"`
	SkuID         string                          `db:"sku_id"`
	WarehouseID   int64                           `db:"warehouse_id"`
	ActorID       int64                           `db:"actor_id"`
	TransactionID int64                           `db:"transaction_id"`
	ChangeCount   int32                           `db:"change_count"`
	ChangeAmount  float64                         `db:"change_amount"`
	TransactionAt time.Time                       `db:"transaction_at"`
	Type          warehouse_iface.StockChangeType `db:"type"`
	CreatedAt     time.Time                       `db:"created_at"`
}
