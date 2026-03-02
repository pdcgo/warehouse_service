package outbound

import (
	"context"
	"errors"
	"strings"

	"connectrpc.com/connect"
	"github.com/pdcgo/schema/services/access_iface/v1"
	"github.com/pdcgo/schema/services/common/v1"
	"github.com/pdcgo/schema/services/warehouse_iface/v1"
	"github.com/pdcgo/shared/authorization"
	"github.com/pdcgo/shared/custom_connect"
	"github.com/pdcgo/shared/db_connect"
	"github.com/pdcgo/shared/db_models"
	"github.com/pdcgo/shared/interfaces/authorization_iface"
	"github.com/pdcgo/shared/pkg/common_helper"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/gorm"
)

type TeamInvTransaction struct{}

// GetEntityID implements authorization.Entity.
func (t *TeamInvTransaction) GetEntityID() string {
	return "team_inv_transaction"
}

// OutboundList implements warehouse_ifaceconnect.OutboundServiceHandler.
func (o *outboundImpl) OutboundList(
	ctx context.Context,
	req *connect.Request[warehouse_iface.OutboundListRequest],
) (*connect.Response[warehouse_iface.OutboundListResponse], error) {
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

	payload := req.Msg
	paySort := payload.Sort
	db := o.
		db.
		WithContext(ctx)

	result := warehouse_iface.OutboundListResponse{
		Data:     []*warehouse_iface.Outbound{},
		PageInfo: &common.PageInfo{},
	}

	filter := payload.Filter

	orderQuery := common_helper.NewChainParam(
		func(next common_helper.NextFuncParam[*gorm.DB]) common_helper.NextFuncParam[*gorm.DB] {
			return func(query *gorm.DB) (*gorm.DB, error) {

				query = query.
					Table("orders o")

				if filter.ShopId != 0 {
					query = query.
						Where("o.order_mp_id = ?", filter.ShopId)
				} else {
					query = query.
						Joins("JOIN marketplaces mp ON mp.id = o.order_mp_id").
						Where("o.invertory_tx_id = it.id")

					if len(filter.Marketplaces) != 0 {
						mpstring := make([]string, len(filter.Marketplaces))
						for i, mp := range filter.Marketplaces {
							switch mp {
							case common.MarketplaceType_MARKETPLACE_TYPE_SHOPEE:
								mpstring[i] = "shopee"
							case common.MarketplaceType_MARKETPLACE_TYPE_TIKTOK:
								mpstring[i] = "tiktok"
							case common.MarketplaceType_MARKETPLACE_TYPE_LAZADA:
								mpstring[i] = "lazada"
							case common.MarketplaceType_MARKETPLACE_TYPE_CUSTOM:
								mpstring[i] = "custom"
							case common.MarketplaceType_MARKETPLACE_TYPE_MENGANTAR:
								mpstring[i] = "mengantar"
							case common.MarketplaceType_MARKETPLACE_TYPE_TOKOPEDIA:
								mpstring[i] = "tokopedia"
							default:
								return query, errors.New("invalid marketplace type")
							}
						}

						query = query.
							Where("mp.mp_type IN ?", mpstring)

					}

					if filter.Q != "" {

						switch filter.SearchType {
						case warehouse_iface.OutboundSearchType_OUTBOUND_SEARCH_TYPE_SHOPNAME:
							q := "%" + strings.ToLower(filter.Q) + "%"
							query = query.
								Where("(mp.mp_name ilike ?) or (mp.mp_username ilike ?)", q, q)
						}
					}
				}

				return next(query)
			}
		},
	)

	_, err = db_connect.NewQueryChain(
		db,
		func(db *gorm.DB, next db_connect.NextFunc) db_connect.NextFunc {
			return func(query *gorm.DB) (*gorm.DB, error) { // base
				query = query.
					Table("inv_transactions it")

				switch source.RequestFrom {
				case access_iface.RequestFrom_REQUEST_FROM_WAREHOUSE:
					query = query.
						Where("it.warehouse_id = ?", source.TeamId)
				default:
					return nil, errors.New("query unsupported")
				}

				return next(query)
			}
		},
		func(db *gorm.DB, next db_connect.NextFunc) db_connect.NextFunc {
			return func(query *gorm.DB) (*gorm.DB, error) { // outbound type

				if filter.OutboundType != warehouse_iface.OutboundType_OUTBOUND_TYPE_UNSPECIFIED {
					switch filter.OutboundType {
					case warehouse_iface.OutboundType_OUTBOUND_TYPE_TRANSFER_OUT:
						query = query.
							Where("it.type = ?", db_models.InvTxTransferOut)
					case warehouse_iface.OutboundType_OUTBOUND_TYPE_ORDER:
						query = query.
							Where("it.type = ?", db_models.InvTxOrder)
					}
				} else {
					query = query.
						Where("it.type in ?", []db_models.InvTxType{
							db_models.InvTxAdjout,
							db_models.InvTxOrder,
							db_models.InvTxTransferOut,
						})
				}

				return next(query)
			}
		},
		func(db *gorm.DB, next db_connect.NextFunc) db_connect.NextFunc {
			return func(query *gorm.DB) (*gorm.DB, error) { // filter delete
				if !filter.IncludeDeleted {
					query = query.
						Where("it.deleted != true")
				}

				return next(query)
			}
		},
		func(db *gorm.DB, next db_connect.NextFunc) db_connect.NextFunc {
			return func(query *gorm.DB) (*gorm.DB, error) { // team id
				if filter.TeamId != 0 {
					query = query.
						Where("it.team_id = ?", filter.TeamId)
				}

				return next(query)
			}
		},
		func(db *gorm.DB, next db_connect.NextFunc) db_connect.NextFunc {
			return func(query *gorm.DB) (*gorm.DB, error) { // user id
				if filter.UserId != 0 {
					query = query.
						Where("it.create_by_id = ?", filter.UserId)
				}

				return next(query)
			}
		},

		func(db *gorm.DB, next db_connect.NextFunc) db_connect.NextFunc {
			return func(query *gorm.DB) (*gorm.DB, error) { // search query

				switch filter.SearchType {
				case warehouse_iface.OutboundSearchType_OUTBOUND_SEARCH_TYPE_ORDER_RECEIPT,
					warehouse_iface.OutboundSearchType_OUTBOUND_SEARCH_TYPE_UNSPECIFIED:

					if filter.Q == "" {
						return next(query)
					}

					fq := "%" + strings.ToLower(filter.Q) + "%"
					query = query.
						Where("(lower(it.receipt) like ?) or (lower(it.extern_ord_id) like ?)", fq, fq)

				case warehouse_iface.OutboundSearchType_OUTBOUND_SEARCH_TYPE_SKU_REFID:
					if len(filter.SkuIds) == 0 {
						return next(query)
					}

					txitemQuery := db.
						Table("inv_tx_items iti").
						Where("iti.sku_id in ?", filter.SkuIds).
						Where("iti.inv_transaction_id = it.id")

					query = query.
						Where("EXISTS (?)",
							txitemQuery.Select("1"),
						)
				}

				return next(query)
			}
		},

		func(db *gorm.DB, next db_connect.NextFunc) db_connect.NextFunc {
			return func(query *gorm.DB) (*gorm.DB, error) { // filter status

				if len(filter.Status) != 0 {
					query = query.
						Where("it.status IN ?", filter.Status)
				}

				return next(query)
			}
		},

		func(db *gorm.DB, next db_connect.NextFunc) db_connect.NextFunc {
			return func(query *gorm.DB) (*gorm.DB, error) { // filter shipping

				if len(filter.ShippingIds) != 0 {
					query = query.
						Where("it.shipping_id in ?", filter.ShippingIds)
				}

				return next(query)
			}
		},

		func(db *gorm.DB, next db_connect.NextFunc) db_connect.NextFunc {
			return func(query *gorm.DB) (*gorm.DB, error) { // filter shopid
				if filter.ShopId != 0 {
					query = query.
						Where("it.shop_id = ?", filter.ShopId)
				}

				return next(query)
			}
		},

		func(db *gorm.DB, next db_connect.NextFunc) db_connect.NextFunc {
			return func(query *gorm.DB) (*gorm.DB, error) { // filter shipment status

				if filter.Shipment != warehouse_iface.ShipmentStatus_SHIPMENT_STATUS_UNSPECIFIED {
					switch filter.Shipment {
					case warehouse_iface.ShipmentStatus_SHIPMENT_STATUS_SEND:
						query = query.
							Where("it.is_shipped = ?", true)

					case warehouse_iface.ShipmentStatus_SHIPMENT_STATUS_UNSEND:
						query = query.
							Where("it.is_shipped != ?", true)
					}
				}

				return next(query)
			}
		},

		func(db *gorm.DB, next db_connect.NextFunc) db_connect.NextFunc {
			return func(query *gorm.DB) (*gorm.DB, error) { // filter marketplace, shop, dan query
				var isOrderPreload bool

				isOrderPreload = len(filter.Marketplaces) != 0 ||
					filter.ShopId != 0 ||
					(filter.Q != "" &&
						filter.SearchType == warehouse_iface.OutboundSearchType_OUTBOUND_SEARCH_TYPE_SHOPNAME)

				if isOrderPreload {
					orderQuery, err := orderQuery(db)
					if err != nil {
						return nil, err
					}

					query = query.
						Where("EXISTS (?)",
							orderQuery.Select("1"),
						)
				}

				return next(query)
			}
		},

		func(db *gorm.DB, next db_connect.NextFunc) db_connect.NextFunc {
			return func(query *gorm.DB) (*gorm.DB, error) { // filter time
				timeFilter := filter.TimeRange
				if timeFilter != nil {
					if timeFilter.StartDate.IsValid() {
						query = query.
							Where("it.created > ?", timeFilter.StartDate.AsTime())
					}
					if timeFilter.EndDate.IsValid() {
						query = query.
							Where("it.created <= ?", timeFilter.EndDate.AsTime())
					}
				}
				return next(query)
			}
		},

		func(db *gorm.DB, next db_connect.NextFunc) db_connect.NextFunc {
			return func(query *gorm.DB) (*gorm.DB, error) { // pagination
				var queryPaginated *gorm.DB
				queryPaginated, result.PageInfo, err = db_connect.SetPaginationQuery(db, func() (*gorm.DB, error) {
					return query.Session(&gorm.Session{}), nil

				}, payload.Page)

				if err != nil {
					return query, err
				}

				return next(
					queryPaginated,
				)
			}
		},

		func(db *gorm.DB, next db_connect.NextFunc) db_connect.NextFunc {
			return func(query *gorm.DB) (*gorm.DB, error) { // sorting
				var key string
				switch paySort.Type {
				case common.SortType_SORT_TYPE_ASC:
					key = "asc nulls last"
				case common.SortType_SORT_TYPE_DESC:
					key = "desc nulls last"
				default:
					key = "desc nulls last"
				}

				switch paySort.Field {
				case warehouse_iface.OutboundSortField_OUTBOUND_SORT_FIELD_CREATED:
					query = query.Order("it.created " + key)
				case warehouse_iface.OutboundSortField_OUTBOUND_SORT_FIELD_MP_CREATED:
					query = query.
						Joins("JOIN orders o ON o.invertory_tx_id = it.id").
						Order("o.order_time " + key)
				case warehouse_iface.OutboundSortField_OUTBOUND_SORT_FIELD_DEADLINE:
					query = query.
						Joins("JOIN orders o ON o.invertory_tx_id = it.id").
						Where("o.order_deadline is not null").
						Order("o.order_deadline " + key)

				default:
					query = query.Order("it.id desc")
				}
				return next(query)
			}
		},
		NewGetResult(&result),
	)

	return connect.NewResponse(&result), err
}

func NewGetResult(result *warehouse_iface.OutboundListResponse) db_connect.NextHandler {
	return func(db *gorm.DB, next db_connect.NextFunc) db_connect.NextFunc {
		return func(query *gorm.DB) (*gorm.DB, error) {
			var err error

			var list InvTransactionList

			err = query.
				Preload("Items").
				Find(&list).
				Error

			if err != nil {
				return nil, err
			}

			result.Data = list.toProtos()

			// preload order
			tx_ids := make([]uint64, len(result.Data))
			itemMap := map[uint64]*warehouse_iface.Outbound{}
			for i, tx := range result.Data {
				tx_ids[i] = tx.Id
				itemMap[tx.Id] = tx
			}

			ords := []*db_models.Order{}

			err = db.
				Model(&db_models.Order{}).
				Where("invertory_tx_id in ?", tx_ids).
				Find(&ords).
				Error

			if err != nil {
				return nil, err
			}

			ordIds := []uint64{}
			ordermap := map[uint64]*warehouse_iface.Outbound{}
			for _, ord := range ords {
				ordIds = append(ordIds, uint64(ord.ID))
				ordermap[uint64(ord.ID)] = itemMap[uint64(*ord.InvertoryTxID)]
				itemMap[uint64(*ord.InvertoryTxID)].Extra = &warehouse_iface.Outbound_Order{
					Order: &warehouse_iface.Order{
						Id:     uint64(ord.ID),
						ShopId: uint64(ord.OrderMpID),
						// CustomerId: ord.,
						OrderTime:    timestamppb.New(ord.OrderTime),
						DeadlineTime: timestamppb.New(ord.OrderDeadline),
					},
				}
			}

			custs := []*db_models.CustomerAddress{}
			err = db.
				Model(&db_models.CustomerAddress{}).
				Where("order_id in ?", ordIds).
				Select([]string{"id", "order_id"}).
				Find(&custs).
				Error

			if err != nil {
				return nil, err
			}

			for _, cust := range custs {
				switch data := ordermap[uint64(cust.OrderID)].Extra.(type) {
				case *warehouse_iface.Outbound_Order:
					data.Order.CustomerId = uint64(cust.ID)
				}
			}

			return nil, nil
		}
	}
}

type InvTransactionList []*db_models.InvTransaction

func (list InvTransactionList) toProtos() []*warehouse_iface.Outbound {
	result := make([]*warehouse_iface.Outbound, len(list))
	for i, item := range list {
		items := []*warehouse_iface.OutboundItem{}
		for _, ditem := range item.Items {
			items = append(items, &warehouse_iface.OutboundItem{
				Id:    uint64(ditem.ID),
				SkuId: string(ditem.SkuID),
				Count: int64(ditem.Count),
				Owned: ditem.Owned,
				Total: ditem.Total,
			})
		}

		var shippingID uint64
		if item.ShippingID != nil {
			shippingID = uint64(*item.ShippingID)
		}

		result[i] = &warehouse_iface.Outbound{
			Id:          uint64(item.ID),
			TeamId:      uint64(item.TeamID),
			WarehouseId: uint64(item.WarehouseID),
			CreateById:  uint64(item.CreateByID),
			Status:      string(item.Status),
			Receipt:     item.Receipt,
			ReceiptFile: item.ReceiptFile,
			ExternOrdId: item.ExternOrdID,
			IsShipped:   item.IsShipped,
			ShippingId:  shippingID,
			IsDeleted:   item.Deleted,
			Items:       items,
			Created:     timestamppb.New(item.Created),
		}
	}

	return result
}
