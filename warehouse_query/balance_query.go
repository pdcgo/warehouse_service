package warehouse_query

import (
	"fmt"
	"time"

	"github.com/pdcgo/warehouse_service/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func NewWarehouseBalanceHistQuery(tx *gorm.DB, lock bool) WarehouseBalanceHistQuery {
	sqlQuery := tx.Model(&models.WareBalanceAccountHistory{})
	if lock {
		sqlQuery = sqlQuery.Clauses(clause.Locking{
			Strength: "UPDATE",
			Options:  "NOWAIT",
		})
	}

	return &warehouseBalanceHistQueryImpl{
		tx: sqlQuery,
	}
}

type WarehouseBalanceHistQuery interface {
	WithHistID(histID uint) WarehouseBalanceHistQuery
	FromWarehouse(warehouseID uint) WarehouseBalanceHistQuery
	FromAccount(accountID uint) WarehouseBalanceHistQuery
	CreatedBy(userID uint) WarehouseBalanceHistQuery
	BalanceAt(at *time.Time) WarehouseBalanceHistQuery
	CreatedTime(timeMin, timeMax *time.Time) WarehouseBalanceHistQuery
	BalanceTime(timeMin, timeMax *time.Time) WarehouseBalanceHistQuery
	GetQuery() *gorm.DB
}

type warehouseBalanceHistQueryImpl struct {
	tx *gorm.DB
}

// GetQuery implements WarehouseBalanceHistQuery.
func (w *warehouseBalanceHistQueryImpl) GetQuery() *gorm.DB {
	return w.tx
}

func (w *warehouseBalanceHistQueryImpl) WithHistID(histID uint) WarehouseBalanceHistQuery {
	if histID == 0 {
		return w
	}
	w.tx = w.tx.Where("ware_balance_account_histories.id = ?", histID)
	return w
}

func (w *warehouseBalanceHistQueryImpl) FromWarehouse(warehouseID uint) WarehouseBalanceHistQuery {
	if warehouseID == 0 {
		return w
	}
	w.tx = w.tx.Where("ware_balance_account_histories.warehouse_id = ?", warehouseID)
	return w
}

func (w *warehouseBalanceHistQueryImpl) FromAccount(accountID uint) WarehouseBalanceHistQuery {
	if accountID == 0 {
		return w
	}
	w.tx = w.tx.Where("ware_balance_account_histories.account_id = ?", accountID)
	return w
}

func (w *warehouseBalanceHistQueryImpl) CreatedBy(userID uint) WarehouseBalanceHistQuery {
	if userID == 0 {
		return w
	}
	w.tx = w.tx.Where("ware_balance_account_histories.created_by_id = ?", userID)
	return w
}

// In Day format
func (w *warehouseBalanceHistQueryImpl) BalanceAt(at *time.Time) WarehouseBalanceHistQuery {
	if at == nil {
		return w
	}

	field := "DATE(ware_balance_account_histories.at AT TIME ZONE 'Asia/Jakarta')"
	if w.tx.Dialector.Name() == "sqlite" {
		field = "DATE(ware_balance_account_histories.at, 'localtime')"
	}

	w.tx = w.tx.Where(fmt.Sprintf("%s = ?", field), at.Format("2006-01-02"))
	return w
}

func (w *warehouseBalanceHistQueryImpl) CreatedTime(timeMin, timeMax *time.Time) WarehouseBalanceHistQuery {
	if timeMin != nil {
		w.tx = w.tx.Where("ware_balance_account_histories.created_at >= ?", timeMin)
	}
	if timeMax != nil {
		w.tx = w.tx.Where("ware_balance_account_histories.created_at <= ?", timeMax)
	}
	return w
}

func (w *warehouseBalanceHistQueryImpl) BalanceTime(timeMin, timeMax *time.Time) WarehouseBalanceHistQuery {
	if timeMin != nil {
		w.tx = w.tx.Where("ware_balance_account_histories.at >= ?", timeMin)
	}
	if timeMax != nil {
		w.tx = w.tx.Where("ware_balance_account_histories.at <= ?", timeMax)
	}
	return w
}
