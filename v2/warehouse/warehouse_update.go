package warehouse

import (
	"context"

	"connectrpc.com/connect"
	"github.com/pdcgo/schema/services/warehouse_iface/v1"
	"github.com/pdcgo/shared/db_models"
)

// WarehouseUpdate implements [warehouse_ifaceconnect.WarehouseServiceHandler].
func (w *warehouseServiceImpl) WarehouseUpdate(
	ctx context.Context,
	req *connect.Request[warehouse_iface.WarehouseUpdateRequest],
) (*connect.Response[warehouse_iface.WarehouseUpdateResponse], error) {
	pay := req.Msg
	db := w.db.WithContext(ctx)

	err := db.
		Model(&db_models.Warehouse{}).
		Where("id = ?", pay.Id).
		Updates(map[string]interface{}{
			"name":          pay.Name,
			"desc":          pay.Desc,
			"address":       pay.Address,
			"is_full":       pay.IsFull,
			"is_closed":     pay.IsClosed,
			"use_fixed_fee": pay.UseFixedFee,
			"fee_fix":       pay.FeeFix,
			"fee_percent":   pay.FeePercent,
			"max_fee":       pay.MaxFee,
			"open_time":     parseHHMM(pay.OpenTime),
			"close_time":    parseHHMM(pay.CloseTime),
			"close_order":   parseHHMM(pay.CloseOrder),
		}).
		Error
	if err != nil {
		return nil, err
	}

	return connect.NewResponse(&warehouse_iface.WarehouseUpdateResponse{}), nil
}
