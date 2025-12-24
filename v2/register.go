package warehouse_service

import (
	"net/http"

	"github.com/pdcgo/schema/services/warehouse_iface/v1/warehouse_ifaceconnect"
	"github.com/pdcgo/shared/custom_connect"
	"github.com/pdcgo/shared/interfaces/authorization_iface"
	"github.com/pdcgo/warehouse_service/v2/inventory"
	"github.com/pdcgo/warehouse_service/v2/outbound"
	"gorm.io/gorm"
)

type ServiceReflectNames []string
type RegisterHandler func() ServiceReflectNames

func NewRegister(
	db *gorm.DB,
	auth authorization_iface.Authorization,
	mux *http.ServeMux,
	defaultInterceptor custom_connect.DefaultInterceptor,
	// cache ware_cache.Cache,
	// dispather report.ReportDispatcher,
) RegisterHandler {
	return func() ServiceReflectNames {
		grpcReflects := ServiceReflectNames{}

		path, handler := warehouse_ifaceconnect.NewOutboundServiceHandler(
			outbound.NewOutboundService(db, auth),
			defaultInterceptor,
		)
		mux.Handle(path, handler)
		grpcReflects = append(grpcReflects, warehouse_ifaceconnect.OutboundServiceName)

		path, handler = warehouse_ifaceconnect.NewInventoryServiceHandler(
			inventory.NewInventoryService(db, auth),
			defaultInterceptor,
		)
		mux.Handle(path, handler)
		grpcReflects = append(grpcReflects, warehouse_ifaceconnect.InventoryServiceName)

		return grpcReflects
	}
}
