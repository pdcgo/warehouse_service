package models

import (
	"slices"
	"time"

	"github.com/pdcgo/shared/db_models"
)

type ExpenseType string

const (
	ExpenseTypeEquity      ExpenseType = "equity"       // Modal
	ExpenseTypeBasicSalary ExpenseType = "basic_salary" // Gaji Pokok
	ExpenseTypeServer      ExpenseType = "server"       // Server

	ExpenseTypePettyCash   ExpenseType = "petty_cash"  // Kas Kecil
	ExpenseTypeTransport   ExpenseType = "transport"   // Transport
	ExpenseTypeReceivable  ExpenseType = "receivable"  // Piutang
	ExpenseTypeInternet    ExpenseType = "internet"    // Internet
	ExpenseTypePacking     ExpenseType = "packing"     // Packing
	ExpenseTypeShipping    ExpenseType = "shipping"    // Ongkir
	ExpenseTypeElectricity ExpenseType = "electricity" // Listrik
	ExpenseTypeKitchen     ExpenseType = "kitchen"     // Dapur
	ExpenseTypeEquipment   ExpenseType = "equipment"   // Perlengkapan
	ExpenseTypeTools       ExpenseType = "tools"       // Peralatan
	ExpenseTypeOther       ExpenseType = "other"       // Lain-Lain

	// additional types
	// ExpenseTypeBank        ExpenseType = "bank"         // Bank
	// ExpenseTypePayable     ExpenseType = "payable"      // Hutang
	// ExpenseTypeBonusSalary ExpenseType = "bonus_salary" // Bonus
)

var CanCreateExpense = map[db_models.TeamType]map[ExpenseType]bool{
	db_models.AdminTeamType: map[ExpenseType]bool{
		ExpenseTypeEquity:      true,
		ExpenseTypeBasicSalary: true,
		ExpenseTypePettyCash:   true,
		ExpenseTypeServer:      true,
		ExpenseTypeTransport:   true,
		ExpenseTypeReceivable:  true,
		ExpenseTypeInternet:    true,
		ExpenseTypePacking:     true,
		ExpenseTypeShipping:    true,
		ExpenseTypeElectricity: true,
		ExpenseTypeKitchen:     true,
		ExpenseTypeEquipment:   true,
		ExpenseTypeTools:       true,
		ExpenseTypeOther:       true,
		// ExpenseTypeBank:        true,
		// ExpenseTypePayable:     true,
		// ExpenseTypeBonusSalary: true,
	},
	db_models.WarehouseTeamType: map[ExpenseType]bool{
		ExpenseTypePettyCash:   true,
		ExpenseTypeTransport:   true,
		ExpenseTypeReceivable:  true,
		ExpenseTypeInternet:    true,
		ExpenseTypePacking:     true,
		ExpenseTypeShipping:    true,
		ExpenseTypeElectricity: true,
		ExpenseTypeKitchen:     true,
		ExpenseTypeEquipment:   true,
		ExpenseTypeTools:       true,
		ExpenseTypeOther:       true,
	},
}

func (ExpenseType) EnumList() []string {
	return []string{
		"equity",
		"basic_salary",
		"server",

		"petty_cash",
		"transport",
		"internet",
		"packing",
		"receivable",
		"shipping",
		"electricity",
		"kitchen",
		"equipment",
		"tools",
		"other",

		// // additional types
		// "bank",
		// "payable",
		// "bonus_salary",
	}
}

func (t ExpenseType) NeedAdminPermission() bool {
	values := []ExpenseType{
		ExpenseTypeEquity,
		ExpenseTypeBasicSalary,
		ExpenseTypeServer,
	}
	return slices.Contains(values, t)
}

type ListExpenseType []ExpenseType

type GroupExpenseType map[db_models.TeamType]ListExpenseType

var GroupExpense GroupExpenseType = GroupExpenseType{
	db_models.WarehouseTeamType: ListExpenseType{
		ExpenseTypePettyCash,
		ExpenseTypeTransport,
		ExpenseTypeReceivable,
		ExpenseTypeInternet,
		ExpenseTypePacking,
		ExpenseTypeShipping,
		ExpenseTypeElectricity,
		ExpenseTypeKitchen,
		ExpenseTypeEquipment,
		ExpenseTypeTools,
		ExpenseTypeOther,
	},
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

	Account *WareExpenseAccount `json:"account,omitempty" gorm:"foreignKey:AccountID"`
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
	AccountID   uint `json:"account_id" gorm:"index:ware_account_at,unique"`
	CreatedByID uint `json:"created_by_id"`

	Amount    float64   `json:"amount"`
	At        time.Time `json:"at" gorm:"index:ware_account_at,unique"` // per day
	CreatedAt time.Time `json:"created_at"`

	Account *WareExpenseAccount `json:"account,omitempty" gorm:"foreignkey:AccountID;"`
}
