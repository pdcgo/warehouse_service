package inbound

import (
	"context"

	"connectrpc.com/connect"
	"github.com/pdcgo/schema/services/warehouse_iface/v1"
	"github.com/pdcgo/shared/interfaces/authorization_iface"
	"gorm.io/gorm"
)

type inboundServiceImpl struct {
	db   *gorm.DB
	auth authorization_iface.Authorization
}

// InboundCancel implements warehouse_ifaceconnect.InboundServiceHandler.
func (i *inboundServiceImpl) InboundCancel(context.Context, *connect.Request[warehouse_iface.InboundCancelRequest]) (*connect.Response[warehouse_iface.InboundCancelResponse], error) {
	panic("unimplemented")
}

// InboundDetailSearch implements warehouse_ifaceconnect.InboundServiceHandler.
func (i *inboundServiceImpl) InboundDetailSearch(context.Context, *connect.Request[warehouse_iface.InboundDetailSearchRequest]) (*connect.Response[warehouse_iface.InboundDetailSearchResponse], error) {
	panic("unimplemented")
}

// InboundReject implements warehouse_ifaceconnect.InboundServiceHandler.
func (i *inboundServiceImpl) InboundReject(context.Context, *connect.Request[warehouse_iface.InboundRejectRequest]) (*connect.Response[warehouse_iface.InboundRejectResponse], error) {
	panic("unimplemented")
}

func NewInboundService(db *gorm.DB, auth authorization_iface.Authorization) *inboundServiceImpl {
	return &inboundServiceImpl{db, auth}
}
