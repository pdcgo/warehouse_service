package warehouse

import (
	"gorm.io/gorm"
)

type warehouseServiceImpl struct {
	db *gorm.DB
}

func NewWarehouseService(db *gorm.DB) *warehouseServiceImpl {
	return &warehouseServiceImpl{db}
}
