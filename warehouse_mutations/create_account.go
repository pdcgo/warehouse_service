package warehouse_mutations

import (
	"errors"
	"fmt"
	"time"

	"github.com/pdcgo/shared/interfaces/identity_iface"
	"github.com/pdcgo/warehouse_service/models"
	"gorm.io/gorm"
)

func NewCreateWarehouseExpenseAccount(tx *gorm.DB, agent identity_iface.Agent) CreateWarehouseExpenseAccount {
	return &createWarehouseExpenseAccountImpl{
		tx:    tx,
		agent: agent,
	}
}

type CreateWarehouseExpenseAccount interface {
	Create(warehouseID, accountTypeID uint, name, numberID string, isOpsAccount bool) (*models.WareExpenseAccountWarehouse, error)
}

type createWarehouseExpenseAccountImpl struct {
	tx    *gorm.DB
	agent identity_iface.Agent

	data *models.WareExpenseAccountWarehouse
}

func (w *createWarehouseExpenseAccountImpl) Create(warehouseID, accountTypeID uint, name, numberID string, isOpsAccount bool) (*models.WareExpenseAccountWarehouse, error) {
	var err error

	err = w.checkNumberID(warehouseID, numberID)
	if err != nil {
		return nil, err
	}

	if isOpsAccount {
		err = w.checkOpsAccount(warehouseID)
		if err != nil {
			return nil, err
		}
	}

	account := models.WareExpenseAccount{
		AccountTypeID: accountTypeID,
		Name:          name,
		NumberID:      numberID,
		CreatedAt:     time.Now(),
	}
	err = w.tx.Create(&account).Error
	if err != nil {
		return nil, err
	}

	wareExpenseAccountWarehouse := models.WareExpenseAccountWarehouse{
		AccountID:    account.ID,
		WarehouseID:  uint(warehouseID),
		IsOpsAccount: isOpsAccount,
	}
	err = w.tx.Create(&account).Error
	if err != nil {
		return nil, err
	}

	wareExpenseAccountWarehouse.Account = &account
	w.data = &wareExpenseAccountWarehouse

	return w.data, err
}

func (w *createWarehouseExpenseAccountImpl) checkNumberID(warehouseID uint, numberID string) error {
	data := models.WareExpenseAccountWarehouse{}
	err := w.tx.Model(&models.WareExpenseAccountWarehouse{}).
		Joins("JOIN ware_expense_accounts ON ware_expense_accounts.id = ware_expense_account_warehouses.account_id").
		Where("ware_expense_account_warehouses.warehouse_id = ?", warehouseID).
		Where("ware_expense_accounts.number_id = ?", numberID).
		Find(&data).Error
	if err != nil {
		return err
	}
	if data.ID != 0 {
		err := fmt.Errorf("warehouse expense number id %s already registered", numberID)
		return err
	}

	return nil
}

func (w *createWarehouseExpenseAccountImpl) checkOpsAccount(warehouseID uint) error {
	data := models.WareExpenseAccountWarehouse{}
	err := w.tx.Model(&models.WareExpenseAccountWarehouse{}).
		Joins("JOIN ware_expense_accounts ON ware_expense_accounts.id = ware_expense_account_warehouses.account_id").
		Where("ware_expense_account_warehouses.warehouse_id = ?", warehouseID).
		Where("ware_expense_account_warehouses.is_ops_account = ?", true).
		Find(&data).Error
	if err != nil {
		return err
	}
	if data.ID != 0 {
		err := errors.New("warehouse already have operational expense account")
		return err
	}

	return nil
}
