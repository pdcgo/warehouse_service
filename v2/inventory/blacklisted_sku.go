package inventory

import (
	"context"

	"connectrpc.com/connect"
	"github.com/pdcgo/schema/services/warehouse_iface/v1"
	"github.com/pdcgo/shared/db_models"
)

// Placements implements warehouse_ifaceconnect.InventoryServiceHandler.
func (i *inventoryServiceImpl) BlacklistedSku(ctx context.Context, req *connect.Request[warehouse_iface.BlacklistedSkuRequest]) (*connect.Response[warehouse_iface.BlacklistedSkuResponse], error) {
	var err error
	identity := i.
		auth.
		AuthIdentityFromHeader(req.Header())

	err = identity.Err()
	if err != nil {
		return nil, err
	}

	db := i.db.WithContext(ctx)
	pay := req.Msg

	result := warehouse_iface.BlacklistedSkuResponse{
		Data: map[string]*warehouse_iface.SkuBlacklistDetail{},
	}
	skus := []*db_models.Sku{}

	err = db.
		Model(&db_models.Sku{}).
		Where("id IN ?", pay.Skus).
		Preload("Variant").
		Find(&skus).
		Error

	if err != nil {
		return nil, err
	}

	for _, sku := range skus {
		result.Data[sku.ID.String()] = &warehouse_iface.SkuBlacklistDetail{
			SkuId:         sku.ID.String(),
			RefId:         sku.Variant.RefID.String(),
			IsBlacklisted: sku.IsBlacklisted,
		}
	}
	return connect.NewResponse(&result), nil

}
