package models

import (
	"time"
)

type ExpenseType string

type WareExpenseAccount struct {
	ID        uint   `gorm:"primarykey"`
	NumberID  string `gorm:"unique"`
	Name      string
	Disabled  bool
	CreatedAt time.Time
}

type WareExpenseAccountWarehouse struct {
	ID           uint `gorm:"primarykey"`
	AccountID    uint
	WarehouseID  uint
	IsOpsAccount bool
}

type WareExpenseHistory struct {
	ID          uint `gorm:"primarykey"`
	WarehouseID uint
	AccountID   uint
	ExpenseType ExpenseType
	Amount      float64
	CreatedAt   time.Time
}
