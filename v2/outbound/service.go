package outbound

import (
	"context"

	"connectrpc.com/connect"
	"github.com/pdcgo/schema/services/warehouse_iface/v1"
	"github.com/pdcgo/shared/interfaces/authorization_iface"
	"gorm.io/gorm"
)

type outboundImpl struct {
	db   *gorm.DB
	auth authorization_iface.Authorization
}

// OutboundDetail implements warehouse_ifaceconnect.OutboundServiceHandler.
func (o *outboundImpl) OutboundDetail(context.Context, *connect.Request[warehouse_iface.OutboundDetailRequest]) (*connect.Response[warehouse_iface.OutboundDetailResponse], error) {
	panic("unimplemented")
}

// OrderDetailSearch implements warehouse_ifaceconnect.OutboundServiceHandler.
func (o *outboundImpl) OrderDetailSearch(context.Context, *connect.Request[warehouse_iface.OrderDetailSearchRequest]) (*connect.Response[warehouse_iface.OrderDetailSearchResponse], error) {
	panic("unimplemented")
}

func NewOutboundService(
	db *gorm.DB,
	auth authorization_iface.Authorization,
) *outboundImpl {
	return &outboundImpl{
		db:   db,
		auth: auth,
	}
}
