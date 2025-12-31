package inventory_test

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	"github.com/pdcgo/schema/services/access_iface/v1"
	"github.com/pdcgo/schema/services/warehouse_iface/v1"
	"github.com/pdcgo/shared/authorization/authorization_mock"
	"github.com/pdcgo/shared/custom_connect"
	"github.com/pdcgo/shared/db_models"
	"github.com/pdcgo/shared/pkg/moretest"
	"github.com/pdcgo/shared/pkg/moretest/moretest_mock"
	"github.com/pdcgo/warehouse_service/v2/inventory"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

func TestBlacklistedSku(t *testing.T) {

	var db gorm.DB

	var migrate moretest.SetupFunc = func(t *testing.T) func() error {
		err := db.AutoMigrate(
			&db_models.Sku{},
		)
		assert.Nil(t, err)

		return nil
	}

	sku := db_models.Sku{
		ID:            db_models.SkuID("test-sku-1"),
		VariantID:     1,
		TeamID:        1,
		ProductID:     1,
		WarehouseID:   1,
		IsBlacklisted: false,

		Variant: &db_models.VariationValue{
			ID:        1,
			RefID:     db_models.RefID("test-ref-id-1"),
			ProductID: 1,
		},
	}
	sku2 := db_models.Sku{
		ID:            db_models.SkuID("test-sku-2"),
		VariantID:     2,
		TeamID:        2,
		ProductID:     2,
		WarehouseID:   2,
		IsBlacklisted: false,

		Variant: &db_models.VariationValue{
			ID:        2,
			RefID:     db_models.RefID("test-ref-id-2"),
			ProductID: 2,
		},
	}
	var init_sku moretest.SetupFunc = func(t *testing.T) func() error {
		err := db.Create(&[]db_models.Sku{sku, sku2}).Error
		assert.Nil(t, err)

		return nil
	}

	moretest.Suite(t, "TestBlacklistedSku",
		moretest.SetupListFunc{
			moretest_mock.MockSqliteDatabase(&db),
			migrate,
			init_sku,
		},
		func(t *testing.T) {

			authMock := authorization_mock.EmptyAuthorizationMock{}
			service := inventory.NewInventoryService(&db, &authMock)

			ctx := context.WithValue(context.TODO(), custom_connect.SourceKey, &access_iface.RequestSource{
				TeamId:      1,
				RequestFrom: access_iface.RequestFrom_REQUEST_FROM_SELLING,
			})

			t.Run("success add sku is blacklist", func(t *testing.T) {

				skus := []string{sku.ID.String(), sku2.ID.String()}
				req := connect.NewRequest(&warehouse_iface.BlacklistedSkuAddRequest{
					Skus: skus,
				})

				_, err := service.BlacklistedSkuAdd(ctx, req)
				assert.NoError(t, err)

				var blacklistSkus []db_models.Sku
				err = db.Where("id IN ?", skus).Find(&blacklistSkus).Error
				assert.NoError(t, err)

				assert.Len(t, blacklistSkus, 2)
				assert.ElementsMatch(t, skus, []string{blacklistSkus[0].ID.String(), blacklistSkus[1].ID.String()})
				assert.ElementsMatch(t, []bool{true, true}, []bool{blacklistSkus[0].IsBlacklisted, blacklistSkus[1].IsBlacklisted})

				t.Run("success list sku is blacklisted", func(t *testing.T) {

					req := connect.NewRequest(&warehouse_iface.BlacklistedSkuRequest{
						Skus: skus,
					})

					res, err := service.BlacklistedSku(ctx, req)
					assert.NoError(t, err)

					data := res.Msg.Data
					assert.Len(t, data, 2)
					assert.ElementsMatch(t, []bool{true, true}, []bool{data[sku.ID.String()].IsBlacklisted, data[sku2.ID.String()].IsBlacklisted})
				})
			})

			t.Run("success remove sku is blacklist", func(t *testing.T) {

				skus := []string{sku.ID.String(), sku2.ID.String()}
				req := connect.NewRequest(&warehouse_iface.BlacklistedSkuRemoveRequest{
					Skus: skus,
				})

				_, err := service.BlacklistedSkuRemove(ctx, req)
				assert.NoError(t, err)

				var unblacklistSkus []db_models.Sku
				err = db.Where("id IN ?", skus).Find(&unblacklistSkus).Error
				assert.NoError(t, err)

				assert.Len(t, unblacklistSkus, 2)
				assert.ElementsMatch(t, skus, []string{unblacklistSkus[0].ID.String(), unblacklistSkus[1].ID.String()})
				assert.ElementsMatch(t, []bool{false, false}, []bool{unblacklistSkus[0].IsBlacklisted, unblacklistSkus[1].IsBlacklisted})

				t.Run("success list sku is unblacklisted", func(t *testing.T) {

					req := connect.NewRequest(&warehouse_iface.BlacklistedSkuRequest{
						Skus: skus,
					})

					res, err := service.BlacklistedSku(ctx, req)
					assert.NoError(t, err)

					data := res.Msg.Data
					assert.Len(t, data, 2)
					assert.ElementsMatch(t, []bool{false, false}, []bool{data[sku.ID.String()].IsBlacklisted, data[sku2.ID.String()].IsBlacklisted})
				})
			})

		},
	)
}
