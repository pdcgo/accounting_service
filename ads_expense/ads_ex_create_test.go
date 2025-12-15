package ads_expense_test

import (
	"testing"

	"connectrpc.com/connect"
	"github.com/pdcgo/accounting_service/accounting_core"
	"github.com/pdcgo/accounting_service/accounting_mock"
	"github.com/pdcgo/accounting_service/ads_expense"
	"github.com/pdcgo/schema/services/access_iface/v1"
	"github.com/pdcgo/schema/services/accounting_iface/v1"
	"github.com/pdcgo/schema/services/common/v1"
	"github.com/pdcgo/shared/authorization/authorization_mock"
	"github.com/pdcgo/shared/custom_connect"
	"github.com/pdcgo/shared/db_models"
	"github.com/pdcgo/shared/pkg/moretest"
	"github.com/pdcgo/shared/pkg/moretest/moretest_mock"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

func TestCreateAdsExpense(t *testing.T) {
	var db gorm.DB

	moretest.Suite(t, "testing create expense iklan",
		moretest.SetupListFunc{
			moretest_mock.MockSqliteDatabase(&db),
			func(t *testing.T) func() error { // migrating
				err := db.AutoMigrate(
					&accounting_core.Account{},
					&accounting_core.Transaction{},
					&accounting_core.JournalEntry{},
					&accounting_core.AccountingTag{},
					&accounting_core.TransactionTag{},
					&accounting_core.TransactionShop{},
					&accounting_core.TypeLabel{},
					&accounting_core.TransactionTypeLabel{},
					&db_models.Marketplace{},
				)
				assert.Nil(t, err)
				return nil
			},
			func(t *testing.T) func() error {
				marketplace := db_models.Marketplace{
					ID:         2,
					TeamID:     1,
					MpUsername: "usertestmp",
					MpName:     "test mp",
				}

				err := db.Save(&marketplace).Error
				assert.Nil(t, err)

				return nil
			},
			accounting_mock.PopulateAccountKey(&db, 1),
		},
		func(t *testing.T) {
			service := ads_expense.NewAdsExpenseService(&db, &authorization_mock.EmptyAuthorizationMock{
				AuthIdentityMock: &authorization_mock.AuthIdentityMock{
					IdentityMock: &authorization_mock.IdentityMock{
						ID: 1,
					},
				},
			})

			ctx := custom_connect.SetRequestSource(t.Context(), &access_iface.RequestSource{
				TeamId:      1,
				RequestFrom: access_iface.RequestFrom_REQUEST_FROM_SELLING,
			})

			payload := &accounting_iface.AdsExCreateRequest{
				TeamId:        1,
				ShopId:        2,
				ExternalRefId: "asdasdasdasdasd",
				Source:        accounting_iface.AccountSource_ACCOUNT_SOURCE_SHOP,
				MpType:        common.MarketplaceType_MARKETPLACE_TYPE_SHOPEE,
				Amount:        120000,
				Desc:          "gmv payment",
			}

			res, err := service.AdsExCreate(ctx, &connect.Request[accounting_iface.AdsExCreateRequest]{
				Msg: payload,
			})
			assert.Nil(t, err)

			assert.NotEmpty(t, res.Msg.TransactionId)
			t.Run("testing entries", func(t *testing.T) {
				entries := accounting_core.JournalEntriesList{}

				err := db.
					Model(&accounting_core.JournalEntry{}).
					Preload("Account").
					Where("transaction_id = ?", res.Msg.TransactionId).
					Find(&entries).
					Error
				assert.Nil(t, err)
				assert.Len(t, entries, 2)

				for _, entry := range entries {
					switch entry.Account.AccountKey {
					case accounting_core.SellingReceivableAccount:
						assert.Equal(t, 120000.00, entry.Credit)

					}
				}

			})

			t.Run("testing run twice with same ext ref", func(t *testing.T) {
				_, err := service.AdsExCreate(ctx, &connect.Request[accounting_iface.AdsExCreateRequest]{
					Msg: payload,
				})
				assert.Nil(t, err)
			})

		},
	)
}
