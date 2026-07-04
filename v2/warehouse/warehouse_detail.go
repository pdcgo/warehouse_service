package warehouse

import (
	"context"
	"errors"

	"connectrpc.com/connect"
	"github.com/pdcgo/schema/services/warehouse_iface/v1"
	"github.com/pdcgo/shared/db_models"
)

// WarehouseDetail implements [warehouse_ifaceconnect.WarehouseServiceHandler].
func (w *warehouseServiceImpl) WarehouseDetail(
	ctx context.Context,
	req *connect.Request[warehouse_iface.WarehouseDetailRequest],
) (*connect.Response[warehouse_iface.WarehouseDetailResponse], error) {
	pay := req.Msg
	db := w.db.WithContext(ctx)

	var wh db_models.Warehouse
	err := db.
		Where("id = ? AND deleted = ?", pay.Id, false).
		Limit(1).
		Find(&wh).
		Error
	if err != nil {
		return nil, err
	}
	if wh.ID == 0 {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("warehouse not found"))
	}

	detail := &warehouse_iface.WarehouseDetail{
		Id:          uint64(wh.ID),
		Name:        wh.Name,
		Desc:        wh.Desc,
		Address:     wh.Address,
		IsFull:      wh.IsFull,
		IsClosed:    wh.IsClosed,
		UseFixedFee: wh.UseFixedFee,
		FeeFix:      wh.FeeFix,
		FeePercent:  wh.FeePercent,
		MaxFee:      wh.MaxFee,
		OpenTime:    formatHHMM(wh.OpenTime),
		CloseTime:   formatHHMM(wh.CloseTime),
		CloseOrder:  formatHHMM(wh.CloseOrder),
	}
	if wh.WarehouseStat != nil {
		detail.RackCount = uint64(wh.WarehouseStat.RackCount)
		detail.OrderCount = uint64(wh.WarehouseStat.OrderCount)
		detail.Capacity = uint64(wh.WarehouseStat.Capacity)
		detail.MaxCapacity = uint64(wh.WarehouseStat.MaxCapacity)
		detail.ProductCount = uint64(wh.WarehouseStat.ProductCount)
	}

	return connect.NewResponse(&warehouse_iface.WarehouseDetailResponse{Data: detail}), nil
}
