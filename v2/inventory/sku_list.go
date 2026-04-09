package inventory

import (
	"context"

	"connectrpc.com/connect"
	"github.com/pdcgo/schema/services/common/v1"
	"github.com/pdcgo/schema/services/warehouse_iface/v1"
	"github.com/pdcgo/shared/db_connect"
	"github.com/pdcgo/shared/db_models"
	"github.com/pdcgo/shared/pkg/common_helper"
	"gorm.io/gorm"
)

// SkuList implements [warehouse_ifaceconnect.InventoryServiceHandler].
func (i *inventoryServiceImpl) SkuList(
	ctx context.Context,
	req *connect.Request[warehouse_iface.SkuListRequest],
) (*connect.Response[warehouse_iface.SkuListResponse], error) {
	var err error

	result := &warehouse_iface.SkuListResponse{
		Skus:     []*warehouse_iface.SkuListItem{},
		PageInfo: &common.PageInfo{},
	}

	db := i.db.WithContext(ctx)

	caller := common_helper.NewChainParam(
		func(next common_helper.NextFuncParam[*gorm.DB]) common_helper.NextFuncParam[*gorm.DB] {
			return func(query *gorm.DB) (*gorm.DB, error) { // base query
				rquery := query.
					Table("public.placements p").
					Where("p.rack_id in ?", req.Msg.RackIds).
					Where("p.sku_id = s.id").
					Select("1")

				squery := query.
					Table("public.skus s").
					Where("s.warehouse_id = ?", req.Msg.WarehouseId).
					Where("EXISTS (?)", rquery).
					Where("s.team_id in ?", req.Msg.TeamIds)

				return next(squery)
			}
		},
		func(next common_helper.NextFuncParam[*gorm.DB]) common_helper.NextFuncParam[*gorm.DB] {
			return func(query *gorm.DB) (*gorm.DB, error) { // getting paginated
				var err error
				var paginated *gorm.DB

				paginated, result.PageInfo, err = db_connect.SetPaginationQuery(db, func() (*gorm.DB, error) {
					return query.Session(&gorm.Session{}), nil
				}, req.Msg.Page)

				if err != nil {
					return nil, err
				}

				return next(paginated)
			}
		},
		func(next common_helper.NextFuncParam[*gorm.DB]) common_helper.NextFuncParam[*gorm.DB] {
			return func(query *gorm.DB) (*gorm.DB, error) { // getting data
				err := query.
					Select([]string{
						"s.id as sku_id",
						"s.product_id as product_id",
					}).
					Find(&result.Skus).
					Error

				if err != nil {
					return nil, err
				}

				return next(query)
			}
		},
		func(next common_helper.NextFuncParam[*gorm.DB]) common_helper.NextFuncParam[*gorm.DB] {
			return func(data *gorm.DB) (*gorm.DB, error) { // preloading data product
				productIds := []uint64{}

				productMap := map[uint64]*warehouse_iface.ProductDetail{}

				for i, sku := range result.Skus {
					productIds = append(productIds, sku.ProductId)

					if productMap[sku.ProductId] == nil {
						productMap[sku.ProductId] = &warehouse_iface.ProductDetail{}
					}

					result.Skus[i].ProductDetail = productMap[sku.ProductId]
				}

				prods := []*db_models.Product{}
				err := db.
					Model(&db_models.Product{}).
					Where("id in ?", productIds).
					Find(&prods).Error

				if err != nil {
					return nil, err
				}

				for _, prod := range prods {
					productMap[uint64(prod.ID)].Name = prod.Name
					productMap[uint64(prod.ID)].Image = prod.Image[0]
				}

				return next(data)
			}
		},
	)

	_, err = caller(db)
	return connect.NewResponse(result), err

}
