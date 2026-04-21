package warehouse_service

import (
	"context"
	"net/http"

	"github.com/pdcgo/event_source"
	"github.com/pdcgo/schema/services/warehouse_iface/v1"
	"github.com/pdcgo/shared/db_models"
	"google.golang.org/protobuf/encoding/protojson"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type WarehousePushHandler event_source.PushHandler

func NewWarehousePushHandler(db *gorm.DB) WarehousePushHandler {
	return func(ctx context.Context, msg *event_source.PushRequest) error {
		var err error

		var event warehouse_iface.StockEvent
		err = protojson.Unmarshal(msg.Message.Data, &event)
		if err != nil {
			return err
		}

		db = db.WithContext(ctx)

		switch eventData := event.Data.(type) {
		case *warehouse_iface.StockEvent_StockChange:
			stockChange := eventData.StockChange

			for _, log := range stockChange.Changes {
				err = db.Transaction(func(tx *gorm.DB) error {
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
						select 
							sum(ih.count * -1)
						from invertory_histories ih 
						where 
							ih.tx_id is null
							and ih.sku_id = ?
						`,
						sku.ID,
					)

					initAmount := gorm.Expr(
						`
						select 
							sum(
								-1 * ih.count * (
									ih.price + coalesce(ih.ext_price, 0)
								)
							)
						from invertory_histories ih 
						where 
							ih.tx_id is null
							and ih.sku_id = ?
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
								end_stock_amount
							)
							values (
								date_trunc('day', @t ::timestamptz), 
								@sku_id, 
								@warehouse_id, 
								(@init_count), 
								(@init_count), 
								(@init_amount), 
								(@init_amount)
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
