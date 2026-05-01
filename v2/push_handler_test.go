package warehouse_service_test

import (
	"errors"
	"testing"

	"github.com/pdcgo/event_source"
	"github.com/pdcgo/event_source/event_source_mock"
	"github.com/pdcgo/schema/services/warehouse_iface/v1"
	"github.com/pdcgo/shared/db_models"
	"github.com/pdcgo/shared/pkg/debugtool"
	"github.com/pdcgo/shared/pkg/moretest"
	"github.com/pdcgo/shared/pkg/moretest/moretest_mock"
	"github.com/pdcgo/warehouse_service/v2"
	"github.com/pdcgo/warehouse_service/warehouse_models"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/gorm"
)

func TestPushIntegrity(t *testing.T) {
	var db gorm.DB

	var migrate moretest.SetupFunc = func(t *testing.T) func() error {
		err := db.AutoMigrate(
			&db_models.Sku{},
			&warehouse_models.DailySkuHistory{},
			&db_models.InvertoryHistory{},
		)
		assert.NoError(t, err)

		return nil
	}
	var seed moretest.SetupFunc = func(t *testing.T) func() error {
		err := db.
			Save(&db_models.Sku{
				ID:           "11111111",
				VariantID:    1,
				TeamID:       1,
				ProductID:    1,
				WarehouseID:  1,
				StockReady:   0,
				StockPending: 0,
				StockTotal:   0,
			}).
			Error
		assert.NoError(t, err)

		// seed invertory histories
		err = db.Save(&db_models.InvertoryHistory{
			SkuID: db_models.SkuID("11111111"),
			Count: -1,
			Price: 1000,
		}).Error
		assert.NoError(t, err)

		return nil
	}

	moretest.Suite(t, "testing push integrity",
		moretest.SetupListFunc{
			moretest_mock.MockPostgresDatabase(&db),
			migrate,
			seed,
		},
		func(t *testing.T) {
			var err error

			t.Run("start with negative", func(t *testing.T) {
				db.Transaction(func(tx *gorm.DB) error {
					handler := warehouse_service.NewWarehousePushHandler(tx, event_source.EmptySender)

					event := event_source_mock.NewMockEvent(t, &warehouse_iface.StockEvent{
						Data: &warehouse_iface.StockEvent_StockChange{
							StockChange: &warehouse_iface.StockChange{
								CreatedTime: timestamppb.Now(),
								Changes: []*warehouse_iface.StockChangeLog{
									{
										SkuId:         "11111111",
										WarehouseId:   1,
										ChangeCount:   -1,
										ChangeAmount:  -1000,
										ActorId:       1,
										TransactionId: 1,
									},
								},
							},
						},
					})

					err = handler(t.Context(), event)
					assert.NoError(t, err)

					t.Run("check value", func(t *testing.T) {
						t.Run("history must one", func(t *testing.T) {
							var histories []warehouse_models.DailySkuHistory
							err = tx.Model(warehouse_models.DailySkuHistory{}).Where("sku_id = ?", "11111111").Find(&histories).Error
							assert.NoError(t, err)

							assert.Len(t, histories, 1)
						})

						history := warehouse_models.DailySkuHistory{}
						err = tx.Model(history).Where("sku_id = ?", "11111111").First(&history).Error
						assert.NoError(t, err)

						assert.Equal(t, int64(0), history.EndStockCount)
						assert.Equal(t, float64(0), history.EndStockAmount)
						assert.Equal(t, int64(-1), history.DiffStockCount)
						assert.Equal(t, float64(-1000), history.DiffStockAmount)
						assert.Equal(t, int64(0), history.StartStockCount)
						assert.Equal(t, float64(0), history.StartStockAmount)

						assert.Equal(t, history.DiffStockAmount, history.EndStockAmount-history.StartStockAmount)
						assert.Equal(t, history.DiffStockCount, history.EndStockCount-history.StartStockCount)

						debugtool.LogJson(history)

					})

					return errors.New("rollback")
				})
			})

			// t.Run("start with positive", func(t *testing.T) {

			// 	tx := db.Begin()
			// 	defer tx.Rollback()

			// 	handler := warehouse_service.NewWarehousePushHandler(tx, event_source.EmptySender)

			// 	event := event_source_mock.NewMockEvent(t, &warehouse_iface.StockEvent{
			// 		Data: &warehouse_iface.StockEvent_StockChange{
			// 			StockChange: &warehouse_iface.StockChange{
			// 				CreatedTime: timestamppb.Now(),
			// 				Changes: []*warehouse_iface.StockChangeLog{
			// 					{
			// 						SkuId:         "11111111",
			// 						WarehouseId:   1,
			// 						ChangeCount:   1,
			// 						ChangeAmount:  1000,
			// 						ActorId:       1,
			// 						TransactionId: 1,
			// 					},
			// 				},
			// 			},
			// 		},
			// 	})

			// 	err = handler(t.Context(), event)
			// 	assert.NoError(t, err)
			// 	err = handler(t.Context(), event)
			// 	assert.NoError(t, err)

			// 	t.Run("check daily sku histories", func(t *testing.T) {
			// 		t.Run("history must one", func(t *testing.T) {
			// 			var histories []warehouse_models.DailySkuHistory
			// 			err = tx.Model(warehouse_models.DailySkuHistory{}).Where("sku_id = ?", "11111111").Find(&histories).Error
			// 			assert.NoError(t, err)

			// 			assert.Len(t, histories, 1)
			// 		})

			// 		history := warehouse_models.DailySkuHistory{}
			// 		err = tx.Model(history).Where("sku_id = ?", "11111111").First(&history).Error
			// 		assert.NoError(t, err)

			// 		assert.Equal(t, int64(2), history.EndStockCount)
			// 		assert.Equal(t, float64(2000), history.EndStockAmount)
			// 		assert.Equal(t, int64(2), history.DiffStockCount)
			// 		assert.Equal(t, float64(2000), history.DiffStockAmount)

			// 		assert.Equal(t, history.DiffStockAmount, history.EndStockAmount-history.StartStockAmount)
			// 		assert.Equal(t, history.DiffStockCount, history.EndStockCount-history.StartStockCount)

			// 		debugtool.LogJson(history)

			// 	})
			// })

		},
	)
}

func TestPushHandler(t *testing.T) {
	var db gorm.DB

	var migrate moretest.SetupFunc = func(t *testing.T) func() error {
		err := db.AutoMigrate(
			&db_models.Sku{},
			&warehouse_models.DailySkuHistory{},
			&db_models.InvertoryHistory{},
		)
		assert.NoError(t, err)

		return nil
	}

	var seed moretest.SetupFunc = func(t *testing.T) func() error {
		err := db.Save(&db_models.Sku{
			ID:           "11111111",
			VariantID:    1,
			TeamID:       1,
			ProductID:    1,
			WarehouseID:  1,
			StockReady:   1,
			StockPending: 1,
			StockTotal:   1,
		}).Error
		assert.NoError(t, err)

		return nil
	}

	moretest.Suite(t, "testing push handler",
		moretest.SetupListFunc{
			moretest_mock.MockPostgresDatabase(&db),
			migrate,
			seed,
		},
		func(t *testing.T) {
			var err error

			handler := warehouse_service.NewWarehousePushHandler(&db, nil)

			event := event_source_mock.NewMockEvent(t, &warehouse_iface.StockEvent{
				Data: &warehouse_iface.StockEvent_StockChange{
					StockChange: &warehouse_iface.StockChange{
						CreatedTime: timestamppb.Now(),
						Changes: []*warehouse_iface.StockChangeLog{
							{
								SkuId:         "11111111",
								WarehouseId:   1,
								ChangeCount:   1,
								ChangeAmount:  100,
								ActorId:       1,
								TransactionId: 1,
							},
						},
					},
				},
			})

			err = handler(t.Context(), event)
			assert.NoError(t, err)

			var histories []warehouse_models.DailySkuHistory
			err = db.Where("sku_id = ?", "11111111").Find(&histories).Error
			assert.NoError(t, err)

			assert.NotEmpty(t, histories)
			debugtool.LogJson(histories)
		},
	)
}
