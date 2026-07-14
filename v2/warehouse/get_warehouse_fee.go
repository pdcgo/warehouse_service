package warehouse

import (
	"context"
	"errors"

	"connectrpc.com/connect"
	"github.com/pdcgo/schema/services/warehouse_iface/v1"
	"github.com/pdcgo/shared/db_models"
)

// GetWarehouseFee implements [warehouse_ifaceconnect.WarehouseServiceHandler]. It
// computes the warehouse's fulfillment fee for an order value using the fee config
// this service already manages (use_fixed_fee/fee_fix/fee_percent/max_fee) via the
// legacy formula `Warehouse.GetWarehouseFee`: flat FeeFix, OR percent-of-value rounded
// up to 100s and capped at MaxFee. Called by the selling v3 OrderService at create;
// the result is frozen onto the order.
func (w *warehouseServiceImpl) GetWarehouseFee(
	ctx context.Context,
	req *connect.Request[warehouse_iface.GetWarehouseFeeRequest],
) (*connect.Response[warehouse_iface.GetWarehouseFeeResponse], error) {
	pay := req.Msg
	db := w.db.WithContext(ctx)

	var wh db_models.Warehouse
	err := db.
		Where("id = ? AND deleted = ?", pay.WarehouseId, false).
		Limit(1).
		Find(&wh).
		Error
	if err != nil {
		return nil, err
	}
	if wh.ID == 0 {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("warehouse not found"))
	}

	fee, err := wh.GetWarehouseFee(pay.OrderValue)
	if err != nil {
		// Misconfigured fee (e.g. fixed-fee mode with an empty amount) — the caller
		// cannot proceed until the warehouse fixes its config.
		return nil, connect.NewError(connect.CodeFailedPrecondition, err)
	}

	return connect.NewResponse(&warehouse_iface.GetWarehouseFeeResponse{Fee: fee}), nil
}
