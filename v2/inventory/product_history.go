package inventory

import (
	"context"

	"connectrpc.com/connect"
	"github.com/pdcgo/schema/services/warehouse_iface/v1"
)

// ProductHistory implements warehouse_ifaceconnect.InventoryServiceHandler.
func (i *inventoryServiceImpl) ProductHistory(ctx context.Context, req *connect.Request[warehouse_iface.ProductHistoryRequest]) (*connect.Response[warehouse_iface.ProductHistoryResponse], error) {
	var err error
	// pay := req.Msg
	result := warehouse_iface.ProductHistoryResponse{}

	// db := i.db.WithContext(ctx)

	return connect.NewResponse(&result), err
}
