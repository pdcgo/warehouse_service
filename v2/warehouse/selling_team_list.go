package warehouse

import (
	"context"
	"strings"

	"connectrpc.com/connect"
	"github.com/pdcgo/schema/services/common/v1"
	"github.com/pdcgo/schema/services/warehouse_iface/v1"
	"github.com/pdcgo/shared/db_connect"
	"github.com/pdcgo/shared/pkg/common_helper"
	"gorm.io/gorm"
)

// SellingTeamList implements [warehouse_ifaceconnect.WarehouseServiceHandler].
func (w *warehouseServiceImpl) SellingTeamList(
	ctx context.Context,
	req *connect.Request[warehouse_iface.SellingTeamListRequest]) (*connect.Response[warehouse_iface.SellingTeamListResponse], error) {
	var err error
	pay := req.Msg

	result := &warehouse_iface.SellingTeamListResponse{
		List:     []*common.Team{},
		PageInfo: &common.PageInfo{},
	}

	db := w.db.WithContext(ctx)

	caller := common_helper.NewChainParam(
		func(next common_helper.NextFuncParam[*gorm.DB]) common_helper.NextFuncParam[*gorm.DB] {
			return func(query *gorm.DB) (*gorm.DB, error) { // build query
				teamIdQuery := query.
					Table("public.warehouse_products wp").
					Joins("left join products p on p.id = wp.product_id").
					Where("wp.warehouse_id = ?", pay.WarehouseId).
					Where("wp.stock > ?", 0).
					Select("distinct p.team_id")

				teamQuery := query.
					Table("(?) as tw", teamIdQuery).
					Joins("left join teams t on t.id = tw.team_id")

				// filtering search q
				q := strings.TrimSpace(pay.Q)
				q = strings.ToLower(q)
				if q != "" {
					teamQuery = teamQuery.Where("t.name ilike ?", "%"+q+"%")
				}

				return next(teamQuery)
			}
		},
		func(next common_helper.NextFuncParam[*gorm.DB]) common_helper.NextFuncParam[*gorm.DB] {
			return func(query *gorm.DB) (*gorm.DB, error) { // pagination
				var err error
				var paginated *gorm.DB

				paginated, result.PageInfo, err = db_connect.SetPaginationQuery(db, func() (*gorm.DB, error) {
					return query.Session(&gorm.Session{}), nil
				}, pay.Page)

				if err != nil {
					return nil, err

				}

				return next(paginated)

			}
		},
		func(next common_helper.NextFuncParam[*gorm.DB]) common_helper.NextFuncParam[*gorm.DB] {
			return func(query *gorm.DB) (*gorm.DB, error) { // getting data

				err = query.
					Select([]string{
						"t.id",
						"t.name",
						"t.team_code",
						"t.type",
					}).
					Find(&result.List).
					Error
				if err != nil {
					return nil, err
				}
				return next(query)
			}
		},
	)
	_, err = caller(db)

	return connect.NewResponse(result), err

}
