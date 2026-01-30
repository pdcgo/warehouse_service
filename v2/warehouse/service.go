package warehouse

import (
	"github.com/pdcgo/shared/interfaces/authorization_iface"
	"gorm.io/gorm"
)

type warehouseServiceImpl struct {
	db   *gorm.DB
	auth authorization_iface.Authorization
}

func NewWarehouseService(db *gorm.DB, auth authorization_iface.Authorization) *warehouseServiceImpl {
	return &warehouseServiceImpl{db, auth}
}
