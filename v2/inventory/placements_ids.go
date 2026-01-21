package inventory

import (
	"context"
	"errors"

	"connectrpc.com/connect"
	"github.com/pdcgo/schema/services/access_iface/v1"
	"github.com/pdcgo/schema/services/warehouse_iface/v1"
	"github.com/pdcgo/shared/authorization"
	"github.com/pdcgo/shared/custom_connect"
	"github.com/pdcgo/shared/db_models"
	"github.com/pdcgo/shared/interfaces/authorization_iface"
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

	for _, txID := range pay.TxIds {
		result.BulkPlacements[txID] = &warehouse_iface.BulkPlacement{
			Data: map[string]*warehouse_iface.PlacementDetail{},
		}
		result.BulkPlacements[txID].Data, err = i.getPlacements(db, txID)

		if err != nil {
			return nil, err
		}
	}

	return connect.NewResponse(&result), nil
}
