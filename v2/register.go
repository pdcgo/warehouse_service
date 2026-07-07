package warehouse_service

import (
	"net/http"

	"connectrpc.com/connect"
	"github.com/pdcgo/san_collection/san_caches"
	"github.com/pdcgo/schema/services/warehouse_iface/v1/warehouse_ifaceconnect"
	"github.com/pdcgo/shared/configs"
	"github.com/pdcgo/shared/custom_connect"
	"github.com/pdcgo/shared/interfaces/authorization_iface"
	"github.com/pdcgo/user_service/access_interceptors"
	"github.com/pdcgo/warehouse_service/v2/inbound"
	"github.com/pdcgo/warehouse_service/v2/inventory"
	"github.com/pdcgo/warehouse_service/v2/outbound"
	"github.com/pdcgo/warehouse_service/v2/warehouse"
	"gorm.io/gorm"
)

type ServiceReflectNames []string
type RegisterHandler func() ServiceReflectNames

func NewRegister(
	db *gorm.DB,
	auth authorization_iface.Authorization,
	mux *http.ServeMux,
	defaultInterceptor custom_connect.DefaultInterceptor,
	pushHandler WarehousePushHttpHandler,
	cfg *configs.AppConfig,
	cacheMgr san_caches.CacheManager,
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

		path, handler = warehouse_ifaceconnect.NewInboundServiceHandler(
			inbound.NewInboundService(db, auth),
			defaultInterceptor,
		)
		mux.Handle(path, handler)
		grpcReflects = append(grpcReflects, warehouse_ifaceconnect.InboundServiceName)

		path, handler = warehouse_ifaceconnect.NewInventoryServiceHandler(
			inventory.NewInventoryService(db, auth),
			defaultInterceptor,
		)
		mux.Handle(path, handler)
		grpcReflects = append(grpcReflects, warehouse_ifaceconnect.InventoryServiceName)

		// v2 roling: enforce the (role_base.v1.request_policy) declared on each
		// WarehouseService request message (admin-only management; reads authenticated;
		// WarehouseIDs public). Per-handler option — only WarehouseService is gated.
		warehouseRoleOpt := connect.WithInterceptors(
			access_interceptors.NewAccessInterceptor(db, cfg.JwtSecret, cacheMgr),
		)
		path, handler = warehouse_ifaceconnect.NewWarehouseServiceHandler(
			warehouse.NewWarehouseService(db),
			defaultInterceptor,
			warehouseRoleOpt,
		)
		mux.Handle(path, handler)
		grpcReflects = append(grpcReflects, warehouse_ifaceconnect.WarehouseServiceName)

		mux.HandleFunc("/push", pushHandler)

		return grpcReflects
	}
}
