package warehouse

import (
	"context"
	"strings"

	"connectrpc.com/connect"
	"github.com/pdcgo/schema/services/warehouse_iface/v1"
)

// TeamRackList implements [warehouse_ifaceconnect.WarehouseServiceHandler].
func (w *warehouseServiceImpl) TeamRackList(
	ctx context.Context,
	req *connect.Request[warehouse_iface.TeamRackListRequest],
) (*connect.Response[warehouse_iface.TeamRackListResponse], error) {
	var err error

	result := &warehouse_iface.TeamRackListResponse{
		List: []*warehouse_iface.Rack{},
	}

	db := w.db.WithContext(ctx)
	search := req.Msg.Q
	search = strings.TrimSpace(search)

	query := db.
		Debug().
		Table("public.placements p").
		Joins("left join public.skus s on s.id = p.sku_id").
		Where("s.team_id in ?", req.Msg.TeamIds).
		Where("s.warehouse_id = ?", req.Msg.WarehouseId).
		Group("p.rack_id").
		Select([]string{
			"p.rack_id",
			"sum(p.count) as count",
		})

	rquery := db.
		Table("(?) p", query).
		Joins("join public.racks r on r.id = p.rack_id")

	if search != "" {
		rquery = rquery.
			Where("r.name ilike ?", search+"%")
	}

	rquery = rquery.
		Select([]string{
			"r.id as id",
			"r.name as name",
			"p.count as count",
		})

	err = rquery.Find(&result.List).Error
	if err != nil {
		return nil, err
	}

	return connect.NewResponse(result), nil
}
