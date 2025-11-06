package adjustment_test

import (
	"testing"

	"connectrpc.com/connect"
	"github.com/pdcgo/accounting_service/accounting_core"
	"github.com/pdcgo/accounting_service/accounting_mock"
	"github.com/pdcgo/accounting_service/adjustment"
	"github.com/pdcgo/schema/services/accounting_iface/v1"
	"github.com/pdcgo/schema/services/common/v1"
	"github.com/pdcgo/shared/authorization/authorization_mock"
	"github.com/pdcgo/shared/pkg/moretest"
	"github.com/pdcgo/shared/pkg/moretest/moretest_mock"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

func TestAccountAdjustment(t *testing.T) {
	var db gorm.DB

	var migrate moretest.SetupFunc = func(t *testing.T) func() error {
		err := db.AutoMigrate(
			&accounting_core.Account{},
			&accounting_core.Transaction{},
			&accounting_core.JournalEntry{},
			&accounting_core.AccountingTag{},
			&accounting_core.TransactionTag{},
			&accounting_core.TransactionCustomerService{},
		)

		assert.Nil(t, err)
		return nil
	}

	moretest.Suite(t, "testing accounting adjustment",
		moretest.SetupListFunc{
			moretest_mock.MockSqliteDatabase(&db),
			migrate,
			accounting_mock.PopulateAccountKey(&db, 5),
			accounting_mock.PopulateAccountKey(&db, 6),
		},
		func(t *testing.T) {

			adjSrv := adjustment.NewAdjustmentService(&db, &authorization_mock.EmptyAuthorizationMock{
				AuthIdentityMock: &authorization_mock.AuthIdentityMock{
					IdentityMock: &authorization_mock.IdentityMock{
						ID: 1,
					},
				},
			})

			t.Run("testing akun adjustment", func(t *testing.T) {

				// {
				//   "teamId": "2",
				//   "adjustments": [
				//     {
				//       "adjs": [
				//         {
				//           "accountKey": "payable",
				//           "bookeepingId": "6",
				//           "teamId": "5",
				//           "amount": 12333
				//         }
				//       ],
				//       "description": "asdads asdasd"
				//     }
				//   ],
				//   "requestFrom": "REQUEST_FROM_ADMIN"
				// }

				_, err := adjSrv.AccountAdjustment(t.Context(), &connect.Request[accounting_iface.AccountAdjustmentRequest]{
					Msg: &accounting_iface.AccountAdjustmentRequest{
						TeamId: 5,
						Adjustments: []*accounting_iface.AccountAdjustment{
							{
								Adjs: []*accounting_iface.AdjustmentItem{
									{
										AccountKey:   "payable",
										BookeepingId: 5,
										TeamId:       6,
										Amount:       12333,
									},
								},
								Description: "asdasdasdasdasd",
							},
						},
						RequestFrom: common.RequestFrom_REQUEST_FROM_ADMIN,
					},
				})

				assert.NotNil(t, err)
			})

		},
	)

}
