package models

import (
	"time"

	"github.com/pdcgo/shared/db_models"
)

type ExpenseType string

var CanCreateExpense = map[db_models.TeamType]map[ExpenseType]bool{
	db_models.WarehouseTeamType: map[ExpenseType]bool{
		"kitchen": true,
		"other":   true,
	},
	db_models.AdminTeamType: map[ExpenseType]bool{
		"bank": true,
	},
}

type WareExpenseAccount struct {
	ID        uint   `gorm:"primarykey"`
	NumberID  string `gorm:"unique"`
	Name      string
	Disabled  bool
	CreatedAt time.Time
}

// GetEntityID implements authorization_iface.Entity
func (w *WareExpenseAccount) GetEntityID() string {
	return "ware_expense_account"
}

type WareExpenseAccountWarehouse struct {
	ID           uint `gorm:"primarykey"`
	AccountID    uint
	WarehouseID  uint
	IsOpsAccount bool

	Account *WareExpenseAccount `gorm:"foreignKey:AccountID"`
}

type WareExpenseHistory struct {
	ID          uint `gorm:"primarykey"`
	WarehouseID uint
	AccountID   uint
	ExpenseType ExpenseType
	Amount      float64
	CreatedAt   time.Time
}
