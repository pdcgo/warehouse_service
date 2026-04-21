package warehouse_mutations

import (
	"time"

	"github.com/pdcgo/shared/interfaces/identity_iface"
	"github.com/pdcgo/warehouse_service/warehouse_models"
	"github.com/pdcgo/warehouse_service/warehouse_query"
	"gorm.io/gorm"
)

func NewWareBalanceHistMutation(tx *gorm.DB, agent identity_iface.Agent) WareBalanceAccountHistService {
	return &wareBalanceAccountHistImpl{
		tx:    tx,
		agent: agent,
	}
}

type WareBalanceAccountHistService interface {
	Create(accountID uint, amount float64, at time.Time) error
}

type wareBalanceAccountHistImpl struct {
	tx    *gorm.DB
	agent identity_iface.Agent

	account *warehouse_models.WareExpenseAccountWarehouse
	data    *warehouse_models.WareBalanceAccountHistory
}

func (w *wareBalanceAccountHistImpl) Create(accountID uint, amount float64, at time.Time) error {
	var err error
	w.account = &warehouse_models.WareExpenseAccountWarehouse{}
	w.data = &warehouse_models.WareBalanceAccountHistory{}

	accountQuery := warehouse_query.NewWarehouseExpenseAccountQuery(w.tx, false)
	err = accountQuery.
		FromAccount(accountID).
		GetQuery().
		Find(w.account).Error
	if err != nil {
		return err
	}

	balanceQuery := warehouse_query.NewWarehouseBalanceHistQuery(w.tx, false)
	sqlQuery := balanceQuery.
		FromWarehouse(w.account.WarehouseID).
		FromAccount(w.account.AccountID).
		BalanceAt(at).
		GetQuery()

	err = sqlQuery.Find(w.data).Error
	if err != nil {
		return err
	}
	if w.data.ID == 0 { // create if doesn't exist
		w.data = &warehouse_models.WareBalanceAccountHistory{
			WarehouseID: w.account.WarehouseID,
			AccountID:   w.account.AccountID,
			CreatedByID: w.agent.GetUserID(),
			Amount:      amount,
			At:          at,
			CreatedAt:   time.Now(),
		}

		err := w.tx.Create(w.data).Error
		if err != nil {
			return err
		}

		return nil
	}

	err = w.tx.Model(&warehouse_models.WareBalanceAccountHistory{}).
		Where("id = ?", w.data.ID).
		Where("warehouse_id = ?", w.account.WarehouseID).
		Updates(map[string]interface{}{
			"amount": amount,
			"at":     at,
		}).
		Error
	if err != nil {
		return err
	}

	return nil
}
