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

func TestUpdateInbound(t *testing.T) {
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
			&accounting_core.TransactionCustomerService{},
		)
		assert.Nil(t, err)
		return nil
	}

	moretest.Suite(t, "testing update restock",
		moretest.SetupListFunc{
			moretest_mock.MockSqliteDatabase(&db),
			migrate,
			accounting_mock.PopulateAccountKey(&db, 50),
			accounting_mock.PopulateAccountKey(&db, 51),
		},

		func(t *testing.T) {
			ctx := t.Context()

			stocksrv := stock.NewStockService(&db, &authorization_mock.EmptyAuthorizationMock{
				AuthIdentityMock: &authorization_mock.AuthIdentityMock{
					IdentityMock: &authorization_mock.IdentityMock{
						ID: 1,
					},
				},
			})

			var restockID uint64

			t.Run("testing create restock", func(t *testing.T) {
				res, err := stocksrv.InboundCreate(ctx, &connect.Request[stock_iface.InboundCreateRequest]{
					Msg: &stock_iface.InboundCreateRequest{
						TeamId:      50,
						WarehouseId: 51,
						ExtTxId:     1,
						Extras: &stock_iface.StockInfoExtra{
							CreatedById:     1,
							Receipt:         "testrestockreceipt",
							ExternalOrderId: "externalorder",
						},
						Source:        stock_iface.InboundSource_INBOUND_SOURCE_RESTOCK,
						PaymentMethod: stock_iface.PaymentMethod_PAYMENT_METHOD_CASH,
						ShippingFee:   10000,
						Products: []*stock_iface.VariantItem{
							{
								VariantId: 1,
								Count:     12,
								ItemPrice: 10000,
							},
						},
					},
				})

				assert.Nil(t, err)
				assert.NotEqual(t, 0, res.Msg.TransactionId)
				restockID = res.Msg.TransactionId

			})

			t.Run("testing update restock", func(t *testing.T) {
				_, err := stocksrv.InboundUpdate(ctx, &connect.Request[stock_iface.InboundUpdateRequest]{
					Msg: &stock_iface.InboundUpdateRequest{
						TeamId:        50,
						WarehouseId:   51,
						ExtTxId:       1,
						Source:        stock_iface.InboundSource_INBOUND_SOURCE_RESTOCK,
						PaymentMethod: stock_iface.PaymentMethod_PAYMENT_METHOD_CASH,
						ShippingFee:   12000,
						Products: []*stock_iface.VariantItem{
							{
								VariantId: 1,
								Count:     12,
								ItemPrice: 12000,
							},
						},
					},
				})

				assert.Nil(t, err)
				t.Run("testing data", func(t *testing.T) {
					var entries accounting_core.JournalEntriesList
					err = db.
						Model(&accounting_core.JournalEntry{}).
						Preload("Account").
						Where("team_id = ?", 50).
						Where("transaction_id = ?", restockID).
						Order("id asc").
						Find(&entries).
						Error

					assert.Nil(t, err)

					var rollbackCount int
					for _, entry := range entries {
						if entry.Rollback {
							rollbackCount++
						}
					}
					assert.Equal(t, 2, rollbackCount)
					// entries.PrintJournalEntries(&db)
					// // balance
					// debugtool.LogJson(entries.AccountBalance())

				})
			})

		},
	)

}
