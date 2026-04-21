package warehouse_models

import (
	"time"

	"github.com/pdcgo/shared/db_models"
)

type DailySkuHistory struct {
	T                time.Time       `gorm:"column:t;uniqueIndex:idx_daily_sku_histories_unique"`
	SkuID            db_models.SkuID `gorm:"column:sku_id;uniqueIndex:idx_daily_sku_histories_unique"`
	WarehouseID      uint64          `gorm:"column:warehouse_id;uniqueIndex:idx_daily_sku_histories_unique"`
	StartStockCount  int64           `gorm:"column:start_stock_count"`
	EndStockCount    int64           `gorm:"column:end_stock_count"`
	StartStockAmount float64         `gorm:"column:start_stock_amount"`
	EndStockAmount   float64         `gorm:"column:end_stock_amount"`
	DiffStockCount   int64           `gorm:"column:diff_stock_count"`
	DiffStockAmount  float64         `gorm:"column:diff_stock_amount"`
}

func (DailySkuHistory) TableName() string {
	return "daily_sku_histories"
}
