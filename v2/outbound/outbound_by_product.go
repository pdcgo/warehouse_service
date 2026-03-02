package outbound

import (
	"context"
	"errors"

	"connectrpc.com/connect"
	"github.com/pdcgo/schema/services/access_iface/v1"
	"github.com/pdcgo/schema/services/common/v1"
	"github.com/pdcgo/schema/services/warehouse_iface/v1"
	"github.com/pdcgo/shared/custom_connect"
	"github.com/pdcgo/shared/db_connect"
	"github.com/pdcgo/shared/db_models"
	"github.com/pdcgo/shared/interfaces/authorization_iface"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// OutboundByProduct implements warehouse_ifaceconnect.OutboundServiceHandler.
func (o *outboundImpl) OutboundByProduct(
	ctx context.Context,
	req *connect.Request[warehouse_iface.OutboundByProductRequest]) (*connect.Response[warehouse_iface.OutboundByProductResponse], error) {
	var err error

	source, err := custom_connect.GetRequestSource(ctx)
	if err != nil {
		return nil, err
	}

	identity := o.
		auth.
		AuthIdentityFromHeader(req.Header())

	err = identity.Err()
	if err != nil {
		return nil, err
	}

	var domainID uint
	switch source.RequestFrom {
	case access_iface.RequestFrom_REQUEST_FROM_WAREHOUSE:
		domainID = uint(source.TeamId)
	default:
		return nil, errors.New("you re not warehouse")
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

	db := o.db.WithContext(ctx)
	pay := req.Msg
	filter := pay.Filter

	result := warehouse_iface.OutboundByProductResponse{
		Data:     []*warehouse_iface.OutboundByProductItem{},
		PageInfo: &common.PageInfo{},
	}

	_, err = db_connect.NewQueryChain(db,
		func(db *gorm.DB, next db_connect.NextFunc) db_connect.NextFunc {
			return func(query *gorm.DB) (*gorm.DB, error) { // filter warehouse dan team dan shop
				newquery := db.
					Table("orders o").
					Joins("JOIN order_items oi ON oi.order_id = o.id").
					Select([]string{
						"oi.variation_id as variation_id",
						"oi.product_id as product_id",
						"count(oi.order_id) as tx_count",
						"sum(oi.count) as item_count",
					}).
					// Where("status = ?", db_models.OrdCreated).
					Where("o.invertory_tx_id is not null").
					Group("variation_id, product_id")

				if filter.TeamId != 0 {
					newquery = newquery.
						Where("o.team_id = ?", filter.TeamId)
				}

				if filter.ShopId != 0 {
					newquery = newquery.
						Where("o.order_mp_id = ?", filter.ShopId)
				}

				trange := filter.TimeRange
				if trange.StartDate.IsValid() {
					newquery = newquery.
						Where("o.created_at >= ?", trange.StartDate.AsTime())
				}

				if trange.EndDate.IsValid() {
					newquery = newquery.
						Where("o.created_at <= ?", trange.EndDate.AsTime())
				}

				return next(newquery)
			}
		},
		func(db *gorm.DB, next db_connect.NextFunc) db_connect.NextFunc {
			return func(query *gorm.DB) (*gorm.DB, error) { // filtering warehouse id

				if filter.WarehouseId != 0 {
					exists := db.
						Table("inv_transactions it").
						Where("it.id = o.invertory_tx_id").
						Where("it.warehouse_id = ?", filter.WarehouseId)

					if len(filter.TxStatus) == 0 {
						exists = exists.
							Where("it.status = ?", "waiting")
					} else {
						exists = exists.
							Where("it.status in ?", filter.TxStatus)
					}

					query = query.
						Where(
							"EXISTS (?)",
							exists.
								Select("1"),
						)
				}

				return next(query)
			}
		},
		func(db *gorm.DB, next db_connect.NextFunc) db_connect.NextFunc {
			return func(query *gorm.DB) (*gorm.DB, error) { // paginated

				var queryPaginated *gorm.DB
				queryPaginated, result.PageInfo, err = db_connect.SetPaginationQuery(db, func() (*gorm.DB, error) {
					return query.Session(&gorm.Session{}), nil

				}, pay.Page)

				if err != nil {
					return query, err
				}

				return next(
					queryPaginated.
						Order("item_count desc"),
				)
			}
		},
		func(db *gorm.DB, next db_connect.NextFunc) db_connect.NextFunc {
			return func(query *gorm.DB) (*gorm.DB, error) { // getting data

				err = query.
					Find(&result.Data).
					Error

				if err != nil {
					return nil, err
				}

				return next(query)
			}
		},

		func(db *gorm.DB, next db_connect.NextFunc) db_connect.NextFunc {
			return func(query *gorm.DB) (*gorm.DB, error) { // getting sku id
				varIds := []uint64{}

				skuIds := map[uint64]*string{}

				for _, data := range result.Data {
					varIds = append(varIds, data.VariationId)
					data.SkuId = ""
					skuIds[data.VariationId] = &data.SkuId
				}

				if len(varIds) == 0 {
					return next(query)
				}
				temp := []db_models.SkuID{}
				err = db.
					Model(&db_models.Sku{}).
					Where("variant_id in ?", varIds).
					Where("warehouse_id = ?", filter.WarehouseId).
					Select("id").
					Find(&temp).
					Error

				if err != nil {
					return nil, err
				}

				for _, t := range temp {
					sdata, err := t.Extract()
					if err != nil {
						return nil, err
					}
					*skuIds[uint64(sdata.VariantID)] = string(t)

				}

				return next(query)
			}
		},

		func(db *gorm.DB, next db_connect.NextFunc) db_connect.NextFunc {
			return func(query *gorm.DB) (*gorm.DB, error) { // preloading variant
				variantIds := []uint64{}
				variantMap := map[uint64]*warehouse_iface.OutboundByProductItem{}

				for _, dd := range result.Data {
					item := dd
					variantIds = append(variantIds, item.VariationId)
					variantMap[item.VariationId] = item
				}

				datas := []struct {
					VariantId     uint64
					ProductName   string
					Images        datatypes.JSONSlice[string]
					VariantImage  string
					VariantNames  datatypes.JSONSlice[string]
					VariantValues datatypes.JSONSlice[string]
					VariantRefId  string
				}{}

				err = db.
					Table("variation_values vv").
					Joins("left join products p on p.id = vv.product_id").
					Where("vv.id in ?", variantIds).
					Select([]string{
						"p.name as product_name",
						"p.image as images",
						"vv.image as variant_image",
						"vv.variation_name as variant_names",
						"vv.variation_value as variant_values",
						"vv.ref_id as variant_ref_id",
						"vv.id as variant_id",
					}).
					Find(&datas).
					Error

				if err != nil {
					return nil, err
				}

				for _, item := range datas {
					variantMap[item.VariantId].Variant = &warehouse_iface.OutboundByProductItemVariant{
						VariantId:     item.VariantId,
						ProductName:   item.ProductName,
						Images:        item.Images,
						VariantImage:  item.VariantImage,
						VariantNames:  item.VariantNames,
						VariantValues: item.VariantValues,
						VariantRefId:  item.VariantRefId,
					}
				}

				return next(query)
			}
		},
	)

	if err != nil {
		return nil, err
	}

	return connect.NewResponse(&result), nil

}
