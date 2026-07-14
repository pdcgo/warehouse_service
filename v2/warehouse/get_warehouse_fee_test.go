package warehouse_test

import (
	"testing"

	"connectrpc.com/connect"
	warehouse_iface "github.com/pdcgo/schema/services/warehouse_iface/v1"
	"github.com/pdcgo/shared/db_models"
	"github.com/pdcgo/shared/pkg/moretest"
	"github.com/pdcgo/shared/pkg/moretest/moretest_mock"
	"github.com/pdcgo/warehouse_service/v2/warehouse"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

// GetWarehouseFee ports the legacy Warehouse.GetWarehouseFee formula over the fee
// config this service manages: flat FeeFix, OR percent-of-value rounded up to 100s
// capped at MaxFee.
func TestGetWarehouseFee(t *testing.T) {
	var scenario moretest_mock.DbScenario
	moretest.Suite(t, "get warehouse fee",
		moretest.SetupListFunc{moretest_mock.MockPostgresDatabase(&scenario)},
		func(t *testing.T) {
			scenario(t, func(tx *gorm.DB) {
				assert.NoError(t, tx.AutoMigrate(&db_models.Warehouse{}))
				svc := warehouse.NewWarehouseService(tx)

				assert.NoError(t, tx.Create(&[]db_models.Warehouse{
					{ID: 1, Name: "Fixed", UseFixedFee: true, FeeFix: 1500},
					{ID: 2, Name: "Percent", FeePercent: 2, MaxFee: 5000},
					{ID: 3, Name: "Free", FeePercent: 0},
					{ID: 4, Name: "Broken", UseFixedFee: true, FeeFix: 0},
					{ID: 5, Name: "Gone", Deleted: true},
				}).Error)

				fee := func(id uint64, value float64) (float64, error) {
					res, err := svc.GetWarehouseFee(t.Context(), connect.NewRequest(&warehouse_iface.GetWarehouseFeeRequest{
						WarehouseId: id,
						OrderValue:  value,
					}))
					if err != nil {
						return 0, err
					}
					return res.Msg.Fee, nil
				}

				// Fixed fee: flat regardless of value.
				f, err := fee(1, 123456)
				assert.NoError(t, err)
				assert.InDelta(t, 1500, f, 0.001)

				// Percent: ceil(100000 × 2 × 0.01) × 100 = 2000 × 100 → capped at 5000.
				f, err = fee(2, 100000)
				assert.NoError(t, err)
				assert.InDelta(t, 5000, f, 0.001)

				// Percent under the cap: ceil(1000 × 0.02) × 100 = 20 × 100 = 2000.
				f, err = fee(2, 1000)
				assert.NoError(t, err)
				assert.InDelta(t, 2000, f, 0.001)

				// No percent configured → 0 (free).
				f, err = fee(3, 50000)
				assert.NoError(t, err)
				assert.InDelta(t, 0, f, 0.001)

				// Misconfigured fixed fee → FailedPrecondition.
				_, err = fee(4, 1000)
				assert.Equal(t, connect.CodeFailedPrecondition, connect.CodeOf(err))

				// Deleted / unknown → NotFound.
				_, err = fee(5, 1000)
				assert.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
				_, err = fee(999, 1000)
				assert.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
			})
		})
}
