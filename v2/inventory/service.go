package inventory

import (
	"github.com/pdcgo/shared/interfaces/authorization_iface"
	"gorm.io/gorm"
)

type TeamInvTransaction struct{}

// GetEntityID implements authorization.Entity.
func (t *TeamInvTransaction) GetEntityID() string {
	return "team_inv_transaction"
}

type inventoryServiceImpl struct {
	db   *gorm.DB
	auth authorization_iface.Authorization
}

func NewInventoryService(
	db *gorm.DB,
	auth authorization_iface.Authorization,
) *inventoryServiceImpl {
	return &inventoryServiceImpl{db, auth}

}
