package inventory

import (
	"context"

	"connectrpc.com/connect"
	"github.com/golang/protobuf/proto"
	"github.com/pdcgo/schema/services/warehouse_iface/v1"
	"github.com/pdcgo/shared/db_models"
	"github.com/pdcgo/shared/pkg/common_helper"
	"gorm.io/gorm"
)

// SkuByIDs implements [warehouse_ifaceconnect.InventoryServiceHandler].
func (i *inventoryServiceImpl) SkuByIDs(
	ctx context.Context,
	req *connect.Request[warehouse_iface.SkuByIDsRequest],
) (*connect.Response[warehouse_iface.SkuByIDsResponse], error) {
	var err error
	result := &warehouse_iface.SkuByIDsResponse{
		Skus: map[string]*warehouse_iface.SkuListItem{},
	}

	pay := req.Msg

	db := i.db.WithContext(ctx)

	prodMap := map[uint64]*warehouse_iface.ProductDetail{}

	caller := common_helper.NewChainParam(
		func(next common_helper.NextFuncParam[*gorm.DB]) common_helper.NextFuncParam[*gorm.DB] {
			return func(query *gorm.DB) (*gorm.DB, error) { // getting sku

				res := []*warehouse_iface.SkuListItem{}

				err = query.
					Table("public.skus s").
					Where("s.id in ?", pay.SkuIds).
					Where("s.warehouse_id = ?", pay.WarehouseId).
					Select([]string{
						"s.id as sku_id",
						"s.product_id",
					}).
					Find(&res).
					Error

				if err != nil {
					return query, err
				}

				for _, sku := range res {
					if _, ok := prodMap[sku.ProductId]; !ok {
						prodMap[sku.ProductId] = &warehouse_iface.ProductDetail{}
					}

					sku.ProductDetail = prodMap[sku.ProductId]
					result.Skus[sku.SkuId] = sku
				}

				return next(query)
			}
		},
		func(next common_helper.NextFuncParam[*gorm.DB]) common_helper.NextFuncParam[*gorm.DB] {
			return func(query *gorm.DB) (*gorm.DB, error) { // preload product detail

				prodIds := []uint64{}
				for pid := range prodMap {
					prodIds = append(prodIds, pid)
				}

				res := []*db_models.Product{}

				err = query.
					Model(&db_models.Product{}).
					Where("id in ?", prodIds).
					Select([]string{
						"id",
						"name",
						"image",
					}).
					Find(&res).
					Error

				if err != nil {
					return query, err
				}

				for _, prod := range res {
					p := warehouse_iface.ProductDetail{
						Name:  prod.Name,
						Image: prod.Image[0],
					}

					proto.Merge(prodMap[uint64(prod.ID)], &p)
				}

				return next(query)
			}
		},
	)

	_, err = caller(db)

	return connect.NewResponse(result), err
}
