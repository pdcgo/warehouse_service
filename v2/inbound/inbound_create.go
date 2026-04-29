package inbound

import (
	"context"

	"connectrpc.com/connect"
	"github.com/pdcgo/schema/services/warehouse_iface/v1"
)

// InboundCreate implements [warehouse_ifaceconnect.InboundServiceHandler].
func (i *inboundServiceImpl) InboundCreate(
	ctx context.Context,
	req *connect.Request[warehouse_iface.InboundCreateRequest]) (*connect.Response[warehouse_iface.InboundCreateResponse], error) {
	panic("unimplemented")
}
