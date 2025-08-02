package warehouse_mutations

import (
	"errors"
	"time"

	"github.com/pdcgo/warehouse_service/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func NewExpenseAccountService(tx *gorm.DB, warehouseID uint) ExpenseAccount {
	return &expenseAccountImpl{
		tx:          tx,
		warehouseId: warehouseID,
	}
}

var ErrExpenseAccountNotFound = errors.New("expense account not found")

type ExpenseAccount interface {
	GetByQuery(lock bool, query func(tx *gorm.DB) *gorm.DB) (*models.WareExpenseAccountWarehouse, error)
	Update(name, numberId string) error
	Create(name, numberId string, isOpsAccount bool) (*models.WareExpenseAccountWarehouse, error)
	Disabled(isDisabled bool) error
}

type expenseAccountImpl struct {
	tx *gorm.DB

	warehouseId uint

	data *models.WareExpenseAccountWarehouse
}

func (e *expenseAccountImpl) GetByQuery(lock bool, query func(tx *gorm.DB) *gorm.DB) (*models.WareExpenseAccountWarehouse, error) {
	e.data = &models.WareExpenseAccountWarehouse{}

	tx := e.tx
	if lock {
		tx = tx.Clauses(clause.Locking{
			Strength: "UPDATE",
			Options:  "NOWAIT",
		})
	}
	tx = tx.Model(&models.WareExpenseAccountWarehouse{}).
		Joins("JOIN ware_expense_accounts ON ware_expense_accounts.id = ware_expense_account_warehouses.account_id")
	if e.warehouseId != 0 {
		tx = tx.Where("ware_expense_account_warehouses.warehouse_id = ?", e.warehouseId)
	}
	if query != nil {
		tx = query(tx)
	}
	err := tx.
		Preload("Account").
		Find(e.data).Error
	if err != nil {
		return nil, err
	}

	if e.data.ID == 0 {
		return nil, ErrExpenseAccountNotFound
	}

	return e.data, nil
}

func (e *expenseAccountImpl) Update(name, numberId string) error {
	if e.data == nil {
		return errors.New("expense account not initialized")
	}

	err := e.tx.Model(&models.WareExpenseAccount{}).
		Where("ware_expense_accounts.id = ?", e.data.ID).
		Updates(map[string]interface{}{
			"name":      name,
			"number_id": numberId,
		}).Error
	if err != nil {
		return err
	}

	e.data.Account.Name = name
	e.data.Account.NumberID = numberId

	return nil
}

// Disabled implements ExpenseAccount.
func (e *expenseAccountImpl) Disabled(isDisabled bool) error {
	if e.data == nil {
		return errors.New("expense account not initialized")
	}

	err := e.tx.Model(&models.WareExpenseAccount{}).
		Where("ware_expense_accounts.id = ?", e.data.ID).
		Updates(map[string]interface{}{
			"disabled": isDisabled,
		}).Error
	if err != nil {
		return err
	}

	e.data.Account.Disabled = isDisabled

	return nil
}

func (e *expenseAccountImpl) Create(name, numberId string, isOpsAccount bool) (*models.WareExpenseAccountWarehouse, error) {
	account, err := e.GetByQuery(false, func(tx *gorm.DB) *gorm.DB {
		return tx.
			Where("ware_expense_account_warehouses.warehouse_id = ?", e.warehouseId).
			Where("(ware_expense_accounts.number_id = ?)", numberId)
	})
	if err != nil {
		if !errors.Is(err, ErrExpenseAccountNotFound) {
			return nil, err
		}
	}
	if account != nil {
		return nil, errors.New("expense account already exist")
	}

	if isOpsAccount {
		opsAccount, err := e.GetByQuery(false, func(tx *gorm.DB) *gorm.DB {
			return tx.
				Where("ware_expense_account_warehouses.warehouse_id = ?", e.warehouseId).
				Where("ware_expense_account_warehouses.is_ops_account = ?", isOpsAccount)
		})
		if err != nil {
			if !errors.Is(err, ErrExpenseAccountNotFound) {
				return nil, err
			}
		}
		if opsAccount != nil {
			return nil, errors.New("ops account already exist")
		}
	}

	wareExpenseAccount := models.WareExpenseAccount{
		Name:      name,
		NumberID:  numberId,
		CreatedAt: time.Now(),
	}
	err = e.tx.Create(&wareExpenseAccount).Error
	if err != nil {
		return nil, err
	}

	wareExpenseAccountWarehouse := models.WareExpenseAccountWarehouse{
		AccountID:    wareExpenseAccount.ID,
		WarehouseID:  uint(e.warehouseId),
		IsOpsAccount: isOpsAccount,
	}
	err = e.tx.Create(&wareExpenseAccountWarehouse).Error
	if err != nil {
		return nil, err
	}

	wareExpenseAccountWarehouse.Account = &wareExpenseAccount
	e.data = &wareExpenseAccountWarehouse

	return e.data, nil
}
