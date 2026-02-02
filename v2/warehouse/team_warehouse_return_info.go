package warehouse

import (
	"context"

	"connectrpc.com/connect"
	"github.com/pdcgo/schema/services/common/v1"
	"github.com/pdcgo/schema/services/warehouse_iface/v1"
	"github.com/pdcgo/shared/db_models"
)

// TeamWarehouseReturnInfo implements warehouse_ifaceconnect.WarehouseServiceHandler.
func (w *warehouseServiceImpl) TeamWarehouseReturnInfo(
	ctx context.Context,
	req *connect.Request[warehouse_iface.TeamWarehouseReturnInfoRequest]) (*connect.Response[warehouse_iface.TeamWarehouseReturnInfoResponse], error) {
	var err error

	identity := w.auth.AuthIdentityFromHeader(req.Header())

	err = identity.Err()

	if err != nil {
		return nil, err
	}

	result := &warehouse_iface.TeamWarehouseReturnInfoResponse{
		WarehouseReturn: &warehouse_iface.Warehouse{},
		UserReturn:      &common.User{},
	}
	db := w.db.WithContext(ctx)
	pay := req.Msg

	info := db_models.TeamInfo{}

	err = db.
		Model(&db_models.TeamInfo{}).
		Where("team_id = ?", pay.TeamId).
		First(&info).
		Error

	if err != nil {
		return nil, err
	}

	warehouseID := *info.ReturnWarehouseID
	userID := *info.ReturnUserID

	if warehouseID != 0 {
		err = db.
			Model(&db_models.Warehouse{}).
			First(&result.WarehouseReturn, info.ReturnWarehouseID).
			Error

		if err != nil {
			return nil, err
		}
	}

	if userID != 0 {
		err = db.
			Model(&db_models.User{}).
			First(&result.UserReturn, info.ReturnUserID).
			Error

		if err != nil {
			return nil, err
		}
	}

	return connect.NewResponse(result), nil
}
