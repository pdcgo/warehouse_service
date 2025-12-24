package inventory

import (
	"context"
	"errors"

	"connectrpc.com/connect"
	"github.com/pdcgo/gudang/src/authorization"
	"github.com/pdcgo/schema/services/access_iface/v1"
	"github.com/pdcgo/schema/services/warehouse_iface/v1"
	"github.com/pdcgo/shared/custom_connect"
	"github.com/pdcgo/shared/db_models"
	"github.com/pdcgo/shared/interfaces/authorization_iface"
	"github.com/pdcgo/shared/pkg/debugtool"
)

// Placements implements warehouse_ifaceconnect.InventoryServiceHandler.
func (i *inventoryServiceImpl) Placements(ctx context.Context, req *connect.Request[warehouse_iface.PlacementsRequest]) (*connect.Response[warehouse_iface.PlacementsResponse], error) {
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

	var domainID uint
	switch source.RequestFrom {
	case access_iface.RequestFrom_REQUEST_FROM_ADMIN:
		domainID = uint(authorization.RootDomain)
	case access_iface.RequestFrom_REQUEST_FROM_WAREHOUSE:
		domainID = uint(source.TeamId)
	default:
		domainID = uint(source.TeamId)
	}

	err = identity.
		HasPermission(authorization_iface.CheckPermissionGroup{
			&TeamInvTransaction{}: &authorization_iface.CheckPermission{
				DomainID: domainID,
				Actions:  []authorization_iface.Action{authorization_iface.Read},
			},
		}).
		Err()

	if err != nil {
		return nil, err
	}

	db := i.db.WithContext(ctx)
	pay := req.Msg

	result := warehouse_iface.PlacementsResponse{}

	// check warehouse id
	var warehouseID uint64
	err = db.Model(&db_models.InvTransaction{}).Where("id = ?", pay.TxId).Select("warehouse_id").Find(&warehouseID).Error
	if err != nil {
		return nil, err
	}
	if warehouseID != source.TeamId {
		return nil, errors.New("warehouse access error")
	}

	hists := []*db_models.InvertoryHistory{}
	err = db.
		Model(&db_models.InvertoryHistory{}).
		Where("tx_id = ?", pay.TxId).
		Find(&hists).
		Error

	if err != nil {
		return nil, err
	}

	result.Data = map[string]*warehouse_iface.PlacementDetail{}

	rackIDs := []uint{}
	for _, hist := range hists {
		skuData, err := hist.SkuID.Extract()
		if err != nil {
			return nil, err
		}

		result.Data[string(hist.SkuID)] = &warehouse_iface.PlacementDetail{
			Racks: []*warehouse_iface.RackPlacement{},
			SkuDetail: &warehouse_iface.SkuDataDetail{
				ProductId:   uint64(skuData.ProductID),
				VariantId:   uint64(skuData.VariantID),
				WarehouseId: uint64(skuData.WarehouseID),
				TeamId:      uint64(skuData.TeamID),
			},
		}
		rackIDs = append(rackIDs, hist.RackID)

	}

	// preloading rack

	racks := []*db_models.Rack{}
	rackNames := map[uint]string{}

	err = db.
		Model(&db_models.Rack{}).
		Where("id in ?", rackIDs).
		Find(&racks).
		Error

	if err != nil {
		return nil, err
	}

	debugtool.LogJson(racks)

	for _, rack := range racks {
		rackNames[rack.ID] = rack.Name
	}

	for _, hist := range hists {
		racks := result.Data[string(hist.SkuID)].Racks
		racks = append(racks, &warehouse_iface.RackPlacement{
			RackId:    uint64(hist.RackID),
			ItemCount: int64(hist.Count),
			RackName:  rackNames[hist.RackID],
		})
		result.Data[string(hist.SkuID)].Racks = racks
	}

	return connect.NewResponse(&result), nil

}
