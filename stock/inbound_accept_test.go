package stock_test

import (
	"testing"

	"connectrpc.com/connect"
	"github.com/pdcgo/accounting_service/accounting_core"
	"github.com/pdcgo/accounting_service/accounting_mock"
	"github.com/pdcgo/accounting_service/stock"
	"github.com/pdcgo/schema/services/stock_iface/v1"
	"github.com/pdcgo/shared/authorization/authorization_mock"
	"github.com/pdcgo/shared/pkg/moretest"
	"github.com/pdcgo/shared/pkg/moretest/moretest_mock"
	"github.com/stretchr/testify/assert"

	"gorm.io/gorm"
)

func TestInboundAccept(t *testing.T) {
	var db gorm.DB

	var migrate moretest.SetupFunc = func(t *testing.T) func() error {
		err := db.AutoMigrate(
			&accounting_core.Transaction{},
			&accounting_core.JournalEntry{},
			&accounting_core.AccountDailyBalance{},
			&accounting_core.Account{},
			&accounting_core.TypeLabel{},
			&accounting_core.TransactionTypeLabel{},
			&accounting_core.TypeLabelDailyBalance{},
		)
		assert.Nil(t, err)
		return nil
	}

	moretest.Suite(t, "testing inbound accept",
		moretest.SetupListFunc{
			moretest_mock.MockSqliteDatabase(&db),
			migrate,
			accounting_mock.PopulateAccountKey(&db, 51),
			accounting_mock.PopulateAccountKey(&db, 67),
		},
		func(t *testing.T) {
			srv := stock.NewStockService(&db, &authorization_mock.EmptyAuthorizationMock{
				AuthIdentityMock: &authorization_mock.AuthIdentityMock{
					IdentityMock: &authorization_mock.IdentityMock{
						ID: 1,
					},
				},
			})

			_, err := srv.InboundAccept(t.Context(), &connect.Request[stock_iface.InboundAcceptRequest]{
				Msg: &stock_iface.InboundAcceptRequest{
					TeamId:      51,
					WarehouseId: 67,

					Source:      stock_iface.InboundSource_INBOUND_SOURCE_RESTOCK,
					ShippingFee: 20000,
					ExtTxId:     575611,
					Accepts: []*stock_iface.VariantItem{
						{
							VariantId: 11639,
							Count:     12,
							ItemPrice: 45357.142857142855,
						},
					},
				},
			})

			assert.Nil(t, err)

			t.Run("test incident 25-09-2025", func(t *testing.T) {
				// gegara presisi yang beda
				_, err := srv.InboundAccept(t.Context(), &connect.Request[stock_iface.InboundAcceptRequest]{
					Msg: &stock_iface.InboundAcceptRequest{
						TeamId:      51,
						WarehouseId: 67,

						Source:      stock_iface.InboundSource_INBOUND_SOURCE_RESTOCK,
						ShippingFee: 61000,
						ExtTxId:     575612,
						Accepts: []*stock_iface.VariantItem{
							{
								VariantId: 14328,
								Count:     30,
								ItemPrice: 50033.333333333336,
							},
						},
					},
				})

				assert.Nil(t, err)
			})
		},
	)

}
