package inventory

import (
	"context"
	"strings"

	"connectrpc.com/connect"
	"github.com/pdcgo/schema/services/access_iface/v1"
	"github.com/pdcgo/schema/services/warehouse_iface/v1"
	"github.com/pdcgo/shared/authorization"
	"github.com/pdcgo/shared/custom_connect"
	"github.com/pdcgo/shared/db_connect"
	"github.com/pdcgo/shared/db_models"
	"github.com/pdcgo/shared/interfaces/authorization_iface"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/gorm"
)

// ProductList implements warehouse_ifaceconnect.InventoryServiceHandler.
func (i *inventoryServiceImpl) ProductList(
	ctx context.Context,
	req *connect.Request[warehouse_iface.ProductListRequest],
) (*connect.Response[warehouse_iface.ProductListResponse], error) {
	var err error
	pay := req.Msg
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
		domainID = authorization.RootDomain
	default:
		domainID = uint(source.TeamId)
	}

	err = identity.
		HasPermission(authorization_iface.CheckPermissionGroup{
			&db_models.Product{}: &authorization_iface.CheckPermission{
				DomainID: domainID,
				Actions:  []authorization_iface.Action{authorization_iface.Read},
			},
		}).
		Err()

	if err != nil {
		return nil, err
	}

	db := i.db.WithContext(ctx).Debug()
	result := warehouse_iface.ProductListResponse{}

	filterPay := pay.Filter

	_, err = db_connect.NewQueryChain(db,
		func(db *gorm.DB, next db_connect.NextFunc) db_connect.NextFunc { // base query
			return func(query *gorm.DB) (*gorm.DB, error) {
				return next(
					query.
						Table("skus s"),
				)
			}
		},
		func(db *gorm.DB, next db_connect.NextFunc) db_connect.NextFunc { // filtering scope warehouse
			return func(query *gorm.DB) (*gorm.DB, error) {
				return next(
					query.
						Where("s.warehouse_id = ?", filterPay.WarehouseId),
				)
			}
		},
		func(db *gorm.DB, next db_connect.NextFunc) db_connect.NextFunc { // filter rack id
			return func(query *gorm.DB) (*gorm.DB, error) {

				if filterPay.RackId == 0 {
					return next(query)
				}

				subquery := db.
					Table("placements pl").
					Where("pl.sku_id = s.id").
					Where("pl.rack_id = ?", filterPay.RackId).
					Select("1")

				return next(
					query.
						Where("exists (?)", subquery),
				)
			}
		},
		func(db *gorm.DB, next db_connect.NextFunc) db_connect.NextFunc { // filter team id
			return func(query *gorm.DB) (*gorm.DB, error) {
				if filterPay.TeamId == 0 {
					return next(query)
				}

				return next(
					query.
						Where("s.team_id = ?", filterPay.TeamId),
				)
			}
		},
		func(db *gorm.DB, next db_connect.NextFunc) db_connect.NextFunc { // filter user id, search
			return func(query *gorm.DB) (*gorm.DB, error) {

				if filterPay.UserId == 0 || filterPay.Q == "" {
					return next(query)
				}

				subquery := db.
					Table("products p").
					Where("p.id = s.product_id")

				if filterPay.UserId != 0 {
					subquery = subquery.Where("p.user_id = ?", filterPay.UserId)
				}

				if filterPay.Q != "" {
					switch filterPay.SearchType {
					case warehouse_iface.ProductSearchType_PRODUCT_SEARCH_TYPE_SKU:
						switch db_models.CheckRefType(filterPay.Q) {
						case db_models.ProductRef:
							var q db_models.RefID = db_models.RefID(filterPay.Q)
							data, _ := q.ExtractData()
							data.WarehouseID = 0
							newq, _ := db_models.NewRefID(data)
							strnewq := string(newq)

							subquery = subquery.Where("p.ref_id ilike ?", "%"+strnewq+"%")

						case db_models.VariantRef:
							var q db_models.RefID = db_models.RefID(filterPay.Q)
							data, _ := q.ExtractData()
							data.WarehouseID = 0
							newq, _ := db_models.NewRefID(data)
							strnewq := string(newq)

							subqueryvar := db.
								Table("variation_values vv").
								Where("vv.id = s.variant_id").
								Where("vv.ref_id ilike ?", "%"+strings.ToLower(strnewq)+"%")

							query = query.
								Where("exists (?)", subqueryvar)

						case db_models.UnknownRef:
							querys := strings.Split(filterPay.Q, "-")
							if len(querys) >= 4 {
								querys[3] = "X"
							}
							fixq := strings.Join(querys, "-")
							subquery = subquery.Where("p.ref_id ilike ?", "%"+fixq+"%")
						}

					default:
						q := strings.ToLower(filterPay.Q)
						subquery = subquery.Where("p.name ilike ?", "%"+q+"%")

					}
				}

				subquery = subquery.Select("1")

				return next(
					query.
						Where("exists (?)", subquery),
				)
			}
		},
		func(db *gorm.DB, next db_connect.NextFunc) db_connect.NextFunc { // filter stock empty
			return func(query *gorm.DB) (*gorm.DB, error) {
				if !filterPay.EmptyStock {
					return next(query)
				}

				return next(
					query.
						Where("s.stock_ready <= 0"),
				)
			}
		},
		func(db *gorm.DB, next db_connect.NextFunc) db_connect.NextFunc { // paginated
			return func(query *gorm.DB) (*gorm.DB, error) {
				var queryPaginated *gorm.DB
				queryPaginated, result.PageInfo, err = db_connect.SetPaginationQuery(db, func() (*gorm.DB, error) {
					return query.Session(&gorm.Session{}), nil

				}, pay.Page)

				if err != nil {
					return query, err
				}

				return next(
					queryPaginated,
				)
			}
		},
		func(db *gorm.DB, next db_connect.NextFunc) db_connect.NextFunc { // sorting data
			return func(query *gorm.DB) (*gorm.DB, error) {
				return next(
					query.Order("stock_ready desc"),
				)
			}
		},
		func(db *gorm.DB, next db_connect.NextFunc) db_connect.NextFunc { // getting data
			return func(query *gorm.DB) (*gorm.DB, error) {
				skus := []*db_models.Sku{}
				err = query.Find(&skus).Error

				if err != nil {
					return nil, err
				}

				result.Data = make([]*warehouse_iface.WarehouseProduct, len(skus))
				for i, sku := range skus {
					result.Data[i] = &warehouse_iface.WarehouseProduct{
						SkuId:        string(sku.ID),
						ProductId:    uint64(sku.ProductID),
						VariantId:    uint64(sku.VariantID),
						WarehouseId:  uint64(sku.WarehouseID),
						VariantRefId: "",
						Price:        sku.NextPrice,
						StockReady:   uint64(sku.StockReady),
						StockPending: uint64(sku.StockPending),
						LastInbound:  timestamppb.New(sku.LastInbound),
						LastOutbound: timestamppb.New(sku.LastOutbound),
					}
				}

				return next(
					query,
				)
			}
		},
		func(db *gorm.DB, next db_connect.NextFunc) db_connect.NextFunc { // preloading data
			return func(query *gorm.DB) (*gorm.DB, error) {
				preloadMap := map[uint64][]*warehouse_iface.WarehouseProduct{}
				preloadVariantMap := map[uint64][]*warehouse_iface.WarehouseProduct{}

				pids := []uint64{}
				varIds := []uint64{}

				for _, data := range result.Data {
					if preloadMap[data.ProductId] == nil {
						preloadMap[data.ProductId] = []*warehouse_iface.WarehouseProduct{}
					}

					if preloadVariantMap[data.VariantId] == nil {
						preloadVariantMap[data.VariantId] = []*warehouse_iface.WarehouseProduct{}
					}

					preloadMap[data.ProductId] = append(preloadMap[data.ProductId], data)
					preloadVariantMap[data.VariantId] = append(preloadVariantMap[data.VariantId], data)

					pids = append(pids, data.ProductId)
					varIds = append(varIds, data.VariantId)
				}

				preloadQuery := db.
					Table("products p").
					Where("p.id in ?", pids).
					Select([]string{
						"p.id",
						"p.name",
						"p.image",
						"p.ref_id",
					})

				prods := []*db_models.Product{}
				err = preloadQuery.Find(&prods).Error

				if err != nil {
					return nil, err
				}

				for _, prod := range prods {
					for _, data := range preloadMap[uint64(prod.ID)] {
						data.Name = prod.Name
						data.Image = prod.Image[0]
						data.ProductRefId = string(prod.RefID)
						data.Created = timestamppb.New(prod.Created)
					}
				}

				preloadVarQuery := db.
					Table("variation_values vv").
					Where("vv.id in ?", varIds).
					Select([]string{
						"vv.id",
						"vv.ref_id",
					})

				variations := []*db_models.VariationValue{}
				err = preloadVarQuery.Find(&variations).Error

				if err != nil {
					return nil, err
				}

				for _, varr := range variations {
					for _, data := range preloadVariantMap[uint64(varr.ID)] {
						data.VariantRefId = string(varr.RefID)
					}
				}

				return next(query)
			}
		},
	)

	if err != nil {
		return nil, err
	}

	return connect.NewResponse(&result), err
}
