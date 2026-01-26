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
	"github.com/pdcgo/shared/db_connect"
	"github.com/pdcgo/shared/db_models"
	"github.com/pdcgo/shared/interfaces/authorization_iface"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/gorm"
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

	// mapping data
	orderTxIds := []uint64{}
	txMap := map[uint64]*db_models.InvTransaction{}
	orderMap := map[uint64]*db_models.Order{}

	_, err = db_connect.NewQueryChain(db,
		func(db *gorm.DB, next db_connect.NextFunc) db_connect.NextFunc { // getting transaction
			return func(query *gorm.DB) (*gorm.DB, error) {
				trans := []*db_models.InvTransaction{}
				qinv := db.
					Model(&db_models.InvTransaction{})

				if pay.LoadAll {
					qinv = qinv.
						Preload("Items")
				}

				err = qinv.
					Where("id in ?", pay.TxIds).
					Find(&trans).
					Error

				if err != nil {
					return nil, err
				}

				for _, tran := range trans {
					txMap[uint64(tran.ID)] = tran
					switch tran.Type {
					case db_models.InvTxOrder:
						orderTxIds = append(orderTxIds, uint64(tran.ID))
					default:
						return nil, fmt.Errorf("outbound type %s not supported", tran.Type)
					}
				}

				return next(query)
			}

		},
		func(db *gorm.DB, next db_connect.NextFunc) db_connect.NextFunc { // getting order
			return func(query *gorm.DB) (*gorm.DB, error) {
				if !pay.LoadAll {
					return next(query)
				}

				if len(orderTxIds) == 0 {
					return next(query)
				}

				orders := []*db_models.Order{}

				err = db.
					Model(&db_models.Order{}).
					Where("invertory_tx_id in ?", orderTxIds).
					Find(&orders).
					Error

				if err != nil {
					return nil, err
				}

				for _, ord := range orders {
					orderMap[uint64(*ord.InvertoryTxID)] = ord
				}

				return next(query)
			}
		},
	)

	if err != nil {
		return nil, err
	}

	// adding to result
	result := warehouse_iface.OutboundDetailIDsResponse{
		Data: map[uint64]*warehouse_iface.OutboundDetailResponse{},
	}

	for _, tran := range txMap {

		outbound := txMap[uint64(tran.ID)]
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

		var shippingID uint64
		if outbound.ShippingID != nil {
			shippingID = uint64(*outbound.ShippingID)
		}

		result.Data[uint64(tran.ID)] = &warehouse_iface.OutboundDetailResponse{
			Outbound: &warehouse_iface.Outbound{
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
				ShippingId:  shippingID,
			},
		}

		if !pay.LoadAll {
			continue
		}

		// getting extra
		res := result.Data[uint64(tran.ID)]
		switch outbound.Type {
		case db_models.InvTxOrder:
			ord := orderMap[uint64(outbound.ID)]

			orddetail := warehouse_iface.OrderDetail{
				Id:          uint64(ord.ID),
				TeamId:      uint64(ord.TeamID),
				CreateById:  uint64(ord.CreatedByID),
				DoubleOrder: ord.DoubleOrder,
				OrderRefId:  ord.OrderRefID,
				OrderFrom:   string(ord.OrderFrom),
				OrderTime:   timestamppb.New(ord.OrderTime),
			}

			if ord.InvertoryTxID != nil {
				orddetail.InvertoryTxId = uint64(*ord.InvertoryTxID)
			}

			if ord.InvertoryReturnTxID != nil {
				orddetail.InvertoryReturnTxId = uint64(*ord.InvertoryReturnTxID)
			}

			res.Extra = &warehouse_iface.OutboundDetailResponse_OrderDetail{
				OrderDetail: &orddetail,
			}

		default:
			return nil, fmt.Errorf("outbound type %s not implemented", outbound.Type)
		}

	}

	return connect.NewResponse(&result), nil
}
