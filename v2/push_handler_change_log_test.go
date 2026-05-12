package warehouse_service_test

import (
	"context"
	"testing"
	"time"

	"github.com/pdcgo/schema/services/warehouse_iface/v1"
	"github.com/pdcgo/shared/db_models"
	"github.com/pdcgo/shared/pkg/moretest"
	"github.com/pdcgo/shared/pkg/moretest/moretest_mock"
	"github.com/pdcgo/warehouse_service/v2"
	"github.com/pdcgo/warehouse_service/warehouse_models"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/proto"
	"gorm.io/gorm"
)

func TestChangeLogIntegrity(t *testing.T) {
	var dbScenario moretest_mock.DbScenario

	moretest.Suite(t, "test change log integrity",
		moretest.SetupListFunc{
			moretest_mock.MockPostgresDatabase(&dbScenario),
		},
		func(t *testing.T) {
			var err error

			dbScenario(t, func(tx *gorm.DB) {
				err = tx.AutoMigrate(
					&db_models.InvTxItem{},
					&warehouse_models.InvItemProblem{},
					&db_models.RestockCost{},
					&db_models.InvTransaction{},
					&warehouse_models.StockChangeLog{},
				)
				assert.NoError(t, err)

				items := []*db_models.InvTxItem{
					{
						ID:               5500489,
						InvTransactionID: 1593351,
						Owned:            false,

						SkuID: "224443625d076097",
						Count: 1,
						Price: 30200,
						Total: 30200,
					},
				}

				err = tx.Create(&items).Error
				assert.NoError(t, err)

				transaction := db_models.InvTransaction{

					ID:          1593351,
					TeamID:      63,
					WarehouseID: 67,
					Type:        db_models.InvTxOrder,
					Status:      db_models.InvTxCompleted,
					Created:     time.Now(),
				}

				err = tx.Create(&transaction).Error
				assert.NoError(t, err)

				var stockChange *warehouse_iface.StockChange

				err = warehouse_service.SendLog(
					t.Context(),
					tx,
					func(ctx context.Context, event proto.Message) (string, error) {

						assert.IsType(t, &warehouse_iface.StockEvent{}, event)

						eventD := event.(*warehouse_iface.StockEvent)

						stockChange = eventD.GetStockChange()

						return "", nil
					},
					"abc",
					uint64(1593351),
					warehouse_iface.StockChangeType_STOCK_CHANGE_TYPE_ORDER_ACCEPTED,
				)

				assert.Nil(t, err)
				assert.NotNil(t, stockChange)
				assert.Len(t, stockChange.Changes, 1)

				t.Run("testing idempotency", func(t *testing.T) {
					var stockChange *warehouse_iface.StockChange
					err = warehouse_service.SendLog(
						t.Context(),
						tx,
						func(ctx context.Context, event proto.Message) (string, error) {

							assert.IsType(t, &warehouse_iface.StockEvent{}, event)

							eventD := event.(*warehouse_iface.StockEvent)

							stockChange = eventD.GetStockChange()

							return "", nil
						},
						"abc",
						uint64(1593351),
						warehouse_iface.StockChangeType_STOCK_CHANGE_TYPE_ORDER_ACCEPTED,
					)

					assert.Nil(t, err)
					assert.Nil(t, stockChange)
					// assert.Len(t, stockChange.Changes, 0)

				})

				// t.Error("debug")
				// debugtool.LogJson(stockChange)
			})

		})
}
