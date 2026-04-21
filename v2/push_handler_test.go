package warehouse_service_test

import (
	"testing"

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

func TestPushHandler(t *testing.T) {
	var db gorm.DB

	var migrate moretest.SetupFunc = func(t *testing.T) func() error {
		err := db.AutoMigrate(
			&db_models.Sku{},
			&warehouse_models.DailySkuHistory{},
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

			handler := warehouse_service.NewWarehousePushHandler(&db)

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
