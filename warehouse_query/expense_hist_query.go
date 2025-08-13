package warehouse_query

import (
	"time"

	"github.com/pdcgo/warehouse_service/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func NewWarehouseExpenseQuery(tx *gorm.DB, lock bool) WarehouseExpenseQuery {
	sqlQuery := tx.Model(&models.WareExpenseHistory{})
	if lock {
		sqlQuery = sqlQuery.Clauses(clause.Locking{
			Strength: "UPDATE",
			Options:  "NOWAIT",
		})
	}

	return &warehouseExpenseQueryImpl{
		tx: sqlQuery,
	}
}

func WarehouseExpenseFromQuery(tx *gorm.DB) WarehouseExpenseQuery {
	return &warehouseExpenseQueryImpl{
		tx: tx,
	}
}

type WarehouseExpenseQuery interface {
	WithQuery(tx *gorm.DB) WarehouseExpenseQuery
	WithHistID(histID uint) WarehouseExpenseQuery
	FromWarehouse(warehouseID uint) WarehouseExpenseQuery
	FromAccount(accountID uint) WarehouseExpenseQuery
	CreatedBy(userID uint) WarehouseExpenseQuery
	WithType(expenseType models.ExpenseType) WarehouseExpenseQuery
	WithTypes(expenseTypes []models.ExpenseType) WarehouseExpenseQuery
	CreatedTime(timeMin, timeMax time.Time) WarehouseExpenseQuery
	ExpenseAt(timeMin, timeMax time.Time) WarehouseExpenseQuery
	FlowType(flowType FlowType) WarehouseExpenseQuery
	GetQuery() *gorm.DB
}

type FlowType string

const (
	FlowTypeIncome  FlowType = "income"
	FlowTypeOutcome FlowType = "outcome"
)

func (FlowType) EnumList() []string {
	return []string{
		"income",
		"outcome",
	}
}

type warehouseExpenseQueryImpl struct {
	tx *gorm.DB
}

// WithQuery implements WarehouseExpenseQuery.
func (w *warehouseExpenseQueryImpl) WithQuery(tx *gorm.DB) WarehouseExpenseQuery {
	w.tx = tx
	return w
}

// getQuery implements WarehouseExpenseQuery.
func (w *warehouseExpenseQueryImpl) GetQuery() *gorm.DB {
	return w.tx
}

func (w *warehouseExpenseQueryImpl) WithHistID(histID uint) WarehouseExpenseQuery {
	if histID != 0 {
		w.tx = w.tx.Where("ware_expense_histories.id = ?", histID)
	}
	return w
}

func (w *warehouseExpenseQueryImpl) FromWarehouse(warehouseID uint) WarehouseExpenseQuery {
	if warehouseID != 0 {
		w.tx = w.tx.Where("ware_expense_histories.warehouse_id = ?", warehouseID)
	}
	return w
}

func (w *warehouseExpenseQueryImpl) FromAccount(accountID uint) WarehouseExpenseQuery {
	if accountID != 0 {
		w.tx = w.tx.Where("ware_expense_histories.account_id = ?", accountID)
	}
	return w
}

func (w *warehouseExpenseQueryImpl) CreatedBy(userID uint) WarehouseExpenseQuery {
	if userID != 0 {
		w.tx = w.tx.Where("ware_expense_histories.created_by_id = ?", userID)
	}
	return w
}

func (w *warehouseExpenseQueryImpl) WithType(expenseType models.ExpenseType) WarehouseExpenseQuery {
	if expenseType != "" {
		w.tx = w.tx.Where("ware_expense_histories.expense_type = ?", expenseType)
	}
	return w
}
func (w *warehouseExpenseQueryImpl) WithTypes(expenseTypes []models.ExpenseType) WarehouseExpenseQuery {
	if len(expenseTypes) != 0 {
		w.tx = w.tx.Where("ware_expense_histories.expense_type IN (?)", expenseTypes)
	}
	return w
}

func (w *warehouseExpenseQueryImpl) CreatedTime(timeMin, timeMax time.Time) WarehouseExpenseQuery {
	if !timeMin.IsZero() {
		w.tx = w.tx.Where("ware_expense_histories.created_at >= ?", timeMin)
	}
	if !timeMax.IsZero() {
		w.tx = w.tx.Where("ware_expense_histories.created_at <= ?", timeMax)
	}
	return w
}

func (w *warehouseExpenseQueryImpl) ExpenseAt(timeMin, timeMax time.Time) WarehouseExpenseQuery {
	if !timeMin.IsZero() {
		w.tx = w.tx.Where("ware_expense_histories.at >= ?", timeMin)
	}
	if !timeMax.IsZero() {
		w.tx = w.tx.Where("ware_expense_histories.at <= ?", timeMax)
	}
	return w
}

func (w *warehouseExpenseQueryImpl) FlowType(flowType FlowType) WarehouseExpenseQuery {
	if flowType == "" {
		return w
	}
	switch flowType {
	case FlowTypeIncome:
		w.tx = w.tx.Where("ware_expense_histories >= 0")
	case FlowTypeOutcome:
		w.tx = w.tx.Where("ware_expense_histories < 0")
	}

	return w
}
