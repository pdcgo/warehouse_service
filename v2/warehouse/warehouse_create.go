package warehouse

import (
	"context"
	"errors"
	"time"

	"connectrpc.com/connect"
	"github.com/pdcgo/schema/services/warehouse_iface/v1"
	"github.com/pdcgo/shared/db_models"
)

// WarehouseCreate implements [warehouse_ifaceconnect.WarehouseServiceHandler].
// The warehouse id is client-supplied (warehouses.id is not auto-increment); reject a
// duplicate id (mirrors the legacy "you already have a warehouse" guard).
func (w *warehouseServiceImpl) WarehouseCreate(
	ctx context.Context,
	req *connect.Request[warehouse_iface.WarehouseCreateRequest],
) (*connect.Response[warehouse_iface.WarehouseCreateResponse], error) {
	pay := req.Msg
	db := w.db.WithContext(ctx)

	var count int64
	err := db.
		Model(&db_models.Warehouse{}).
		Where("id = ?", pay.Id).
		Count(&count).
		Error
	if err != nil {
		return nil, err
	}
	if count > 0 {
		return nil, connect.NewError(connect.CodeAlreadyExists, errors.New("warehouse id already exists"))
	}

	wh := &db_models.Warehouse{
		ID:            uint(pay.Id),
		Name:          pay.Name,
		Desc:          pay.Desc,
		Address:       pay.Address,
		IsFull:        pay.IsFull,
		IsClosed:      pay.IsClosed,
		UseFixedFee:   pay.UseFixedFee,
		FeeFix:        pay.FeeFix,
		FeePercent:    pay.FeePercent,
		MaxFee:        pay.MaxFee,
		OpenTime:      parseHHMM(pay.OpenTime),
		CloseTime:     parseHHMM(pay.CloseTime),
		CloseOrder:    parseHHMM(pay.CloseOrder),
		Created:       time.Now(),
		WarehouseStat: &db_models.WarehouseStat{},
	}

	err = db.Create(wh).Error
	if err != nil {
		return nil, err
	}

	return connect.NewResponse(&warehouse_iface.WarehouseCreateResponse{Id: uint64(wh.ID)}), nil
}
