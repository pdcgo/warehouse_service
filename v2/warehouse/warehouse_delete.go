package warehouse

import (
	"context"

	"connectrpc.com/connect"
	"github.com/pdcgo/schema/services/warehouse_iface/v1"
	"github.com/pdcgo/shared/db_models"
)

// WarehouseDelete implements [warehouse_ifaceconnect.WarehouseServiceHandler].
// Soft delete — sets deleted=true (the model has no gorm.DeletedAt; legacy uses a bool).
func (w *warehouseServiceImpl) WarehouseDelete(
	ctx context.Context,
	req *connect.Request[warehouse_iface.WarehouseDeleteRequest],
) (*connect.Response[warehouse_iface.WarehouseDeleteResponse], error) {
	pay := req.Msg
	db := w.db.WithContext(ctx)

	err := db.
		Model(&db_models.Warehouse{}).
		Where("id = ?", pay.Id).
		Update("deleted", true).
		Error
	if err != nil {
		return nil, err
	}

	return connect.NewResponse(&warehouse_iface.WarehouseDeleteResponse{}), nil
}
