package warehouse_query

import (
	"strings"

	"github.com/pdcgo/warehouse_service/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func NewWarehouseExpenseAccountQuery(tx *gorm.DB, lock bool) WarehouseExpenseAccountQuery {
	sqlQuery := tx.Model(&models.WareExpenseAccountWarehouse{}).
		Joins("JOIN ware_expense_accounts ON ware_expense_accounts.id = ware_expense_account_warehouses.account_id")
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
	IsDisabled(isDisabled bool) WarehouseExpenseAccountQuery
	SearchName(name string) WarehouseExpenseAccountQuery
	SearchNumberID(numberID string) WarehouseExpenseAccountQuery
	GetQuery() *gorm.DB
}

type warehouseExpenseAccountQueryImpl struct {
	tx *gorm.DB
}

// GetQuery implements WarehouseExpenseAccountQuery.
func (w *warehouseExpenseAccountQueryImpl) GetQuery() *gorm.DB {
	return w.tx
}

func (w *warehouseExpenseAccountQueryImpl) FromAccount(accountID uint) WarehouseExpenseAccountQuery {
	if accountID == 0 {
		return w
	}
	w.tx = w.tx.Where("ware_expense_account_warehouses.account_id = ?", accountID)
	return w
}

func (w *warehouseExpenseAccountQueryImpl) FromWarehouse(warehouseID uint) WarehouseExpenseAccountQuery {
	if warehouseID == 0 {
		return w
	}
	w.tx = w.tx.Where("ware_expense_account_warehouses.warehouse_id = ?", warehouseID)
	return w
}

func (w *warehouseExpenseAccountQueryImpl) IsOpsAccount(isOpsAccount bool) WarehouseExpenseAccountQuery {
	w.tx = w.tx.Where("ware_expense_account_warehouses.is_ops_account = ?", isOpsAccount)
	return w
}

func (w *warehouseExpenseAccountQueryImpl) IsDisabled(isDisabled bool) WarehouseExpenseAccountQuery {
	w.tx = w.tx.Where("ware_expense_accounts.is_disabled = ?", isDisabled)
	return w
}

func (w *warehouseExpenseAccountQueryImpl) SearchName(name string) WarehouseExpenseAccountQuery {
	if name == "" {
		return w
	}
	w.tx = w.tx.Where("LOWER(ware_expense_accounts.name) LIKE ?", "%"+strings.ToLower(name)+"%")
	return w
}

func (w *warehouseExpenseAccountQueryImpl) SearchNumberID(numberID string) WarehouseExpenseAccountQuery {
	if numberID == "" {
		return w
	}
	w.tx = w.tx.Where("LOWER(ware_expense_accounts.number_id) LIKE ?", "%"+strings.ToLower(numberID)+"%")
	return w
}
