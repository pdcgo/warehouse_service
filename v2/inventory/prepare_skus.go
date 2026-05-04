package inventory

import (
	"context"
	"time"

	"connectrpc.com/connect"
	"github.com/pdcgo/schema/services/warehouse_iface/v1"
	"github.com/pdcgo/shared/db_models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type skuLock struct {
	ID          db_models.SkuID
	WarehouseID int64
}

// PrepareSkus implements [warehouse_ifaceconnect.InventoryServiceHandler].
func (i *inventoryServiceImpl) PrepareSkus(
	ctx context.Context,
	req *connect.Request[warehouse_iface.PrepareSkusRequest],
) (*connect.Response[warehouse_iface.PrepareSkusResponse], error) {
	var err error
	result := &warehouse_iface.PrepareSkusResponse{}

	payload := req.Msg

	db := i.db.WithContext(ctx)

	err = db.Transaction(func(tx *gorm.DB) error {

		var skuLocks []skuLock
		err = tx.
			Clauses(clause.Locking{
				Strength: "UPDATE",
			}).
			Table("skus s").
			Where("id IN ?", payload.SkuIds).
			Select([]string{
				"s.id",
				"s.warehouse_id",
			}).
			Find(&skuLocks).
			Error

		if err != nil {
			return err
		}

		// preparing daily stock history if not exists
		for _, sku := range skuLocks {

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
				"t":            time.Now(),
				"sku_id":       sku.ID,
				"warehouse_id": sku.WarehouseID,
				"init_count":   initCount,
				"init_amount":  initAmount,
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
					on conflict (t, sku_id, warehouse_id) do nothing
				`, params).
				Error

			if err != nil {
				return err
			}

		}

		return nil
	})

	return connect.NewResponse(result), err
}
