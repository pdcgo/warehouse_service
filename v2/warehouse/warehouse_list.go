package warehouse

import (
	"context"

	"connectrpc.com/connect"
	"github.com/pdcgo/schema/services/warehouse_iface/v1"
	"github.com/pdcgo/shared/db_models"
)

// WarehouseList implements warehouse_ifaceconnect.WarehouseServiceHandler.
func (w *warehouseServiceImpl) WarehouseList(
	ctx context.Context,
	req *connect.Request[warehouse_iface.WarehouseListRequest],
) (*connect.Response[warehouse_iface.WarehouseListResponse], error) {
	var err error

	identity := w.auth.AuthIdentityFromHeader(req.Header())

	err = identity.Err()

	if err != nil {
		return nil, err
	}

	result := &warehouse_iface.WarehouseListResponse{
		List: []*warehouse_iface.Warehouse{},
	}

	db := w.db.WithContext(ctx)

	err = db.
		Model(&db_models.Warehouse{}).
		Find(&result.List).
		Error

	if err != nil {
		return nil, err
	}

	return connect.NewResponse(result), nil
}
