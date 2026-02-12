package outbound

import (
	"context"
	"errors"

	"connectrpc.com/connect"
	"github.com/pdcgo/schema/services/access_iface/v1"
	"github.com/pdcgo/schema/services/warehouse_iface/v1"
	"github.com/pdcgo/shared/custom_connect"
	"github.com/pdcgo/shared/db_connect"
	"github.com/pdcgo/shared/db_models"
	"github.com/pdcgo/shared/interfaces/authorization_iface"
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

	result := warehouse_iface.OutboundByProductResponse{
		Data: []*warehouse_iface.OutboundByProductItem{},
	}

	_, err = db_connect.NewQueryChain(db,
		func(db *gorm.DB, next db_connect.NextFunc) db_connect.NextFunc {
			return func(query *gorm.DB) (*gorm.DB, error) { // filter warehouse dan team dan shop
				newquery := db.
					Table("orders o").
					Joins("JOIN order_items oi ON oi.order_id = o.id").
					Select([]string{
						"oi.variation_id as variation_id",
						"count(oi.order_id) as tx_count",
						"sum(oi.count) as item_count",
					}).
					Where("status =  ?", db_models.OrdCreated).
					Group("variation_id").
					Order("item_count desc")

				if pay.TeamId != 0 {
					newquery = newquery.
						Where("o.team_id = ?", pay.TeamId)
				}

				if pay.ShopId != 0 {
					newquery = newquery.
						Where("o.order_mp_id = ?", pay.ShopId)
				}

				return next(newquery)
			}
		},
		func(db *gorm.DB, next db_connect.NextFunc) db_connect.NextFunc {
			return func(query *gorm.DB) (*gorm.DB, error) { // filtering warehouse id

				if pay.WarehouseId != 0 {
					exists := db.
						Table("inv_transactions it").
						Where("it.id = o.invertory_tx_id").
						Where("it.warehouse_id = ?", pay.WarehouseId).
						Select("1")

					query = query.
						Where("EXISTS (?)", exists)
				}

				return next(query)
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
	)

	if err != nil {
		return nil, err
	}

	return connect.NewResponse(&result), nil

}
