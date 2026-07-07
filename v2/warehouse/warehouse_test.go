package warehouse

import (
	"context"
	"io"
	"log/slog"
	"testing"

	"connectrpc.com/connect"
	role_base "github.com/pdcgo/schema/services/role_base/v1"
	warehouse_iface "github.com/pdcgo/schema/services/warehouse_iface/v1"
	"github.com/pdcgo/shared/db_models"
	"github.com/pdcgo/shared/pkg/moretest"
	"github.com/pdcgo/shared/pkg/moretest/moretest_mock"
	"github.com/pdcgo/user_service/user_models"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

// The streaming WarehouseCreate needs a connect stream + request headers, so it can't be
// driven directly here; instead its core (createWarehouseWithTeam) is exercised through the
// scenario transaction. The unary read/update/delete methods enforce auth via the access
// interceptor (proto request_policy), so the impl is called directly with a nil auth.
func TestWarehouseCrud(t *testing.T) {
	var scenario moretest_mock.DbScenario
	moretest.Suite(t, "warehouse crud",
		moretest.SetupListFunc{moretest_mock.MockPostgresDatabase(&scenario)},
		func(t *testing.T) {
			scenario(t, func(tx *gorm.DB) {
				err := tx.AutoMigrate(
					&db_models.Warehouse{},
					&db_models.Team{},
					&user_models.UserTeamRole{},
				)
				assert.NoError(t, err)

				svc := NewWarehouseService(tx)
				ctx := context.Background()

				const callerID uint = 42
				logger := slog.New(slog.NewTextHandler(io.Discard, nil))

				// Create: team + warehouse(id=team id) + owner, in one transaction.
				newID, err := createWarehouseWithTeam(tx, logger, callerID, &warehouse_iface.WarehouseCreateRequest{
					TeamCode:    "whmain",
					Name:        "Main WH",
					Desc:        "primary",
					Address:     "Jakarta",
					UseFixedFee: true,
					FeeFix:      5000,
					MaxFee:      10000,
					OpenTime:    "08:00",
					CloseTime:   "17:00",
				})
				assert.NoError(t, err)
				assert.Greater(t, newID, uint(0))

				// A warehouse team was created (type warehouse, code uppercased).
				var team db_models.Team
				err = tx.First(&team, newID).Error
				assert.NoError(t, err)
				assert.Equal(t, db_models.WarehouseTeamType, team.Type)
				assert.Equal(t, db_models.TeamCode("WHMAIN"), team.TeamCode)

				// The warehouse shares the team's id.
				var wh db_models.Warehouse
				err = tx.First(&wh, newID).Error
				assert.NoError(t, err)
				assert.Equal(t, "Main WH", wh.Name)

				// The caller is registered as team owner.
				var owner user_models.UserTeamRole
				err = tx.
					Where("team_id = ? AND user_id = ?", newID, callerID).
					First(&owner).
					Error
				assert.NoError(t, err)
				assert.Equal(t, role_base.Role_ROLE_WAREHOUSE_OWNER, owner.Role)

				id := uint64(newID)

				// Detail.
				det, err := svc.WarehouseDetail(ctx, connect.NewRequest(&warehouse_iface.WarehouseDetailRequest{Id: id}))
				assert.NoError(t, err)
				assert.Equal(t, "Main WH", det.Msg.Data.Name)
				assert.Equal(t, "08:00", det.Msg.Data.OpenTime)
				assert.True(t, det.Msg.Data.UseFixedFee)

				// List.
				list, err := svc.WarehouseList(ctx, connect.NewRequest(&warehouse_iface.WarehouseListRequest{}))
				assert.NoError(t, err)
				assert.Len(t, list.Msg.List, 1)

				// Update.
				_, err = svc.WarehouseUpdate(ctx, connect.NewRequest(&warehouse_iface.WarehouseUpdateRequest{
					Id:         id,
					Name:       "Main WH v2",
					Address:    "Bandung",
					IsClosed:   true,
					CloseOrder: "16:00",
				}))
				assert.NoError(t, err)

				det2, err := svc.WarehouseDetail(ctx, connect.NewRequest(&warehouse_iface.WarehouseDetailRequest{Id: id}))
				assert.NoError(t, err)
				assert.Equal(t, "Main WH v2", det2.Msg.Data.Name)
				assert.Equal(t, "Bandung", det2.Msg.Data.Address)
				assert.True(t, det2.Msg.Data.IsClosed)
				assert.Equal(t, "16:00", det2.Msg.Data.CloseOrder)

				// Soft delete → drops from list and detail 404s.
				_, err = svc.WarehouseDelete(ctx, connect.NewRequest(&warehouse_iface.WarehouseDeleteRequest{Id: id}))
				assert.NoError(t, err)

				list2, err := svc.WarehouseList(ctx, connect.NewRequest(&warehouse_iface.WarehouseListRequest{}))
				assert.NoError(t, err)
				assert.Len(t, list2.Msg.List, 0)

				_, err = svc.WarehouseDetail(ctx, connect.NewRequest(&warehouse_iface.WarehouseDetailRequest{Id: id}))
				assert.Error(t, err)
				assert.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
			})
		})
}
