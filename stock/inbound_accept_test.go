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
						// {
						// 	VariantId: 382,
						// 	Count:     24,
						// 	ItemPrice: 50357.142857142855,
						// },

						// {
						// 	Count:     6,
						// 	ItemPrice: 50357.142857142855,
						// 	VariantId: 11239,
						// },

						// {
						// 	VariantId: 3086,
						// 	ItemPrice: 45357.142857142855,
						// 	Count:     6,
						// },
						// {
						// 	ItemPrice: 38357.142857142855,
						// 	VariantId: 3814,
						// 	Count:     2,
						// },
						// {
						// 	Count:     6,
						// 	VariantId: 3347,
						// 	ItemPrice: 45357.142857142855,
						// },
					},
				},
			})

			assert.Nil(t, err)
		},
	)

}
