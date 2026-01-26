package inventory

import (
	"context"
	"errors"
	"fmt"

	"connectrpc.com/connect"
	"github.com/pdcgo/schema/services/access_iface/v1"
	"github.com/pdcgo/schema/services/warehouse_iface/v1"
	"github.com/pdcgo/shared/authorization"
	"github.com/pdcgo/shared/custom_connect"
	"github.com/pdcgo/shared/db_connect"
	"github.com/pdcgo/shared/db_models"
	"github.com/pdcgo/shared/interfaces/authorization_iface"
	"gorm.io/gorm"
)

// PlacementsIDs implements warehouse_ifaceconnect.InventoryServiceHandler.
func (i *inventoryServiceImpl) PlacementsIDs(
	ctx context.Context,
	req *connect.Request[warehouse_iface.PlacementsIDsRequest],
) (*connect.Response[warehouse_iface.PlacementsIDsResponse], error) {
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

	result := warehouse_iface.PlacementsIDsResponse{
		BulkPlacements: map[uint64]*warehouse_iface.BulkPlacement{},
	}

	// check warehouse id
	var warehouseIDs []uint64
	err = db.Model(&db_models.InvTransaction{}).Where("id in ?", pay.TxIds).Select("warehouse_id").Find(&warehouseIDs).Error
	if err != nil {
		return nil, err
	}
	for _, warehouseID := range warehouseIDs {
		if warehouseID != source.TeamId {
			return nil, errors.New("warehouse access error")
		}
	}

	// getting placement bulk
	invHistMap := map[uint64][]*db_models.InvertoryHistory{}
	rackMap := map[uint64]*db_models.Rack{}
	variantMap := map[uint64]*db_models.VariationValue{}
	skuMap := map[string]*db_models.Sku{}
	variantIDs := []uint{}
	rackIDs := []uint{}
	skuIDs := []string{}

	_, err = db_connect.NewQueryChain(db,
		func(db *gorm.DB, next db_connect.NextFunc) db_connect.NextFunc { // preloading histories
			return func(query *gorm.DB) (*gorm.DB, error) {

				hists := []*db_models.InvertoryHistory{}
				err = db.
					Model(&db_models.InvertoryHistory{}).
					Where("tx_id in ?", pay.TxIds).
					Find(&hists).
					Error

				if err != nil {
					return nil, err
				}

				for _, hist := range hists {
					txId := uint64(*hist.TxID)
					if invHistMap[txId] == nil {
						invHistMap[txId] = []*db_models.InvertoryHistory{}
					}
					invHistMap[txId] = append(invHistMap[txId], hist)
					skuData, err := hist.SkuID.Extract()
					if err != nil {
						return nil, err
					}
					variantIDs = append(variantIDs, uint(skuData.VariantID))
					rackIDs = append(rackIDs, hist.RackID)
					skuIDs = append(skuIDs, string(hist.SkuID))
				}

				return next(query)
			}
		},
		func(db *gorm.DB, next db_connect.NextFunc) db_connect.NextFunc { // getting sku
			return func(query *gorm.DB) (*gorm.DB, error) {
				if len(skuIDs) == 0 {
					return next(query)
				}

				skuIDs = Unique(skuIDs)

				skus := []*db_models.Sku{}
				err = db.
					Model(&db_models.Sku{}).
					Where("id in ?", skuIDs).
					Find(&skus).
					Error

				if err != nil {
					return nil, err
				}

				for _, sku := range skus {
					skuMap[string(sku.ID)] = sku
				}

				return next(query)
			}
		},
		func(db *gorm.DB, next db_connect.NextFunc) db_connect.NextFunc { // getting rack
			return func(query *gorm.DB) (*gorm.DB, error) {
				if len(rackIDs) == 0 {
					return next(query)
				}

				rackIDs = Unique(rackIDs)

				racks := []*db_models.Rack{}
				err = db.
					Model(&db_models.Rack{}).
					Where("id in ?", rackIDs).
					Find(&racks).
					Error

				if err != nil {
					return nil, err
				}

				for _, rack := range racks {
					rackMap[uint64(rack.ID)] = rack
				}

				return next(query)
			}
		},
		func(db *gorm.DB, next db_connect.NextFunc) db_connect.NextFunc { // getting variant
			return func(query *gorm.DB) (*gorm.DB, error) {
				if len(variantIDs) == 0 {
					return next(query)
				}

				variantIDs = Unique(variantIDs)

				variants := []*db_models.VariationValue{}

				err = db.
					Model(&db_models.VariationValue{}).
					Preload("Product").
					Where("id in ?", variantIDs).
					Find(&variants).
					Error

				if err != nil {
					return nil, err
				}

				for _, variant := range variants {
					variantMap[uint64(variant.ID)] = variant
				}

				return next(query)
			}
		},
	)
	if err != nil {
		return nil, err
	}

	// add to result
	for txID, hists := range invHistMap {
		result.BulkPlacements[txID] = &warehouse_iface.BulkPlacement{}
		result.BulkPlacements[txID].Data, err = histToPlacementDetail(rackMap, skuMap, variantMap, hists)

		if err != nil {
			return nil, err
		}
	}

	// old ---------------------------

	return connect.NewResponse(&result), nil
}

func Unique[A comparable](input []A) []A {
	seen := make(map[A]bool) // A map to track seen elements.
	var result []A           // A slice to store the unique elements.

	for _, v := range input {
		if !seen[v] {
			seen[v] = true
			result = append(result, v)
		}
	}
	return result
}

func histToPlacementDetail(
	rackMap map[uint64]*db_models.Rack,
	skuMap map[string]*db_models.Sku,
	variantMap map[uint64]*db_models.VariationValue,
	hists []*db_models.InvertoryHistory) (map[string]*warehouse_iface.PlacementDetail, error) {
	res := map[string]*warehouse_iface.PlacementDetail{}

	for _, hist := range hists {
		if res[string(hist.SkuID)] != nil {
			continue
		}
		sku := skuMap[string(hist.SkuID)]
		if sku == nil {
			return nil, fmt.Errorf("sku empty %s", hist.SkuID)

		}

		variant := variantMap[uint64(sku.VariantID)]
		if variant == nil {
			return nil, fmt.Errorf("variant empty %d", sku.VariantID)
		}

		res[string(hist.SkuID)] = &warehouse_iface.PlacementDetail{
			Racks: []*warehouse_iface.RackPlacement{},
			SkuDetail: &warehouse_iface.SkuDataDetail{
				ProductId:   uint64(sku.VariantID),
				VariantId:   uint64(sku.VariantID),
				WarehouseId: uint64(sku.WarehouseID),
				TeamId:      uint64(sku.TeamID),
			},
			VariantDetail: &warehouse_iface.PlacementVariantDetail{
				Id:           uint64(sku.VariantID),
				Name:         variant.Product.Name,
				Image:        variant.Product.Image[0],
				VariantRefId: string(variant.RefID),
			},
		}
	}

	for _, hist := range hists {
		rack := rackMap[uint64(hist.RackID)]
		if rack == nil {
			return nil, fmt.Errorf("rack empty %d", hist.RackID)
		}
		res[string(hist.SkuID)].Racks = append(res[string(hist.SkuID)].Racks, &warehouse_iface.RackPlacement{
			RackId:    uint64(hist.RackID),
			ItemCount: int64(hist.Count),
			RackName:  rack.Name,
		})
	}

	return res, nil
}
