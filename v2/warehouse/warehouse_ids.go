package warehouse

import (
	"context"

	"connectrpc.com/connect"
	"github.com/pdcgo/schema/services/warehouse_iface/v1"
	"github.com/pdcgo/shared/db_models"
)

// WarehouseIDs implements warehouse_ifaceconnect.WarehouseServiceHandler.
func (w *warehouseServiceImpl) WarehouseIDs(
	ctx context.Context,
	req *connect.Request[warehouse_iface.WarehouseIDsRequest],
) (*connect.Response[warehouse_iface.WarehouseIDsResponse], error) {
	var err error

	db := w.db.WithContext(ctx)
	pay := req.Msg

	result := warehouse_iface.WarehouseIDsResponse{
		Data: map[uint64]*warehouse_iface.Warehouse{},
	}

	list := []*warehouse_iface.Warehouse{}

	err = db.
		Model(&db_models.Warehouse{}).
		Where("id IN ?", pay.Ids).
		Find(&list).
		Error

	for _, item := range list {
		result.Data[item.Id] = item
	}

	return connect.NewResponse(&result), err
}
