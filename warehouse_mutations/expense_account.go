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
	GetByQuery(lock bool, query func(tx *gorm.DB) *gorm.DB) (*models.WareExpenseAccount, error)
	Update(name, numberId string) error
	Warehouse() (*models.WareExpenseAccountWarehouse, error)
	Create(name, numberId string, isOpsAccount bool) (*models.WareExpenseAccount, error)
}

type expenseAccountImpl struct {
	tx *gorm.DB

	warehouseId uint

	data *models.WareExpenseAccount
}

func (e *expenseAccountImpl) GetByQuery(lock bool, query func(tx *gorm.DB) *gorm.DB) (*models.WareExpenseAccount, error) {
	e.data = &models.WareExpenseAccount{}

	tx := e.tx
	if lock {
		tx = tx.Clauses(clause.Locking{
			Strength: "UPDATE",
			Options:  "NOWAIT",
		})
	}
	tx = tx.Model(&models.WareExpenseAccount{}).
		Joins("JOIN ware_expense_account_warehouses ON ware_expense_account_warehouses.account_id = ware_expense_accounts.id").
		Where("ware_expense_account_warehouses.warehouse_id = ?", e.warehouseId)
	if query != nil {
		tx = query(tx)
	}
	err := tx.Find(e.data).Error
	if err != nil {
		return nil, err
	}

	if e.data.ID == 0 {
		return nil, ErrExpenseAccountNotFound
	}

	return e.data, nil
}

func (e *expenseAccountImpl) Get(lock bool) (*models.WareExpenseAccount, error) {

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

	e.data.Name = name
	e.data.NumberID = numberId

	return nil
}

func (e *expenseAccountImpl) Warehouse() (*models.WareExpenseAccountWarehouse, error) {
	if e.data == nil {
		return nil, errors.New("expense account not initialized")
	}

	result := models.WareExpenseAccountWarehouse{}
	err := e.tx.Model(&models.WareExpenseAccountWarehouse{}).
		Where("ware_expense_account_warehouses.account_id = ?", e.data.ID).
		Where("ware_expense_account_warehouses.warehouse_id = ?", e.warehouseId).
		First(&result).Error
	if err != nil {
		return nil, err
	}

	return &result, nil
}

func (e *expenseAccountImpl) Create(name, numberId string, isOpsAccount bool) (*models.WareExpenseAccount, error) {
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

	e.data = &wareExpenseAccount

	return &wareExpenseAccount, nil
}
