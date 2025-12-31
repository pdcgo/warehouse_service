package inventory

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	"github.com/pdcgo/schema/services/warehouse_iface/v1"
	"github.com/pdcgo/shared/custom_connect"
	"github.com/pdcgo/shared/db_models"
)

// Placements implements warehouse_ifaceconnect.InventoryServiceHandler.
func (i *inventoryServiceImpl) BlacklistedSkuAdd(ctx context.Context, req *connect.Request[warehouse_iface.BlacklistedSkuAddRequest]) (*connect.Response[warehouse_iface.BlacklistedSkuAddResponse], error) {
	var err error
	source, err := custom_connect.GetRequestSource(ctx)
	if err != nil {
		return nil, err
	}

	identity := i.
		auth.
		AuthIdentityFromHeader(req.Header())

	err = identity.Err()
	if err != nil {
		return nil, err
	}

	if source.TeamId != 1 {
		return nil, fmt.Errorf("no permission")
	}

	db := i.db.WithContext(ctx)
	pay := req.Msg

	result := warehouse_iface.BlacklistedSkuAddResponse{}

	err = db.
		Model(&db_models.Sku{}).
		Where("id IN ?", pay.Skus).
		Update("is_blacklisted", true).
		Error

	if err != nil {
		return nil, err
	}

	return connect.NewResponse(&result), nil

}
