package warehouse

import (
	"context"

	"connectrpc.com/connect"
	"github.com/pdcgo/schema/services/warehouse_iface/v1"
)

// Stat implements [warehouse_ifaceconnect.WarehouseServiceHandler].
func (w *warehouseServiceImpl) Stat(context.Context, *connect.Request[warehouse_iface.StatRequest]) (*connect.Response[warehouse_iface.StatResponse], error) {
	panic("unimplemented")
}
