package warehouse_models

import (
	"time"

	"github.com/pdcgo/shared/db_models"
)

type InvItemProblem struct {
	ID       uint            `gorm:"primarykey" json:"id"`
	SkuID    db_models.SkuID `json:"sku_id"`
	TxID     uint            `json:"tx_id"`
	TxItemID uint            `json:"tx_item_id"`

	ProblemType string    `json:"broken_type"`
	ProblemNote string    `json:"problem_note"`
	Count       int       `json:"count"`
	Created     time.Time `json:"created"`
}
