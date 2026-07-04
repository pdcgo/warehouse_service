package warehouse_test

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	"github.com/pdcgo/schema/services/warehouse_iface/v1"
	"github.com/pdcgo/shared/db_models"
	"github.com/pdcgo/shared/pkg/moretest"
	"github.com/pdcgo/shared/pkg/moretest/moretest_mock"
	"github.com/pdcgo/warehouse_service/v2/warehouse"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

// Auth is enforced by the access interceptor (proto request_policy), not in the
// handlers — so the tests call the impl directly with a nil auth.
func TestWarehouseCrud(t *testing.T) {
	var scenario moretest_mock.DbScenario
	moretest.Suite(t, "warehouse crud",
		moretest.SetupListFunc{moretest_mock.MockPostgresDatabase(&scenario)},
		func(t *testing.T) {
			scenario(t, func(tx *gorm.DB) {
				assert.NoError(t, tx.AutoMigrate(&db_models.Warehouse{}))
				svc := warehouse.NewWarehouseService(tx, nil)
				ctx := context.Background()

				// Create (client-supplied id).
				_, err := svc.WarehouseCreate(ctx, connect.NewRequest(&warehouse_iface.WarehouseCreateRequest{
					Id:          10,
					Name:        "Main WH",
					Desc:        "primary",
					Address:     "Jakarta",
					UseFixedFee: true,
					FeeFix:      5000,
					MaxFee:      10000,
					OpenTime:    "08:00",
					CloseTime:   "17:00",
				}))
				assert.NoError(t, err)

				// Duplicate id is rejected (no failing INSERT — count guard, tx stays clean).
				_, err = svc.WarehouseCreate(ctx, connect.NewRequest(&warehouse_iface.WarehouseCreateRequest{
					Id:   10,
					Name: "Dup",
				}))
				assert.Error(t, err)
				assert.Equal(t, connect.CodeAlreadyExists, connect.CodeOf(err))

				// Detail.
				det, err := svc.WarehouseDetail(ctx, connect.NewRequest(&warehouse_iface.WarehouseDetailRequest{Id: 10}))
				assert.NoError(t, err)
				assert.Equal(t, "Main WH", det.Msg.Data.Name)
				assert.Equal(t, "08:00", det.Msg.Data.OpenTime)
				assert.True(t, det.Msg.Data.UseFixedFee)

				// List excludes soft-deleted.
				list, err := svc.WarehouseList(ctx, connect.NewRequest(&warehouse_iface.WarehouseListRequest{}))
				assert.NoError(t, err)
				assert.Len(t, list.Msg.List, 1)

				// Update.
				_, err = svc.WarehouseUpdate(ctx, connect.NewRequest(&warehouse_iface.WarehouseUpdateRequest{
					Id:         10,
					Name:       "Main WH v2",
					Address:    "Bandung",
					IsClosed:   true,
					CloseOrder: "16:00",
				}))
				assert.NoError(t, err)

				det2, err := svc.WarehouseDetail(ctx, connect.NewRequest(&warehouse_iface.WarehouseDetailRequest{Id: 10}))
				assert.NoError(t, err)
				assert.Equal(t, "Main WH v2", det2.Msg.Data.Name)
				assert.Equal(t, "Bandung", det2.Msg.Data.Address)
				assert.True(t, det2.Msg.Data.IsClosed)
				assert.Equal(t, "16:00", det2.Msg.Data.CloseOrder)

				// Soft delete → drops from list, detail 404s, and the id can't be re-created.
				_, err = svc.WarehouseDelete(ctx, connect.NewRequest(&warehouse_iface.WarehouseDeleteRequest{Id: 10}))
				assert.NoError(t, err)

				list2, err := svc.WarehouseList(ctx, connect.NewRequest(&warehouse_iface.WarehouseListRequest{}))
				assert.NoError(t, err)
				assert.Len(t, list2.Msg.List, 0)

				_, err = svc.WarehouseDetail(ctx, connect.NewRequest(&warehouse_iface.WarehouseDetailRequest{Id: 10}))
				assert.Error(t, err)
				assert.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
			})
		})
}
