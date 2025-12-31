package outbound

import (
	"context"
	"errors"
	"fmt"

	"connectrpc.com/connect"
	"github.com/pdcgo/schema/services/access_iface/v1"
	"github.com/pdcgo/schema/services/warehouse_iface/v1"
	"github.com/pdcgo/shared/authorization"
	"github.com/pdcgo/shared/custom_connect"
	"github.com/pdcgo/shared/db_models"
	"github.com/pdcgo/shared/interfaces/authorization_iface"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// OutboundDetail implements warehouse_ifaceconnect.OutboundServiceHandler.
func (o *outboundImpl) OutboundDetail(
	ctx context.Context,
	req *connect.Request[warehouse_iface.OutboundDetailRequest],
) (*connect.Response[warehouse_iface.OutboundDetailResponse], error) {
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
	result := warehouse_iface.OutboundDetailResponse{}

	query := db.Model(&db_models.InvTransaction{})

	switch pay.SearchType {
	case warehouse_iface.OutboundDetailSearchType_OUTBOUND_DETAIL_SEARCH_TYPE_RECEIPT_ORDREF:
		q := pay.Q
		query = query.
			Where("extern_ord_id = ? or receipt = ?", q, q)
	case warehouse_iface.OutboundDetailSearchType_OUTBOUND_DETAIL_SEARCH_TYPE_TXID:
		query = query.
			Where("id = ?", pay.TxId)
	default:
		return nil, errors.New("search type not supported")
	}

	switch source.RequestFrom {
	case access_iface.RequestFrom_REQUEST_FROM_WAREHOUSE:
		query = query.
			Where("warehouse_id = ?", source.TeamId)
	}

	outbound := db_models.InvTransaction{}
	err = query.
		Preload("Items").
		First(&outbound).
		Error
	if err != nil {
		return nil, err
	}

	outitems := make([]*warehouse_iface.OutboundItem, len(outbound.Items))
	for i, item := range outbound.Items {
		skudata, err := item.SkuID.Extract()
		if err != nil {
			return nil, err
		}

		outitems[i] = &warehouse_iface.OutboundItem{
			Id:    uint64(item.ID),
			SkuId: string(item.SkuID),
			Count: int64(item.Count),
			Owned: item.Owned,
			Total: item.Total,
			Price: item.Price,
			SkuDetail: &warehouse_iface.SkuDataDetail{
				ProductId:   uint64(skudata.ProductID),
				VariantId:   uint64(skudata.VariantID),
				WarehouseId: uint64(skudata.WarehouseID),
				TeamId:      uint64(skudata.TeamID),
			},
		}
	}

	result.Outbound = &warehouse_iface.Outbound{
		Id:          uint64(outbound.ID),
		TeamId:      uint64(outbound.TeamID),
		WarehouseId: uint64(outbound.WarehouseID),
		CreateById:  uint64(outbound.CreateByID),
		ExternOrdId: outbound.ExternOrdID,
		Status:      string(outbound.Status),
		Receipt:     outbound.Receipt,
		ReceiptFile: outbound.ReceiptFile,
		IsDeleted:   outbound.Deleted,
		IsShipped:   outbound.IsShipped,
		Created:     timestamppb.New(outbound.Created),
		Items:       outitems,
	}

	if outbound.ShippingID != nil {
		result.Outbound.ShippingId = uint64(*outbound.ShippingID)
	}

	// getting extra
	switch outbound.Type {
	case db_models.InvTxOrder:
		ord := db_models.Order{}
		err = db.
			Model(&db_models.Order{}).
			Where("invertory_tx_id = ?", outbound.ID).
			First(&ord).
			Error

		if err != nil {
			return nil, err
		}

		orddetail := warehouse_iface.OrderDetail{
			Id:          uint64(ord.ID),
			TeamId:      uint64(ord.TeamID),
			CreateById:  uint64(ord.CreatedByID),
			DoubleOrder: ord.DoubleOrder,
			OrderRefId:  ord.OrderRefID,
			OrderFrom:   string(ord.OrderFrom),
		}

		if ord.InvertoryTxID != nil {
			orddetail.InvertoryTxId = uint64(*ord.InvertoryTxID)
		}

		if ord.InvertoryReturnTxID != nil {
			orddetail.InvertoryReturnTxId = uint64(*ord.InvertoryReturnTxID)
		}

		result.Extra = &warehouse_iface.OutboundDetailResponse_OrderDetail{
			OrderDetail: &orddetail,
		}

	default:
		return nil, fmt.Errorf("outbound type %s not implemented", outbound.Type)
	}

	return connect.NewResponse(&result), nil
}
