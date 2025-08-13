package warehouse_query

import (
	"fmt"
	"strings"

	"github.com/pdcgo/warehouse_service/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func NewWarehouseExpenseAccountQuery(tx *gorm.DB, lock bool) WarehouseExpenseAccountQuery {
	sqlQuery := tx.Model(&models.WareExpenseAccountWarehouse{})
	if lock {
		sqlQuery = sqlQuery.Clauses(clause.Locking{
			Strength: "UPDATE",
			Options:  "NOWAIT",
		})
	}

	return &warehouseExpenseAccountQueryImpl{
		tx: sqlQuery,
	}
}

type WarehouseExpenseAccountQuery interface {
	FromAccount(accountID uint) WarehouseExpenseAccountQuery
	FromWarehouse(warehouseID uint) WarehouseExpenseAccountQuery
	IsOpsAccount(isOpsAccount bool) WarehouseExpenseAccountQuery
	OpsAccount(accountOpsType AccountOperationalType) WarehouseExpenseAccountQuery
	JoinWareExpenseAccount(clauseJoin string) WareExpenseAccountQuery
	GetQuery() *gorm.DB
}

type warehouseExpenseAccountQueryImpl struct {
	tx *gorm.DB

	joinWareExpenseAccount bool
}

// GetQuery implements WarehouseExpenseAccountQuery.
func (w *warehouseExpenseAccountQueryImpl) GetQuery() *gorm.DB {
	return w.tx
}

func (w *warehouseExpenseAccountQueryImpl) FromAccount(accountID uint) WarehouseExpenseAccountQuery {
	if accountID != 0 {
		w.tx = w.tx.Where("ware_expense_account_warehouses.account_id = ?", accountID)
	}
	return w
}

func (w *warehouseExpenseAccountQueryImpl) FromWarehouse(warehouseID uint) WarehouseExpenseAccountQuery {
	if warehouseID != 0 {
		w.tx = w.tx.Where("ware_expense_account_warehouses.warehouse_id = ?", warehouseID)
	}
	return w
}

func (w *warehouseExpenseAccountQueryImpl) IsOpsAccount(isOpsAccount bool) WarehouseExpenseAccountQuery {
	w.tx = w.tx.Where("ware_expense_account_warehouses.is_ops_account = ?", isOpsAccount)
	return w
}

type AccountOperationalType string

const (
	AccountOperationalTypeOps    AccountOperationalType = "ops"
	AccountOperationalTypeNonOps AccountOperationalType = "non_ops"
)

func (AccountOperationalType) EnumList() []string {
	return []string{
		"ops",
		"non_ops",
	}
}
func (w *warehouseExpenseAccountQueryImpl) OpsAccount(accountOpsType AccountOperationalType) WarehouseExpenseAccountQuery {
	switch accountOpsType {
	case AccountOperationalTypeOps:
		return w.IsOpsAccount(true)
	case AccountOperationalTypeNonOps:
		return w.IsOpsAccount(false)
	}
	return w
}

func (w *warehouseExpenseAccountQueryImpl) JoinWareExpenseAccount(clauseJoin string) WareExpenseAccountQuery {
	query := wareExpenseAccountQueryImpl{
		joinWareExpenseAccountWarehouse: true,
	}
	if w.joinWareExpenseAccount {
		query.tx = w.tx
		return &query
	}

	joinQuery := "JOIN ware_expense_accounts ON ware_expense_accounts.id = ware_expense_account_warehouses.account_id"

	clauseJoin, _ = strings.CutSuffix(clauseJoin, "JOIN")
	if clauseJoin != "" {
		joinQuery = fmt.Sprintf("%s %s", clauseJoin, joinQuery)
	}

	w.tx = w.tx.Joins(joinQuery)
	w.joinWareExpenseAccount = true

	query.tx = w.tx
	return &query
}

// #################################################
func NewWareExpenseAccountQuery(tx *gorm.DB, lock bool) WareExpenseAccountQuery {
	if lock {
		tx = tx.Clauses(clause.Locking{
			Strength: "UPDATE",
			Options:  "NOWAIT",
		})
	}
	return &wareExpenseAccountQueryImpl{
		tx: tx.Model(&models.WareExpenseAccount{}),
	}
}

type WareExpenseAccountQuery interface {
	WithAccount(accountID uint) WareExpenseAccountQuery
	WithAccountType(accountTypeID uint) WareExpenseAccountQuery
	SearchName(name string) WareExpenseAccountQuery
	SearchNumberID(numberID string) WareExpenseAccountQuery
	IsDisabled(isDisabled bool) WareExpenseAccountQuery
	Disabled(accountStatus AccountStatus) WareExpenseAccountQuery
	JoinWareExpenseAccountWarehouse(clauseJoin string) WarehouseExpenseAccountQuery
	GetQuery() *gorm.DB
}
type wareExpenseAccountQueryImpl struct {
	tx *gorm.DB

	joinWareExpenseAccountWarehouse bool
}

// GetQuery implements WareExpenseAccountQuery.
func (w *wareExpenseAccountQueryImpl) GetQuery() *gorm.DB {
	return w.tx
}

// WithAccount implements WareExpenseAccountQuery.
func (w *wareExpenseAccountQueryImpl) WithAccount(accountID uint) WareExpenseAccountQuery {
	if accountID != 0 {
		w.tx = w.tx.Where("ware_expense_accounts.id = ?", accountID)
	}
	return w
}

// WithAccountType implements WareExpenseAccountQuery.
func (w *wareExpenseAccountQueryImpl) WithAccountType(accountTypeID uint) WareExpenseAccountQuery {
	if accountTypeID != 0 {
		w.tx = w.tx.Where("ware_expense_accounts.account_type_id = ?", accountTypeID)
	}
	return w
}

// IsDisabled implements WareExpenseAccountQuery.
func (w *wareExpenseAccountQueryImpl) IsDisabled(isDisabled bool) WareExpenseAccountQuery {
	w.tx = w.tx.Where("ware_expense_accounts.is_disabled = ?", isDisabled)
	return w
}

type AccountStatus string

const (
	AccountStatusActive   AccountStatus = "active"
	AccountStatusDisabled AccountStatus = "disabled"
)

func (AccountStatus) EnumList() []string {
	return []string{
		"active",
		"disabled",
	}
}

// Disabled implements WareExpenseAccountQuery.
func (w *wareExpenseAccountQueryImpl) Disabled(accountStatus AccountStatus) WareExpenseAccountQuery {
	switch accountStatus {
	case AccountStatusActive:
		return w.IsDisabled(true)
	case AccountStatusDisabled:
		return w.IsDisabled(false)
	}
	return w
}

// SearchName implements WareExpenseAccountQuery.
func (w *wareExpenseAccountQueryImpl) SearchName(name string) WareExpenseAccountQuery {
	if name != "" {
		w.tx = w.tx.Where("LOWER(ware_expense_accounts.name) LIKE ?", "%"+strings.ToLower(name)+"%")
	}
	return w
}

// SearchNumberID implements WareExpenseAccountQuery.
func (w *wareExpenseAccountQueryImpl) SearchNumberID(numberID string) WareExpenseAccountQuery {
	if numberID != "" {
		w.tx = w.tx.Where("LOWER(ware_expense_accounts.number_id) LIKE ?", "%"+strings.ToLower(numberID)+"%")
	}
	return w
}

func (w *wareExpenseAccountQueryImpl) JoinWareExpenseAccountWarehouse(clauseJoin string) WarehouseExpenseAccountQuery {
	query := warehouseExpenseAccountQueryImpl{
		joinWareExpenseAccount: true,
	}

	if w.joinWareExpenseAccountWarehouse {
		query.tx = w.tx
		return &query
	}

	joinQuery := "JOIN ware_expense_account_warehouses ON ware_expense_account_warehouses.account_id = ware_expense_accounts.id"

	clauseJoin, _ = strings.CutSuffix(clauseJoin, "JOIN")
	if clauseJoin != "" {
		joinQuery = fmt.Sprintf("%s %s", clauseJoin, joinQuery)
	}

	w.tx = w.tx.Joins(joinQuery)
	w.joinWareExpenseAccountWarehouse = true

	query.tx = w.tx
	return &query
}
