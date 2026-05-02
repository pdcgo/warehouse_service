package warehouse_service_test

import (
	"context"
	"testing"

	"github.com/pdcgo/event_source"
	"github.com/pdcgo/event_source/event_source_mock"
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

func TestOrderAccepted(t *testing.T) {
	var dbscenario moretest_mock.DbScenario

	moretest.Suite(t, "testing order accepted",
		moretest.SetupListFunc{
			moretest_mock.MockPostgresDatabase(&dbscenario),
		},
		func(t *testing.T) {

			dbscenario(t, func(tx *gorm.DB) {
				var err error

				// migration
				err = tx.AutoMigrate(
					&db_models.InvTransaction{},
					&db_models.InvTxItem{},
					&warehouse_models.InvItemProblem{},
					&db_models.RestockCost{},
					&warehouse_models.StockChangeLog{},
				)
				assert.NoError(t, err)

				// seeding order
				orderTx := db_models.InvTransaction{
					ID:          1,
					WarehouseID: 1,
					CreateByID:  1,
					Items: db_models.InvItemList{
						{
							SkuID: "11111111",
							Count: 2,
							Price: 3000,
							Total: 6000,
						},
					},
				}

				err = tx.Create(&orderTx).Error
				assert.NoError(t, err)

				var handler warehouse_service.WarehousePushHandler
				var eventSender event_source.EventSender = func(ctx context.Context, event proto.Message) (string, error) {
					assert.IsType(t, &warehouse_iface.StockEvent{}, event)

					_, err := event_source.EmptySender(ctx, event)
					assert.NoError(t, err)

					msg := event.(*warehouse_iface.StockEvent)

					stockchange := msg.Data.(*warehouse_iface.StockEvent_StockChange)

					assert.Len(t, stockchange.StockChange.Changes, 1)
					change := stockchange.StockChange.Changes[0]
					assert.Equal(t, "11111111", change.SkuId)
					assert.Equal(t, uint64(1), change.WarehouseId)
					assert.Equal(t, uint64(1), change.ActorId)
					assert.Equal(t, uint64(1), change.TransactionId)
					assert.Equal(t, int32(-2), change.ChangeCount)
					assert.InEpsilon(t, float64(-6000), float64(change.ChangeAmount), 0.0001)

					return "", nil
				}
				handler = warehouse_service.NewWarehousePushHandler(tx, eventSender)

				event := event_source_mock.NewMockEvent(t, &warehouse_iface.StockEvent{
					Data: &warehouse_iface.StockEvent_OrderAccepted{
						OrderAccepted: &warehouse_iface.OrderAccepted{
							TransactionId: 1,
						},
					},
				})

				err = handler(t.Context(), event)
				assert.NoError(t, err)
			})

		},
	)
}
