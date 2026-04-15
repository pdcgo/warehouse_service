package outbound

import (
	"context"
	"strings"

	"connectrpc.com/connect"
	"github.com/pdcgo/schema/services/warehouse_iface/v1"
	"github.com/pdcgo/shared/db_models"
	"github.com/pdcgo/shared/pkg/common_helper"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/gorm"
)

// OrderDetailSearch implements warehouse_ifaceconnect.OutboundServiceHandler.
func (o *outboundImpl) OrderDetailSearch(
	ctx context.Context,
	req *connect.Request[warehouse_iface.OrderDetailSearchRequest],
) (*connect.Response[warehouse_iface.OrderDetailSearchResponse], error) {

	var err error

	txmap := map[uint64]*warehouse_iface.TransactionDetail{}
	retTxMap := map[uint64]*warehouse_iface.TransactionDetail{}
	custMap := map[uint64]*warehouse_iface.CustomerDetail{}

	pay := req.Msg
	result := &warehouse_iface.OrderDetailSearchResponse{}

	db := o.db.WithContext(ctx).Debug()

	caller := common_helper.NewChainParam(
		func(next common_helper.NextFuncParam[*gorm.DB]) common_helper.NextFuncParam[*gorm.DB] {
			return func(query *gorm.DB) (*gorm.DB, error) { // creating base query

				orderQuery := query.
					Table("public.orders o")

				switch search := pay.Search.(type) {
				case *warehouse_iface.OrderDetailSearchRequest_Receipt: // receipt
					receipt := strings.ToLower(search.Receipt)
					receipt = strings.TrimSpace(receipt) + "%"
					txQuery := query.
						Table("public.inv_transactions it").
						Where("it.receipt ilike ?", receipt).
						Where("it.id = o.invertory_tx_id or it.id = o.invertory_return_tx_id").
						Select("1")

					orderQuery = orderQuery.
						Where("EXISTS (?)", txQuery)

				case *warehouse_iface.OrderDetailSearchRequest_OrderRefId: // order_ref_id
					orderRefId := strings.TrimSpace(search.OrderRefId) + "%"
					orderQuery = orderQuery.
						Where("o.order_ref_id ilike ?", orderRefId)
				case *warehouse_iface.OrderDetailSearchRequest_Customer: // customer
					name := strings.TrimSpace(search.Customer.Name)
					city := strings.TrimSpace(search.Customer.City)
					custQuery := query.
						Table("public.customer_addresses ca")

					if name != "" {
						custQuery = custQuery.Where("ca.name ilike ?", name+"%")
					}
					if city != "" {
						custQuery = custQuery.Where("ca.city ilike ?", city+"%")
					}
					custQuery = custQuery.Where("o.id = ca.order_id").
						Select("1")

					orderQuery = orderQuery.
						Where("EXISTS (?)", custQuery)

				}

				if pay.Status != "" {
					orderQuery = orderQuery.
						Where("o.status = ?", pay.Status)
				}

				return next(orderQuery)
			}
		},
		func(next common_helper.NextFuncParam[*gorm.DB]) common_helper.NextFuncParam[*gorm.DB] {
			return func(query *gorm.DB) (*gorm.DB, error) { // loading orders
				var err error

				orders := []*db_models.Order{}
				err = query.
					Limit(int(pay.Limit)).
					Find(&orders).
					Error

				if err != nil {
					return nil, err
				}

				result.Data = make([]*warehouse_iface.OrderDetailSearchItem, len(orders))

				for i, ord := range orders {
					var invertoryTxId uint64
					if ord.InvertoryTxID != nil {
						invertoryTxId = uint64(*ord.InvertoryTxID)
						txmap[invertoryTxId] = &warehouse_iface.TransactionDetail{}
					}

					var invertoryReturnTxId uint64
					if ord.InvertoryReturnTxID != nil {
						invertoryReturnTxId = uint64(*ord.InvertoryReturnTxID)
						retTxMap[invertoryReturnTxId] = &warehouse_iface.TransactionDetail{}
					}

					custMap[uint64(ord.ID)] = &warehouse_iface.CustomerDetail{}

					result.Data[i] = &warehouse_iface.OrderDetailSearchItem{

						Order: &warehouse_iface.OrderDetailSearch{
							Id:                  uint64(ord.ID),
							TeamId:              uint64(ord.TeamID),
							CreateById:          uint64(ord.CreatedByID),
							InvertoryTxId:       invertoryTxId,
							InvertoryReturnTxId: invertoryReturnTxId,
							Status:              string(ord.Status),
							DoubleOrder:         ord.DoubleOrder,
							OrderRefId:          ord.OrderRefID,
							Receipt:             ord.Receipt,
							ReturnReceipt:       ord.ReceiptReturn,
							OrderFrom:           ord.OrderFrom.ToProto(),
							OrderTime:           timestamppb.New(ord.OrderTime),
							Customer:            custMap[uint64(ord.ID)],
						},
						Outbound: txmap[invertoryTxId],
						Inbound:  retTxMap[invertoryReturnTxId],
					}
				}

				return next(query)
			}
		},

		PreloadTransaction(db, txmap),
		PreloadTransaction(db, retTxMap),
		PreloadCustomerDetail(db, custMap),
	)

	_, err = caller(db)

	return connect.NewResponse(result), err
}

func PreloadCustomerDetail(db *gorm.DB, custMap map[uint64]*warehouse_iface.CustomerDetail) common_helper.NextHandlerParam[*gorm.DB] {
	return func(next common_helper.NextFuncParam[*gorm.DB]) common_helper.NextFuncParam[*gorm.DB] {
		return func(query *gorm.DB) (*gorm.DB, error) {
			var err error

			if len(custMap) == 0 {
				return next(query)
			}

			orderIds := make([]uint64, 0, len(custMap))
			for orderId := range custMap {
				orderIds = append(orderIds, orderId)
			}

			customerAddresses := []*db_models.CustomerAddress{}
			err = db.
				Model(&db_models.CustomerAddress{}).
				Where("order_id in ?", orderIds).
				Find(&customerAddresses).
				Error

			if err != nil {
				return nil, err
			}

			for _, addr := range customerAddresses {
				cust := &warehouse_iface.CustomerDetail{
					Id:         uint64(addr.ID),
					Name:       addr.Name,
					Phone:      addr.Phone,
					City:       addr.City,
					District:   addr.District,
					PostalCode: addr.PostalCode,
					Address:    addr.Address,
				}

				proto.Merge(custMap[uint64(addr.OrderID)], cust)
			}

			return next(query)

		}
	}
}

func PreloadTransaction(db *gorm.DB, txmap map[uint64]*warehouse_iface.TransactionDetail) common_helper.NextHandlerParam[*gorm.DB] {
	return func(next common_helper.NextFuncParam[*gorm.DB]) common_helper.NextFuncParam[*gorm.DB] {
		return func(query *gorm.DB) (*gorm.DB, error) {
			var err error

			if len(txmap) == 0 {
				return next(query)
			}

			txIds := make([]uint64, 0, len(txmap))
			for txId := range txmap {
				txIds = append(txIds, txId)
			}

			txs := []*db_models.InvTransaction{}
			err = db.
				Model(&db_models.InvTransaction{}).
				Preload("Items").
				Where("id in ?", txIds).
				Find(&txs).
				Error

			if err != nil {
				return nil, err
			}

			for _, tx := range txs {
				txItems := make([]*warehouse_iface.TransactionItem, len(tx.Items))

				for i, item := range tx.Items {
					skuData, err := item.SkuID.Extract()
					if err != nil {
						return nil, err
					}

					txItems[i] = &warehouse_iface.TransactionItem{
						Id:    uint64(item.ID),
						SkuId: string(item.SkuID),
						Owned: item.Owned,
						Count: int64(item.Count),
						Price: item.Price,
						Total: item.Total,
						SkuDetail: &warehouse_iface.SkuDataDetail{
							ProductId:   uint64(skuData.ProductID),
							VariantId:   uint64(skuData.VariantID),
							WarehouseId: uint64(skuData.WarehouseID),
							TeamId:      uint64(skuData.TeamID),
						},
					}
				}

				data := &warehouse_iface.TransactionDetail{
					Id:          uint64(tx.ID),
					TeamId:      uint64(tx.TeamID),
					WarehouseId: uint64(tx.WarehouseID),
					CreateById:  uint64(tx.CreateByID),
					ExternOrdId: tx.ExternOrdID,
					Receipt:     tx.Receipt,
					ReceiptFile: tx.ReceiptFile,
					Items:       txItems,
					IsShipped:   tx.IsShipped,
					Total:       tx.Total,
					Created:     timestamppb.New(tx.Created),
					Status:      string(tx.Status),
					Type:        string(tx.Type),
				}

				if tx.VerifyByID != nil {
					data.VerifyById = uint64(*tx.VerifyByID)
				}

				if tx.ShippingID != nil {
					data.ShippingId = uint64(*tx.ShippingID)
				}

				if tx.SendAt != nil {
					data.SendAt = timestamppb.New(*tx.SendAt)
				}

				if tx.Arrived != nil {
					data.Arrived = timestamppb.New(*tx.Arrived)
				}

				proto.Merge(txmap[uint64(tx.ID)], data)
			}

			return next(query)

		}
	}
}
