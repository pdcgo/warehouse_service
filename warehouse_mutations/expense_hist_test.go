package warehouse_mutations_test

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/pdcgo/shared/db_models"
	"github.com/pdcgo/shared/pkg/moretest"
	"github.com/pdcgo/shared/pkg/moretest/moretest_mock"
	"github.com/pdcgo/warehouse_service/models"
	"github.com/pdcgo/warehouse_service/warehouse_mutations"
	"github.com/pdcgo/warehouse_service/warehouse_query"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

func TestWareExpenseHistory(t *testing.T) {
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

	seedAccountType := func(key, name string, accountType db_models.ExpenseTypeAccount) *db_models.AccountType {
		accType := db_models.AccountType{
			Key:  key,
			Name: name,
			Type: accountType,
		}
		err := db.Create(&accType).Error
		assert.Nil(t, err)

		return &accType
	}

	seedAccount := func(whID, accountType uint, name, numberID string) *models.WareExpenseAccountWarehouse {
		account := models.WareExpenseAccount{
			AccountTypeID: 1,
			Name:          name,
			NumberID:      numberID,
			CreatedAt:     time.Now(),
		}
		err := db.Create(&account).Error
		assert.Nil(t, err)

		wareAccount := models.WareExpenseAccountWarehouse{
			AccountID:   account.ID,
			WarehouseID: whID,
			Account:     &account,
		}
		err = db.Create(&wareAccount).Error
		assert.Nil(t, err)

		return &wareAccount
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
					&models.WareBalanceAccountHistory{},
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

			// adminTeam := seedTeam("admin_team", db_models.AdminTeamType)
			// adminUser := seedUser(adminTeam, "admin user")

			accountType := seedAccountType("bni", "BNI", db_models.BankTypeAccount)
			account := seedAccount(warehouseTeam.ID, accountType.ID, "TEST ACCOUNT", "123456798")

			t.Run("test create expense history from warehouse", func(t *testing.T) {
				expenseService := warehouse_mutations.NewExpenseHistService(&db, warehouseUser)
				_, err := expenseService.GetAccount(account.AccountID, warehouseTeam.ID)
				assert.Nil(t, err)

				err = expenseService.
					Create(warehouseTeam.Type, &warehouse_mutations.CreateExpensePayload{
						ExpenseType: models.ExpenseTypeOther,
						At:          time.Now(),
						Amount:      54_000,
						Note:        "expense daily",
					})
				assert.Nil(t, err)
			})

			t.Run("test edit expense history from warehouse", func(t *testing.T) {
				expenseQuery := warehouse_query.NewWarehouseExpenseQuery(&db, false)

				expense := models.WareExpenseHistory{}
				err := expenseQuery.
					FromAccount(account.AccountID).
					FromWarehouse(warehouseTeam.ID).
					CreatedBy(warehouseUser.ID).
					GetQuery().
					Find(&expense).
					Error
				assert.Nil(t, err)
				assert.NotEmpty(t, expense)

				expenseService := warehouse_mutations.NewExpenseHistService(&db, warehouseUser)

				_, err = expenseService.GetExpense(expense.ID)
				assert.Nil(t, err)

				err = expenseService.Update(warehouseTeam.Type, &warehouse_mutations.UpdateWareExpenseHistPayload{
					WarehouseID: warehouseTeam.ID,
					AccountID:   account.AccountID,
					CreatedByID: warehouseUser.ID,
					ExpenseType: models.ExpenseTypeKitchen,
					Amount:      51_000,
					Note:        "expense kitchen",
					At:          expense.At,
				})
				assert.Nil(t, err)

				t.Run("test data updated", func(t *testing.T) {
					expenseQuery := warehouse_query.NewWarehouseExpenseQuery(&db, false)

					expense := models.WareExpenseHistory{}
					err := expenseQuery.
						FromAccount(account.AccountID).
						FromWarehouse(warehouseTeam.ID).
						CreatedBy(warehouseUser.ID).
						GetQuery().
						Find(&expense).
						Error
					assert.Nil(t, err)
					assert.NotEmpty(t, expense)
					assert.Equal(t, float64(51_000), expense.Amount)
					assert.Equal(t, models.ExpenseTypeKitchen, expense.ExpenseType)
				})
			})
		},
	)
}
