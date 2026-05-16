package warehouse_models

import "time"

type StockEventLog struct {
	ID        string `gorm:"primarykey"`
	Raw       []byte
	CreatedAt time.Time
}
