package warehouse_mutations

import (
	"errors"
	"time"

	"github.com/pdcgo/shared/db_models"
	"github.com/pdcgo/warehouse_service/models"
	"gorm.io/gorm"
)

func NewExpenseHistService(tx *gorm.DB, warehouseID uint) ExpenseHist {
	return &expenseHistImpl{
		tx: tx,
	}
}

type ExpenseHist interface {
	Create(from db_models.TeamType, accountID uint, expenseType models.ExpenseType, amount float64) error
}

type expenseHistImpl struct {
	tx *gorm.DB

	warehouseID uint
}

var CanCreateExpense = map[db_models.TeamType]map[models.ExpenseType]bool{
	db_models.WarehouseTeamType: {
		"kitchen": true,
		"other":   true,
	},
	db_models.AdminTeamType: {
		"bank": true,
	},
}

func (e *expenseHistImpl) Create(from db_models.TeamType, accountID uint, expenseType models.ExpenseType, amount float64) error {
	if !CanCreateExpense[from][expenseType] {
		return errors.New("not allowed create expense")
	}

	expenseHistory := models.WareExpenseHistory{
		WarehouseID: e.warehouseID,
		AccountID:   accountID,
		ExpenseType: expenseType,
		Amount:      amount,
		CreatedAt:   time.Now(),
	}
	err := e.tx.Create(&expenseHistory).Error
	if err != nil {
		return err
	}

	return nil
}
