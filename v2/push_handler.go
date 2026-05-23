package warehouse_service

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"buf.build/go/protovalidate"
	"github.com/pdcgo/event_source"
	"github.com/pdcgo/schema/services/warehouse_iface/v1"
	"github.com/pdcgo/shared/db_models"
	"github.com/pdcgo/shared/pkg/common_helper"
	"github.com/pdcgo/warehouse_service/warehouse_models"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type EventUnuportedErr struct {
	*warehouse_iface.StockEvent
}

// Error implements [error].
func (e *EventUnuportedErr) Error() string {
	return fmt.Sprintf("unsupported event: %T", e.StockEvent)
}

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

		messageID := msg.Message.MessageID

		return db.Transaction(func(tx *gorm.DB) error {
			handler := common_helper.NewChainParam(
				func(next common_helper.NextFuncParam[*warehouse_iface.StockEvent]) common_helper.NextFuncParam[*warehouse_iface.StockEvent] {
					return func(event *warehouse_iface.StockEvent) (*warehouse_iface.StockEvent, error) { // deduplicate event

						dedup := warehouse_models.StockEventLog{
							ID:  messageID,
							Raw: msg.Message.Data,
						}

						res := tx.
							Clauses(clause.OnConflict{DoNothing: true}).
							Create(&dedup)

						if res.RowsAffected == 0 {
							return event, nil
						}

						return next(event)
					}
				},

				func(next common_helper.NextFuncParam[*warehouse_iface.StockEvent]) common_helper.NextFuncParam[*warehouse_iface.StockEvent] {
					return func(event *warehouse_iface.StockEvent) (*warehouse_iface.StockEvent, error) { // create change log
						var changeEvent *warehouse_iface.StockEvent
						var err error

						switch eventData := event.Data.(type) {
						case *warehouse_iface.StockEvent_RestockAccepted:
							tx = tx.Debug()
							changeEvent, err = CreateStockChangeLog(tx, msg.Message.MessageID, eventData.RestockAccepted.TransactionId, warehouse_iface.StockChangeType_STOCK_CHANGE_TYPE_RESTOCK_ACCEPTED)

						case *warehouse_iface.StockEvent_ReturnAccepted:
							changeEvent, err = CreateStockChangeLog(tx, msg.Message.MessageID, eventData.ReturnAccepted.TransactionId, warehouse_iface.StockChangeType_STOCK_CHANGE_TYPE_RETURN_ACCEPTED)
						case *warehouse_iface.StockEvent_OrderAccepted:
							changeEvent, err = CreateStockChangeLog(tx, msg.Message.MessageID, eventData.OrderAccepted.TransactionId, warehouse_iface.StockChangeType_STOCK_CHANGE_TYPE_ORDER_ACCEPTED)
						case *warehouse_iface.StockEvent_OrderCanceled:
							changeEvent, err = CreateStockChangeLog(tx, msg.Message.MessageID, eventData.OrderCanceled.TransactionId, warehouse_iface.StockChangeType_STOCK_CHANGE_TYPE_ORDER_CANCELED)

						case *warehouse_iface.StockEvent_TransferWarehouseCreated:

							transfer, err := getTransfer(tx, eventData.TransferWarehouseCreated.TransferId)
							if err != nil {
								return changeEvent, err
							}

							changeEvent, err = CreateStockChangeLog(tx, msg.Message.MessageID, uint64(transfer.OutboundTxID), warehouse_iface.StockChangeType_STOCK_CHANGE_TYPE_TRANSFER_WAREHOUSE_OUT)

						case *warehouse_iface.StockEvent_TransferWarehouseAccepted:
							transfer, err := getTransfer(tx, eventData.TransferWarehouseAccepted.TransferId)
							if err != nil {
								return changeEvent, err
							}

							changeEvent, err = CreateStockChangeLog(tx, msg.Message.MessageID, uint64(transfer.InboundTxID), warehouse_iface.StockChangeType_STOCK_CHANGE_TYPE_TRANSFER_WAREHOUSE_IN)

						case *warehouse_iface.StockEvent_TransferWarehouseCanceled:
							transfer, err := getTransfer(tx, eventData.TransferWarehouseCanceled.TransferId)
							if err != nil {
								return changeEvent, err
							}

							changeEvent, err = CreateStockChangeLog(tx, msg.Message.MessageID, uint64(transfer.OutboundTxID), warehouse_iface.StockChangeType_STOCK_CHANGE_TYPE_TRANSFER_WAREHOUSE_OUT_CANCELED)
						case *warehouse_iface.StockEvent_StockFoundBack:
							changeEvent, err = CreateStockChangeLog(tx, msg.Message.MessageID, eventData.StockFoundBack.TransactionId, warehouse_iface.StockChangeType_STOCK_CHANGE_TYPE_STOCK_FOUND_BACK)
						case *warehouse_iface.StockEvent_StockProblem:
							changeEvent, err = CreateStockChangeLog(tx, msg.Message.MessageID, eventData.StockProblem.TransactionId, warehouse_iface.StockChangeType_STOCK_CHANGE_TYPE_STOCK_PROBLEM)

						case *warehouse_iface.StockEvent_StockChange:
							return next(event)

						default:
							slog.Warn("unsupported event", "event", eventData)
							return nil, &EventUnuportedErr{StockEvent: event}
						}

						if err != nil {
							return changeEvent, err
						}

						return next(changeEvent)
					}

				},
				func(next common_helper.NextFuncParam[*warehouse_iface.StockEvent]) common_helper.NextFuncParam[*warehouse_iface.StockEvent] {
					return func(event *warehouse_iface.StockEvent) (*warehouse_iface.StockEvent, error) { // insert to stock log
						if event == nil {
							slog.Warn("skipping event nil", "event", event)
							return nil, nil
						}

						if event.Data == nil {
							slog.Warn("skipping event data nil", "event", event)
							return nil, nil
						}

						switch eventData := event.Data.(type) {
						case *warehouse_iface.StockEvent_StockChange:

							stockChange := eventData.StockChange

							for _, log := range stockChange.Changes {

								err := tx.
									Clauses(clause.OnConflict{DoNothing: true}).
									Create(log).
									Error

								if err != nil {
									return event, err
								}
							}

							return next(event)
						default:
							slog.Warn("unsupported event", "event", eventData)
							return nil, &EventUnuportedErr{StockEvent: event}
						}
					}
				},
				func(next common_helper.NextFuncParam[*warehouse_iface.StockEvent]) common_helper.NextFuncParam[*warehouse_iface.StockEvent] {
					return func(event *warehouse_iface.StockEvent) (*warehouse_iface.StockEvent, error) { // updating to daily sku histories, insert daily stock and sum

						switch eventData := event.Data.(type) {
						case *warehouse_iface.StockEvent_StockChange:
							stockChange := eventData.StockChange

							for _, log := range stockChange.Changes {
								var sku db_models.Sku
								err = tx.
									Clauses(clause.Locking{Strength: "UPDATE"}).
									Where("id = ?", log.SkuId).
									First(&sku).
									Error
								if err != nil {

									return event, err
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
													date_trunc('day', @t ::timestamptz AT TIME ZONE 'Asia/Jakarta'), 
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
									return event, err
								}
							}
						}

						return next(event)
					}
				},
			)

			_, err = handler(&event)
			if err != nil {
				return err
			}

			return nil
		})

	}
}

type WarehousePushHttpHandler http.HandlerFunc

func NewWarehousePushHttpHandler(handler WarehousePushHandler) WarehousePushHttpHandler {
	return WarehousePushHttpHandler(event_source.NewMuxPushhandler(event_source.PushHandler(handler)))
}

func CreateStockChangeLog(
	tx *gorm.DB,
	externalMsgId string,
	txId uint64,
	changeType warehouse_iface.StockChangeType,
) (*warehouse_iface.StockEvent, error) {
	var err error
	var timetx time.Time
	var atField, changeCountField, changeAmountField, actorField string

	var n int

	changeAmountField = `
			(
				(iti.count - coalesce(iip.count, 0))
				* (iti.price + coalesce(rc.per_piece_fee, 0))
				* %d
			) as change_amount
	`

	changeCountField = `
				(iti.count - coalesce(iip.count, 0))
				* %d as change_count
			`

	switch changeType {
	case warehouse_iface.StockChangeType_STOCK_CHANGE_TYPE_ORDER_ACCEPTED,
		warehouse_iface.StockChangeType_STOCK_CHANGE_TYPE_TRANSFER_WAREHOUSE_OUT,
		warehouse_iface.StockChangeType_STOCK_CHANGE_TYPE_TRANSFER_WAREHOUSE_IN_CANCELED:

		n = -1
		atField = "it.created as at"
		actorField = "it.create_by_id as actor_id"
		err = tx.Raw(`
				select created from inv_transactions it where it.id = ?
			`, txId).
			Find(&timetx).
			Error

		if err != nil {
			return nil, err
		}

	case warehouse_iface.StockChangeType_STOCK_CHANGE_TYPE_STOCK_PROBLEM:
		n = -1
		atField = "it.created as at"
		actorField = "it.create_by_id as actor_id"
		err = tx.Raw(`
				select created from inv_transactions it where it.id = ?
			`, txId).
			Find(&timetx).
			Error

		if err != nil {
			return nil, err
		}

		changeAmountField = `
			(
				iti.count
				* (iti.price + coalesce(rc.per_piece_fee, 0))
				* %d
			) as change_amount
	`

		changeCountField = `
				iti.count
				* %d as change_count
			`

	case warehouse_iface.StockChangeType_STOCK_CHANGE_TYPE_RESTOCK_ACCEPTED,
		warehouse_iface.StockChangeType_STOCK_CHANGE_TYPE_RETURN_ACCEPTED,
		warehouse_iface.StockChangeType_STOCK_CHANGE_TYPE_STOCK_FOUND_BACK,
		warehouse_iface.StockChangeType_STOCK_CHANGE_TYPE_TRANSFER_WAREHOUSE_IN,
		warehouse_iface.StockChangeType_STOCK_CHANGE_TYPE_TRANSFER_WAREHOUSE_OUT_CANCELED:

		n = 1
		atField = "it.arrived as at"
		actorField = "it.verify_by_id as actor_id"
		err = tx.Raw(`
				select arrived from inv_transactions it where it.id = ?
			`, txId).
			Find(&timetx).
			Error

		if err != nil {
			return nil, err
		}

	case warehouse_iface.StockChangeType_STOCK_CHANGE_TYPE_ORDER_CANCELED:

		n = 1
		atField = `
		(
			select 
				timestamp
			from inv_timestamps ts
			where 
				ts.status = 'cancel'
				and ts.tx_id = iti.inv_transaction_id
		) as at
		`
		actorField = `
		(
			select 
				user_id
			from inv_timestamps ts
			where 
				ts.status = 'cancel'
				and ts.tx_id = iti.inv_transaction_id
		) as actor_id
		`

		err = tx.Raw(`
				select 
					timestamp
				from inv_timestamps ts
				where 
					ts.status = 'cancel'
					and ts.tx_id = ?
			`, txId).
			Find(&timetx).
			Error

		if err != nil {
			return nil, err
		}
	}

	changeCountField = fmt.Sprintf(changeCountField, n)
	changeAmountField = fmt.Sprintf(changeAmountField, n)

	logs := []*warehouse_iface.StockChangeLog{}

	logquery := tx.
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

	err = tx.
		Table("(?) as d", logquery).
		Where("d.change_count != 0").
		Find(&logs).
		Error

	if err != nil {
		return nil, err
	}

	if len(logs) == 0 {
		switch changeType {
		case warehouse_iface.StockChangeType_STOCK_CHANGE_TYPE_RESTOCK_ACCEPTED,
			warehouse_iface.StockChangeType_STOCK_CHANGE_TYPE_RETURN_ACCEPTED:
			return nil, nil
		default:
			return nil, fmt.Errorf("%d logs empty", txId)
		}

	}

	// sendedLog := []*warehouse_iface.StockChangeLog{}

	for _, log := range logs {
		log.TransactionAt = timestamppb.New(timetx)
		log.Type = changeType
		log.ExternalMsgId = externalMsgId

	}

	// if len(sendedLog) == 0 {
	// 	return nil
	// }

	// _, err = eventSender(ctx, &warehouse_iface.StockEvent{
	// 	Data: &warehouse_iface.StockEvent_StockChange{
	// 		StockChange: &warehouse_iface.StockChange{
	// 			Changes:     sendedLog,
	// 			CreatedTime: timestamppb.New(timetx),
	// 		},
	// 	},
	// })
	// if err != nil {
	// 	return err
	// }

	return &warehouse_iface.StockEvent{
		Data: &warehouse_iface.StockEvent_StockChange{
			StockChange: &warehouse_iface.StockChange{
				CreatedTime: timestamppb.New(timetx),
				Changes:     logs,
			},
		},
	}, nil
}

func getTransfer(tx *gorm.DB, transferId uint64) (*db_models.WarehouseTransfer, error) {
	res := db_models.WarehouseTransfer{}
	err := tx.
		Model(&db_models.WarehouseTransfer{}).
		First(&res, transferId).
		Error

	if err != nil {
		return nil, err
	}

	return &res, nil
}
