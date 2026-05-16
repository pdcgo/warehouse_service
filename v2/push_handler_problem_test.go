package warehouse_service_test

import (
	"testing"

	"github.com/pdcgo/schema/services/warehouse_iface/v1"
	"github.com/pdcgo/shared/db_models"
	"github.com/pdcgo/shared/pkg/debugtool"
	"github.com/pdcgo/shared/pkg/moretest"
	"github.com/pdcgo/shared/pkg/moretest/moretest_mock"
	"github.com/pdcgo/warehouse_service/v2"
	"github.com/pdcgo/warehouse_service/warehouse_models"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

func TestProblem(t *testing.T) {
	var dbScenario moretest_mock.DbScenario
	moretest.Suite(t, "testing problem",
		moretest.SetupListFunc{
			moretest_mock.MockPostgresDatabase(&dbScenario),
		},
		func(t *testing.T) {
			dbScenario(t, func(tx *gorm.DB) {
				err := tx.AutoMigrate(
					db_models.InvTransaction{},
					db_models.InvTxItem{},
					db_models.RestockCost{},
					warehouse_models.InvItemProblem{},
				)
				assert.Nil(t, err)

				txs := []db_models.InvTransaction{
					{
						ID: 1605229,
					},
				}
				err = tx.Create(&txs).Error
				assert.Nil(t, err)

				txItems := []db_models.InvTxItem{
					{
						InvTransactionID: 1605229,
						SkuID:            "22335e2d1df238",
						Count:            1,
						Owned:            true,
						Price:            25366.25,
						Total:            25366.25,
					},
				}
				err = tx.Create(&txItems).Error
				assert.Nil(t, err)

				event, err := warehouse_service.CreateStockChangeLog(tx, "test", 1605229, warehouse_iface.StockChangeType_STOCK_CHANGE_TYPE_STOCK_PROBLEM)
				assert.Nil(t, err)
				assert.NotNil(t, event)
				t.Error("asdasd")
				debugtool.LogJson(event)

			})
		},
	)
}
