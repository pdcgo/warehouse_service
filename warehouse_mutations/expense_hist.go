package warehouse_mutations

import (
	"errors"
	"time"

	"github.com/pdcgo/shared/db_models"
	"github.com/pdcgo/shared/interfaces/identity_iface"
	"github.com/pdcgo/warehouse_service/models"
	"gorm.io/gorm"
)

func NewExpenseHistService(tx *gorm.DB, agent identity_iface.Agent) ExpenseHist {
	return &expenseHistImpl{
		tx:    tx,
		agent: agent,
	}
}

type ExpenseHist interface {
	GetAccount(accountID, warehouseID uint) (*models.WareExpenseAccountWarehouse, error)
	Create(from db_models.TeamType, payload *CreateExpensePayload) error
	GetExpense(expenseHistID uint) (*models.WareExpenseHistory, error)
	Update(from db_models.TeamType, payload *UpdateWareExpenseHistPayload) error
}

type expenseHistImpl struct {
	tx    *gorm.DB
	agent identity_iface.Agent

	account *models.WareExpenseAccountWarehouse
	data    *models.WareExpenseHistory
}

func (e *expenseHistImpl) GetAccount(accountID, warehouseID uint) (*models.WareExpenseAccountWarehouse, error) {

	sqlQuery := e.tx.Model(&models.WareExpenseAccountWarehouse{})
	if accountID != 0 {
		sqlQuery = sqlQuery.Where("ware_expense_account_warehouses.account_id = ?", accountID)
	}
	if warehouseID != 0 {
		sqlQuery = sqlQuery.Where("ware_expense_account_warehouses.warehouse_id = ?", warehouseID)
	}
	err := sqlQuery.
		Preload("Account").
		Find(&e.account).Error
	if err != nil {
		return nil, err
	}
	if e.account.ID == 0 {
		err := errors.New("account not found")
		return nil, err
	}

	return e.account, nil
}

type CreateExpensePayload struct {
	ExpenseType models.ExpenseType
	At          time.Time
	Amount      float64
	Note        string
}

func (e *expenseHistImpl) Create(from db_models.TeamType, payload *CreateExpensePayload) error {
	if e.account == nil {
		return errors.New("account not initialized")
	}
	if from != db_models.AdminTeamType {
		if !models.CanCreateExpense[from][payload.ExpenseType] {
			return errors.New("not allowed create expense")
		}
	}

	expense := models.WareExpenseHistory{
		WarehouseID: e.account.WarehouseID,
		AccountID:   e.account.AccountID,
		CreatedByID: e.agent.GetUserID(),
		ExpenseType: payload.ExpenseType,
		Amount:      payload.Amount,
		Note:        payload.Note,
		At:          payload.At,
		CreatedAt:   time.Now(),
	}
	err := e.tx.Create(expense).Error
	if err != nil {
		return err
	}

	e.data = &expense

	return nil
}

func (e *expenseHistImpl) GetExpense(expenseHistID uint) (*models.WareExpenseHistory, error) {
	err := e.tx.Model(&models.WareExpenseHistory{}).
		Where("ware_expense_histories.id = ?", expenseHistID).
		First(&e.data).Error
	if err != nil {
		return nil, err
	}

	return e.data, nil
}

type UpdateWareExpenseHistPayload struct {
	WarehouseID uint               `json:"warehouse_id"`
	AccountID   uint               `json:"account_id"`
	CreatedByID uint               `json:"created_by_id"`
	ExpenseType models.ExpenseType `json:"expense_type"`
	Amount      float64            `json:"amount"`
	Note        string             `json:"note"`
	At          time.Time          `json:"at"`
}

func (e *expenseHistImpl) Update(from db_models.TeamType, payload *UpdateWareExpenseHistPayload) error {
	if e.data == nil {
		return errors.New("expense data not initialized")
	}

	if e.data.WarehouseID != payload.WarehouseID {
		if from != db_models.AdminTeamType {
			err := errors.New("can't change warehouse expense")
			return err
		}
		e.data.WarehouseID = payload.WarehouseID
	}

	if e.data.ExpenseType != payload.ExpenseType {
		if e.data.ExpenseType.NeedAdminPermission() || payload.ExpenseType.NeedAdminPermission() {
			if from != db_models.AdminTeamType {
				err := errors.New("need admin permission for update")
				return err
			}

			e.data.ExpenseType = payload.ExpenseType
		}
	}

	e.data.AccountID = payload.AccountID
	e.data.CreatedByID = e.agent.GetUserID()
	e.data.Note = payload.Note
	e.data.At = payload.At
	e.data.Amount = payload.Amount
	err := e.tx.Save(e.data).Error
	if err != nil {
		return err
	}

	return nil
}
