package warehouse_mutations_test

import (
	"testing"

	"github.com/pdcgo/shared/db_models"
	"github.com/pdcgo/shared/identity/mock_identity"
	"github.com/pdcgo/shared/pkg/moretest"
	"github.com/pdcgo/shared/pkg/moretest/moretest_mock"
	"github.com/pdcgo/warehouse_service/warehouse_mutations"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

func TestTransactionLog(t *testing.T) {
	var db gorm.DB

	moretest.Suite(t, "testing log",
		moretest.SetupListFunc{
			moretest_mock.MockSqliteDatabase(&db),
			func(t *testing.T) func() error {
				err := db.AutoMigrate(&db_models.InvTimestamp{})
				assert.Nil(t, err)

				return nil
			},
		},
		func(t *testing.T) {
			agent := mock_identity.NewMockAgent(1, "test")
			err := warehouse_mutations.
				NewTransactionLogNewEntry(&db, agent).
				SetActionType(db_models.ActionEditPrice).
				SetBeforeUpdatedData(db_models.ActionEditPrice, 12000).
				SetTxID(1).
				Do()

			assert.Nil(t, err)

			t.Run("test get ulang", func(t *testing.T) {
				data := db_models.InvTimestamp{}
				err = db.Model(&db_models.InvTimestamp{}).First(&data).Error
				assert.Nil(t, err)

				assert.NotEmpty(t, data)
				assert.NotEmpty(t, data.ID)
			})
		},
	)
}
