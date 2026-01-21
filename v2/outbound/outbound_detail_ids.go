package outbound

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

// OutboundDetailIDs implements warehouse_ifaceconnect.OutboundServiceHandler.
func (o *outboundImpl) OutboundDetailIDs(
	ctx context.Context,
	req *connect.Request[warehouse_iface.OutboundDetailIDsRequest],
) (*connect.Response[warehouse_iface.OutboundDetailIDsResponse], error) {
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

	db := o.db.WithContext(ctx)
	pay := req.Msg

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

	result := warehouse_iface.OutboundDetailIDsResponse{
		Data: map[uint64]*warehouse_iface.OutboundDetailResponse{},
	}

	for _, txId := range pay.TxIds {
		result.Data[txId], err = o.outboundDetailItem(db, txId, pay.LoadAll)
		if err != nil {
			return nil, err
		}
	}

	return connect.NewResponse(&result), nil
}
