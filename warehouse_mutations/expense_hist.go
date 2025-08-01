package warehouse_mutations

import (
	"errors"
	"time"

	"github.com/pdcgo/shared/db_models"
	"github.com/pdcgo/warehouse_service/models"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/gorm"
)

func NewExpenseHistService(tx *gorm.DB, warehouseID uint) ExpenseHist {
	return &expenseHistImpl{
		tx: tx,
		data: &models.WareExpenseHistory{
			WarehouseID: warehouseID,
		},
	}
}

type ExpenseHist interface {
	From(userID uint, teamType db_models.TeamType) ExpenseHist
	WithAccountID(accountID uint) ExpenseHist
	WithExpenseHistID(histID uint) ExpenseHist
	Create(expenseType models.ExpenseType, at *timestamppb.Timestamp, amount float64) error
	Update(expenseType models.ExpenseType, at *timestamppb.Timestamp, amount float64) error
}

type expenseHistImpl struct {
	tx *gorm.DB

	data *models.WareExpenseHistory
	from db_models.TeamType
}

func (e *expenseHistImpl) From(userID uint, teamType db_models.TeamType) ExpenseHist {
	e.from = teamType
	e.data.CreatedByID = userID
	return e
}

func (e *expenseHistImpl) WithAccountID(accountID uint) ExpenseHist {
	e.data.AccountID = accountID
	return e
}

func (e *expenseHistImpl) WithExpenseHistID(histID uint) ExpenseHist {
	e.data.ID = histID
	return e
}

func (e *expenseHistImpl) Create(expenseType models.ExpenseType, at *timestamppb.Timestamp, amount float64) error {
	if e.from == "" {
		err := errors.New("invalid expense created from")
		return err
	}
	if !models.CanCreateExpense[e.from][expenseType] {
		return errors.New("not allowed create expense")
	}

	expense := e.data
	if expense.AccountID == 0 {
		err := errors.New("account_id not initialized")
		return err
	}

	sqlQuery := e.tx.Model(&models.WareExpenseAccountWarehouse{})
	if expense.WarehouseID != 0 {
		sqlQuery = sqlQuery.Where("ware_expense_account_warehouses.warehouse_id = ?", expense.WarehouseID)
	}
	if expense.AccountID != 0 {
		sqlQuery = sqlQuery.Where("ware_expense_account_warehouses.account_id = ?", expense.AccountID)
	}

	account := models.WareExpenseAccountWarehouse{}
	err := sqlQuery.Find(&account).Error
	if err != nil {
		return err
	}
	if account.ID == 0 {
		err := errors.New("account not found")
		return err
	}

	expense.ExpenseType = expenseType
	expense.Amount = amount
	expense.At = at.AsTime()
	expense.CreatedAt = time.Now()

	err = e.tx.Create(&expense).Error
	if err != nil {
		return err
	}

	return nil
}

func (e *expenseHistImpl) Update(expenseType models.ExpenseType, at *timestamppb.Timestamp, amount float64) error {
	if e.data.ID != 0 {
		err := errors.New("warehouse expense hist_id is empty")
		return err
	}

	expense := models.WareExpenseHistory{}
	err := e.tx.Model(&models.WareExpenseHistory{}).
		Where("ware_expense_histories.id = ?", e.data.ID).
		First(&expense).Error
	if err != nil {
		return err
	}

	if e.data.WarehouseID != expense.WarehouseID {
		if e.from != db_models.AdminTeamType {
			err := errors.New("can't change warehouse expense")
			return err
		}
		expense.WarehouseID = e.data.WarehouseID
	}

	if e.data.ExpenseType != expense.ExpenseType {
		if e.data.ExpenseType.NeedAdminPermission() || expense.ExpenseType.NeedAdminPermission() {
			if e.from != db_models.AdminTeamType {
				err := errors.New("need admin permission for update")
				return err
			}

			expense.ExpenseType = e.data.ExpenseType
		}
	}

	expense.ExpenseType = expenseType
	expense.Amount = amount
	expense.At = at.AsTime()
	expense.CreatedAt = time.Now()
	err = e.tx.Save(expense).Error
	if err != nil {
		return err
	}

	return nil
}
