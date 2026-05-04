package warehouse_service

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"buf.build/go/protovalidate"
	"github.com/pdcgo/event_source"
	"github.com/pdcgo/schema/services/warehouse_iface/v1"
	"github.com/pdcgo/shared/db_models"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type WarehousePushHandler event_source.PushHandler

func NewWarehousePushHandler(db *gorm.DB, eventSender event_source.EventSender) WarehousePushHandler {

	return func(ctx context.Context, msg *event_source.PushRequest) error {
		var err error

		var event warehouse_iface.StockEvent
		err = protojson.Unmarshal(msg.Message.Data, &event)
		if err != nil {
			return err
		}

		// validating message
		err = protovalidate.GlobalValidator.Validate(&event)
		if err != nil {
			return err
		}

		db = db.WithContext(ctx)

		switch eventData := event.Data.(type) {
		case *warehouse_iface.StockEvent_RestockAccepted:
			err = SendLog(ctx, db, eventSender, eventData.RestockAccepted.TransactionId, warehouse_iface.StockChangeType_STOCK_CHANGE_TYPE_RESTOCK_ACCEPTED)
			if err != nil {
				return err
			}
		case *warehouse_iface.StockEvent_ReturnAccepted:
			err = SendLog(ctx, db, eventSender, eventData.ReturnAccepted.TransactionId, warehouse_iface.StockChangeType_STOCK_CHANGE_TYPE_RETURN_ACCEPTED)
			if err != nil {
				return err
			}

		case *warehouse_iface.StockEvent_OrderAccepted:
			err = SendLog(ctx, db, eventSender, eventData.OrderAccepted.TransactionId, warehouse_iface.StockChangeType_STOCK_CHANGE_TYPE_ORDER_ACCEPTED)
			if err != nil {
				return err
			}
		case *warehouse_iface.StockEvent_OrderCanceled:
			err = SendLog(ctx, db, eventSender, eventData.OrderCanceled.TransactionId, warehouse_iface.StockChangeType_STOCK_CHANGE_TYPE_ORDER_CANCELED)
			if err != nil {
				return err
			}
		case *warehouse_iface.StockEvent_StockChange:
			stockChange := eventData.StockChange

			for _, log := range stockChange.Changes {
				err = db.
					// Debug().
					Transaction(func(tx *gorm.DB) error {
						// locking sku

						var sku db_models.Sku
						err = tx.
							Clauses(clause.Locking{Strength: "UPDATE"}).
							Where("id = ?", log.SkuId).
							First(&sku).
							Error
						if err != nil {
							return err
						}

						initCount := gorm.Expr(
							`
						with d as (
							select 
								sum(ih.count * -1) as count
							from invertory_histories ih 
							where 
								ih.tx_id is null
								and ih.sku_id = ?
						)
						
						select coalesce(count, 0) from d
						`,
							sku.ID,
						)

						initAmount := gorm.Expr(
							`
						with d as (
							select 
								sum(
									-1 * ih.count * (
										ih.price + coalesce(ih.ext_price, 0)
									)
								) as amount
							from invertory_histories ih 
							where 
								ih.tx_id is null
								and ih.sku_id = ?
						)

						select coalesce(amount, 0) from d
						`,
							sku.ID,
						)

						params := map[string]interface{}{
							"t":             stockChange.CreatedTime.AsTime(),
							"sku_id":        log.SkuId,
							"warehouse_id":  log.WarehouseId,
							"init_count":    initCount,
							"init_amount":   initAmount,
							"change_count":  log.ChangeCount,
							"change_amount": log.ChangeAmount,
						}

						err = tx.
							Exec(`
							insert into daily_sku_histories (
								t, 
								sku_id, 
								warehouse_id, 
								start_stock_count, 
								end_stock_count, 
								start_stock_amount, 
								end_stock_amount,
								diff_stock_count,
								diff_stock_amount
							)
							values (
								date_trunc('day', @t ::timestamptz), 
								@sku_id, 
								@warehouse_id, 
								(@init_count) - @change_count, 
								(@init_count), 
								(@init_amount) - @change_amount, 
								(@init_amount),
								@change_count,
								@change_amount
							)
							on conflict (t, sku_id, warehouse_id) do update
							set 
								end_stock_count = daily_sku_histories.end_stock_count + @change_count,
								end_stock_amount = daily_sku_histories.end_stock_amount + @change_amount,
								diff_stock_count = daily_sku_histories.diff_stock_count + @change_count,
								diff_stock_amount = daily_sku_histories.diff_stock_amount + @change_amount
						`, params).
							Error

						if err != nil {
							return err
						}

						return nil
					})

				if err != nil {
					return err
				}
			}
		}

		return nil
	}
}

type WarehousePushHttpHandler http.HandlerFunc

func NewWarehousePushHttpHandler(handler WarehousePushHandler) WarehousePushHttpHandler {
	return WarehousePushHttpHandler(event_source.NewMuxPushhandler(event_source.PushHandler(handler)))
}

func SendLog(ctx context.Context, db *gorm.DB, eventSender event_source.EventSender, txId uint64, changeType warehouse_iface.StockChangeType) error {
	var err error
	var timetx time.Time
	var atField, changeCountField, changeAmountField, actorField string
	var timeArrived bool
	var n int

	switch changeType {
	case warehouse_iface.StockChangeType_STOCK_CHANGE_TYPE_ORDER_ACCEPTED,
		warehouse_iface.StockChangeType_STOCK_CHANGE_TYPE_STOCK_PROBLEM:
		timeArrived = false
		n = -1

	case warehouse_iface.StockChangeType_STOCK_CHANGE_TYPE_RESTOCK_ACCEPTED,
		warehouse_iface.StockChangeType_STOCK_CHANGE_TYPE_ORDER_CANCELED:
		timeArrived = true
		n = 1

	}

	changeAmountField = `(
				(iti.count - coalesce(iip.count, 0))
				* (iti.price + coalesce(rc.per_piece_fee, 0))
				* %d
			) as change_amount`

	changeCountField = `
				(iti.count - coalesce(iip.count, 0))
				* %d as change_count
			`

	if timeArrived {
		err = db.Raw(`
				select arrived from inv_transactions it where it.id = ?
			`, txId).
			Find(&timetx).
			Error
		atField = "it.arrived as at"

		actorField = "it.verify_by_id as actor_id"
	} else {
		err = db.Raw(`
				select created from inv_transactions it where it.id = ?
			`, txId).
			Find(&timetx).
			Error
		atField = "it.created as at"
		actorField = "it.create_by_id as actor_id"
	}

	changeCountField = fmt.Sprintf(changeCountField, n)
	changeAmountField = fmt.Sprintf(changeAmountField, n)

	if err != nil {
		return err
	}

	logs := []*warehouse_iface.StockChangeLog{}

	logquery := db.
		Select([]string{

			"iti.sku_id",
			"it.warehouse_id",
			actorField,
			atField,
			"it.id as transaction_id",
			changeCountField,
			changeAmountField,
		}).
		Table("inv_tx_items iti").
		Joins("join inv_transactions it on it.id = iti.inv_transaction_id").
		Joins("left join inv_item_problems iip on iip.tx_item_id = iti.id").
		Joins("left join restock_costs rc on rc.inv_transaction_id = iti.inv_transaction_id").
		Where("iti.inv_transaction_id = ?", txId)

	err = db.
		Table("(?) as d", logquery).
		Where("d.change_count != 0").
		Find(&logs).
		Error

	if err != nil {
		return err
	}

	if len(logs) == 0 {
		return nil
	}

	for _, log := range logs {
		log.TransactionAt = timestamppb.New(timetx)
		log.Type = changeType

		err = db.
			Create(log).
			Error
		if err != nil {
			return err
		}
	}

	_, err = eventSender(ctx, &warehouse_iface.StockEvent{
		Data: &warehouse_iface.StockEvent_StockChange{
			StockChange: &warehouse_iface.StockChange{
				Changes:     logs,
				CreatedTime: timestamppb.New(timetx),
			},
		},
	})
	if err != nil {
		return err
	}
	return nil
}
