package models

import (
	"slices"
	"time"

	"github.com/pdcgo/shared/db_models"
)

type ExpenseType string

const (
	ExpenseTypeBank        ExpenseType = "bank"
	ExpenseTypeBasicSalary ExpenseType = "basic_salary"
	ExpenseTypeBonusSalary ExpenseType = "bonus_salary"
	ExpenseTypeKitchen     ExpenseType = "kitchen"
	ExpenseTypeOther       ExpenseType = "other"
)

var CanCreateExpense = map[db_models.TeamType]map[ExpenseType]bool{
	db_models.WarehouseTeamType: map[ExpenseType]bool{
		ExpenseTypeKitchen: true,
		ExpenseTypeOther:   true,
	},
	db_models.AdminTeamType: map[ExpenseType]bool{
		ExpenseTypeBank:        true,
		ExpenseTypeBasicSalary: true,
		ExpenseTypeBonusSalary: true,
		ExpenseTypeKitchen:     true,
		ExpenseTypeOther:       true,
	},
}

func (ExpenseType) EnumList() []string {
	return []string{
		"bank",
		"basic_salary",
		"bonus_salary",
		"kitchen",
		"other",
	}
}

func (t ExpenseType) NeedAdminPermission() bool {
	values := []ExpenseType{ExpenseTypeBank, ExpenseTypeBasicSalary, ExpenseTypeBonusSalary}
	return slices.Contains(values, t)
}

type WareExpenseAccount struct {
	ID            uint      `json:"id" gorm:"primarykey"`
	AccountTypeID uint      `json:"account_type_id"`
	Name          string    `json:"name"`
	NumberID      string    `json:"number_id" gorm:"unique"`
	Disabled      bool      `json:"disabled"`
	CreatedAt     time.Time `json:"created_at"`

	AccountType *db_models.AccountType `json:"account_type,omitempty" gorm:"foreignKey:AccountTypeID;"`
}

// GetEntityID implements authorization_iface.Entity
func (w *WareExpenseAccount) GetEntityID() string {
	return "ware_expense_account"
}

type WareExpenseAccountWarehouse struct {
	ID           uint `json:"id" gorm:"primarykey"`
	AccountID    uint `json:"account_id"`
	WarehouseID  uint `json:"warehouse_id"`
	IsOpsAccount bool `json:"is_ops_account"`

	Account *WareExpenseAccount `gorm:"foreignKey:AccountID"`
}

type WareExpenseHistory struct {
	ID          uint        `json:"id" gorm:"primarykey"`
	WarehouseID uint        `json:"warehouse_id"`
	AccountID   uint        `json:"account_id"`
	CreatedByID uint        `json:"created_by_id"`
	ExpenseType ExpenseType `json:"expense_type"`
	Amount      float64     `json:"amount"`
	Note        string      `json:"note"`
	At          time.Time   `json:"at"`
	CreatedAt   time.Time   `json:"created_at"`
}

// GetEntityID implements authorization_iface.Entity
func (w *WareExpenseHistory) GetEntityID() string {
	return "ware_expense_history"
}

type WareBalanceAccountHistory struct {
	ID          uint `json:"id" gorm:"primarykey"`
	WarehouseID uint `json:"warehouse_id"`
	AccountID   uint `json:"account_id" gorm:"index:account_at,unique"`
	CreatedByID uint `json:"created_by_id"`

	Amount    float64   `json:"amount"`
	At        time.Time `json:"at" gorm:"index:account_at,unique"` // per day
	CreatedAt time.Time `json:"created_at"`

	Account *WareExpenseAccount `json:"account,omitempty" gorm:"foreignkey:AccountID;"`
}
