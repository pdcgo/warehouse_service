package inbound

import (
	"context"

	"connectrpc.com/connect"
	"github.com/pdcgo/schema/services/warehouse_iface/v1"
)

// InboundAccept implements warehouse_ifaceconnect.InboundServiceHandler.
func (i *inboundServiceImpl) InboundAccept(
	ctx context.Context,
	req *connect.Request[warehouse_iface.InboundAcceptRequest],
) (*connect.Response[warehouse_iface.InboundAcceptResponse], error) {
	panic("unimplemented")
}
