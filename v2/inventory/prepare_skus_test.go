package inventory_test

import (
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/pdcgo/schema/services/warehouse_iface/v1"
	"github.com/pdcgo/shared/db_models"
	"github.com/pdcgo/shared/pkg/moretest"
	"github.com/pdcgo/shared/pkg/moretest/moretest_mock"
	"github.com/pdcgo/warehouse_service/v2/inventory"
	"github.com/pdcgo/warehouse_service/warehouse_models"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

func TestPrepareSkus(t *testing.T) {
	var dbScenario moretest_mock.DbScenario

	moretest.Suite(t, "test prepare sku",
		moretest.SetupListFunc{
			moretest_mock.MockPostgresDatabase(&dbScenario),
		},
		func(t *testing.T) {
			dbScenario(t, func(tx *gorm.DB) {
				var err error

				// migration first
				err = tx.AutoMigrate(
					&db_models.Sku{},
					&warehouse_models.DailySkuHistory{},
					&db_models.InvertoryHistory{},
				)
				assert.NoError(t, err)

				// creating sku
				err = tx.
					Create(&db_models.Sku{
						ID:          "11111111",
						WarehouseID: 1,
					}).
					Error

				assert.NoError(t, err)

				service := inventory.NewInventoryService(tx, nil)

				_, err = service.PrepareSkus(t.Context(), &connect.Request[warehouse_iface.PrepareSkusRequest]{
					Msg: &warehouse_iface.PrepareSkusRequest{
						SkuIds: []string{
							"11111111",
						},
					},
				})

				assert.NoError(t, err)

				t.Run("testing running kedua", func(t *testing.T) {

					_, err = service.PrepareSkus(t.Context(), &connect.Request[warehouse_iface.PrepareSkusRequest]{
						Msg: &warehouse_iface.PrepareSkusRequest{
							SkuIds: []string{
								"11111111",
							},
						},
					})

					assert.NoError(t, err)
				})

				t.Run("testing value", func(t *testing.T) {
					var dayHist warehouse_models.DailySkuHistory
					err = tx.
						Where("t = ?", time.Now().Format("2006-01-02")).
						Where("sku_id = ?", "11111111").
						First(&dayHist).Error

					assert.NoError(t, err)

					assert.NotEmpty(t, dayHist.ID)
					assert.Equal(t, db_models.SkuID("11111111"), dayHist.SkuID)
					assert.Equal(t, int64(0), dayHist.EndStockCount)
					assert.Equal(t, int64(0), dayHist.StartStockCount)
					assert.Equal(t, float64(0), dayHist.EndStockAmount)
					assert.Equal(t, float64(0), dayHist.StartStockAmount)

				})

			})

		},
	)
}
