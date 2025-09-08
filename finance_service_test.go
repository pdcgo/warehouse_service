package warehouse_service_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/pdcgo/shared/authorization"
	"github.com/pdcgo/shared/db_models"
	"github.com/pdcgo/shared/interfaces/authorization_iface"
	"github.com/pdcgo/shared/interfaces/warehouse_iface"
	"github.com/pdcgo/shared/pkg/moretest"
	"github.com/pdcgo/shared/pkg/moretest/moretest_mock"
	"github.com/pdcgo/warehouse_service"
	"github.com/pdcgo/warehouse_service/models"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

func NewMockAuth(hasRole bool) authorization_iface.Authorization {
	return &mockAuth{
		hasRole: hasRole,
	}
}

type mockAuth struct {
	hasRole bool
}

// AuthIdentityFromToken implements authorization_iface.Authorization.
func (m *mockAuth) AuthIdentityFromToken(token string) authorization_iface.AuthIdentity {
	panic("unimplemented")
}

// AuthIdentityFromHeader implements authorization_iface.Authorization.
func (m *mockAuth) AuthIdentityFromHeader(header http.Header) authorization_iface.AuthIdentity {
	panic("unimplemented")
}

// ApiQueryCheckPermission implements authorization_iface.Authorization.
func (m *mockAuth) ApiQueryCheckPermission(identity authorization_iface.Identity, query authorization_iface.PermissionQuery) (bool, error) {
	panic("unimplemented")
}

// HasPermission implements authorization_iface.Authorization.
func (m *mockAuth) HasPermission(identity authorization_iface.Identity, perms authorization_iface.CheckPermissionGroup) error {
	if !m.hasRole {
		err := errors.New("not have role")
		return err
	}
	return nil
}

func TestFinanceService(t *testing.T) {
	var db gorm.DB

	var seedTeam = func(name string, teamType db_models.TeamType) *db_models.Team {
		code := strings.ToUpper(strings.ReplaceAll(name, " ", ""))
		team := &db_models.Team{
			Name:     name,
			Type:     teamType,
			TeamCode: db_models.TeamCode(code),
		}
		err := db.Create(team).Error
		assert.Nil(t, err)
		return team
	}

	var seedUser = func(team *db_models.Team, name string) *db_models.User {
		username := strings.ToLower(strings.ReplaceAll(name, " ", ""))
		user := &db_models.User{
			Name:     name,
			Username: username,
			Password: "123456",
			Email:    fmt.Sprintf("%s@gmail.com", username),
		}
		err := db.Create(user).Error
		assert.Nil(t, err)

		userTeam := &db_models.UserTeam{
			TeamID: team.ID,
			UserID: user.ID,
		}
		err = db.Create(userTeam).Error
		assert.Nil(t, err)

		return user
	}

	moretest.Suite(
		t,
		"test finance service",
		moretest.SetupListFunc{
			moretest_mock.MockSqliteDatabase(&db),
			func(t *testing.T) func() error {
				err := db.AutoMigrate(
					&models.WareExpenseAccount{},
					&models.WareExpenseAccountWarehouse{},
					&models.WareExpenseHistory{},
					&db_models.Team{},
					&db_models.User{},
					&db_models.UserTeam{},
				)
				assert.Nil(t, err)

				return nil
			},
		},
		func(t *testing.T) {
			warehouseTeam := seedTeam("wrehouse_team", db_models.WarehouseTeamType)
			warehouseUser := seedUser(warehouseTeam, "wh user 1")

			adminTeam := seedTeam("admin_team", db_models.AdminTeamType)
			adminUser := seedUser(adminTeam, "admin user")

			ctx := context.Background()

			t.Run("test create expense account", func(t *testing.T) {
				createAccountPayload := &warehouse_iface.ExpenseAccountCreateReq{
					DomainId:     uint64(adminTeam.ID),
					WarehouseId:  uint64(warehouseTeam.ID),
					NumberId:     "303054897877844124",
					Name:         "Sulistyowardoyo",
					IsOpsAccount: true,
				}

				auth := NewMockAuth(true)

				t.Run("test create expense account", func(t *testing.T) {
					service := warehouse_service.NewWarehouseFinanceService(&db, auth)
					nCtx := context.WithValue(ctx, "identity", &authorization.JwtIdentity{
						UserID: uint(adminUser.ID),
						From:   db_models.AdminTeamType,
					})

					account, err := service.ExpenseAccountCreate(nCtx, createAccountPayload)
					assert.Nil(t, err)

					t.Run("test recreate same account", func(t *testing.T) {
						_, err := service.ExpenseAccountCreate(nCtx, createAccountPayload)
						assert.NotNil(t, err)
					})

					t.Run("test get account", func(t *testing.T) {
						result, err := service.ExpenseAccountGet(nCtx, &warehouse_iface.ExpenseAccountGetReq{
							Id:           account.Id,
							WarehouseId:  uint64(warehouseTeam.ID),
							IsOpsAccount: true,
						})
						assert.Nil(t, err)
						assert.NotEmpty(t, result)
					})

					t.Run("test edit account", func(t *testing.T) {
						editPayload := &warehouse_iface.ExpenseAccountEditReq{
							AccountId:   account.Id,
							DomainId:    uint64(adminTeam.ID),
							WarehouseId: uint64(warehouseTeam.ID),
							NumberId:    "305055648774541",
							Name:        "Sulistyowardoyo Siswoyo",
						}

						t.Run("test edit number id and name", func(t *testing.T) {
							db.Transaction(func(tx *gorm.DB) error {
								service := warehouse_service.NewWarehouseFinanceService(tx, auth)

								nCtx := context.WithValue(ctx, "identity", &authorization.JwtIdentity{
									UserID: uint(adminTeam.ID),
									From:   db_models.AdminTeamType,
								})
								result, err := service.ExpenseAccountEdit(nCtx, editPayload)
								assert.Nil(t, err)

								assert.Equal(t, editPayload.NumberId, result.NumberId)
								assert.Equal(t, editPayload.Name, result.Name)
								return errors.New("dummy error")
							})
						})

						t.Run("test edit to ops account", func(t *testing.T) {
							createAccountPayload.NumberId = "789456132456"
							createAccountPayload.IsOpsAccount = false

							toOpsAccount, err := service.ExpenseAccountCreate(nCtx, createAccountPayload)
							assert.Nil(t, err)

							t.Run("test edit to ops account", func(t *testing.T) {
								editPayload.NumberId = toOpsAccount.NumberId
								editPayload.IsOpsAccount = true

								_, err := service.ExpenseAccountEdit(nCtx, editPayload)
								assert.NotNil(t, err)
							})
						})
					})
				})

				t.Run("test create from warehouse", func(t *testing.T) {
					db.Transaction(func(tx *gorm.DB) error {
						auth := NewMockAuth(false)
						service := warehouse_service.NewWarehouseFinanceService(tx, auth)

						nCtx := context.WithValue(ctx, "identity", &authorization.JwtIdentity{
							UserID: uint(warehouseUser.ID),
							From:   db_models.WarehouseTeamType,
						})
						_, err := service.ExpenseAccountCreate(nCtx, createAccountPayload)
						assert.NotNil(t, err)

						return errors.New("test error")
					})
				})

				t.Run("test expense list", func(t *testing.T) {
					auth := NewMockAuth(true)
					service := warehouse_service.NewWarehouseFinanceService(&db, auth)

					results, err := service.ExpenseAccountList(ctx, &warehouse_iface.ExpenseAccountListReq{
						WarehouseId: uint64(warehouseTeam.ID),
					})
					assert.Nil(t, err)
					assert.NotNil(t, results)
					assert.NotNil(t, results.Data)
				})
			})
		},
	)
}
