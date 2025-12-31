package inventory

import (
	"context"

	"connectrpc.com/connect"
	"github.com/pdcgo/schema/services/warehouse_iface/v1"
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

// ProductDetail implements warehouse_ifaceconnect.InventoryServiceHandler.
func (i *inventoryServiceImpl) ProductDetail(context.Context, *connect.Request[warehouse_iface.ProductDetailRequest]) (*connect.Response[warehouse_iface.ProductDetailResponse], error) {
	panic("unimplemented")
}

// ProductHistory implements warehouse_ifaceconnect.InventoryServiceHandler.
func (i *inventoryServiceImpl) ProductHistory(context.Context, *connect.Request[warehouse_iface.ProductHistoryRequest]) (*connect.Response[warehouse_iface.ProductHistoryResponse], error) {
	panic("unimplemented")
}

// BlacklistedSku implements warehouse_ifaceconnect.InventoryServiceHandler.
func (i *inventoryServiceImpl) BlacklistedSku(context.Context, *connect.Request[warehouse_iface.BlacklistedSkuRequest]) (*connect.Response[warehouse_iface.BlacklistedSkuResponse], error) {
	panic("unimplemented")
}

// BlacklistedSkuAdd implements warehouse_ifaceconnect.InventoryServiceHandler.
func (i *inventoryServiceImpl) BlacklistedSkuAdd(context.Context, *connect.Request[warehouse_iface.BlacklistedSkuAddRequest]) (*connect.Response[warehouse_iface.BlacklistedSkuAddResponse], error) {
	panic("unimplemented")
}

// BlacklistedSkuRemove implements warehouse_ifaceconnect.InventoryServiceHandler.
func (i *inventoryServiceImpl) BlacklistedSkuRemove(context.Context, *connect.Request[warehouse_iface.BlacklistedSkuRemoveRequest]) (*connect.Response[warehouse_iface.BlacklistedSkuRemoveResponse], error) {
	panic("unimplemented")
}

func NewInventoryService(
	db *gorm.DB,
	auth authorization_iface.Authorization,
) *inventoryServiceImpl {
	return &inventoryServiceImpl{db, auth}

}
