package stock_test

import (
	"testing"

	"connectrpc.com/connect"
	"github.com/pdcgo/accounting_service/accounting_core"
	"github.com/pdcgo/accounting_service/accounting_mock"
	"github.com/pdcgo/accounting_service/stock"
	"github.com/pdcgo/schema/services/common/v1"
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

			res, err := srv.InboundAccept(t.Context(), &connect.Request[stock_iface.InboundAcceptRequest]{
				Msg: &stock_iface.InboundAcceptRequest{
					TeamId:      51,
					WarehouseId: 67,

					Source:      common.InboundSource_INBOUND_SOURCE_RESTOCK,
					ShippingFee: 70500,
					ExtTxId:     575611,
					Accepts: []*stock_iface.VariantItem{
						{
							VariantId: 180,
							Count:     36,
							ItemPrice: 45000,
						},
						{
							VariantId: 1820678,
							Count:     47,
							ItemPrice: 33000,
						},
					},
					Losts: []*stock_iface.VariantProblemItem{
						{
							VariantId: 20678,
							Count:     1,
							ItemPrice: 33000,
							Reason:    "problem lost_s on transaction 837036",
						},
					},
				},
			})

			assert.Nil(t, err)
			assert.NotEmpty(t, res.Msg.TransactionId)

			t.Run("testing total", func(t *testing.T) {
				var entries accounting_core.JournalEntriesList
				err = db.
					Model(&accounting_core.JournalEntry{}).
					Where("transaction_id = ?", res.Msg.TransactionId).
					Where("team_id = ?", 51).
					Find(&entries).
					Error

				assert.Nil(t, err)

				debit, credit := entries.DebitCredit()
				assert.True(t, accounting_core.CompareFloatSafe(3274500.00, debit, accounting_core.PrecisionEpsilon))
				assert.True(t, accounting_core.CompareFloatSafe(3274500.00, credit, accounting_core.PrecisionEpsilon))

				// entries.PrintJournalEntries(&db)
			})

			t.Run("test incident 25-09-2025", func(t *testing.T) {
				// gegara presisi yang beda
				_, err := srv.InboundAccept(t.Context(), &connect.Request[stock_iface.InboundAcceptRequest]{
					Msg: &stock_iface.InboundAcceptRequest{
						TeamId:      51,
						WarehouseId: 67,

						Source:      common.InboundSource_INBOUND_SOURCE_RESTOCK,
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

			t.Run("testing jika ada yang broken inciden 22-11-2025", func(t *testing.T) {
				_, err := srv.InboundAccept(t.Context(), &connect.Request[stock_iface.InboundAcceptRequest]{
					Msg: &stock_iface.InboundAcceptRequest{
						TeamId:      51,
						WarehouseId: 67,

						Source: common.InboundSource_INBOUND_SOURCE_RESTOCK,
						Brokens: []*stock_iface.VariantProblemItem{

							{
								Reason:    "problem broken_s on transaction 774925",
								VariantId: 9402,
								Count:     5,
								ItemPrice: 30000.695652173912,
							},
						},
						Extras: &stock_iface.StockInfoExtra{
							Receipt:     "G1 Aprilabu20Bora60Sailor80Cargodusty15B.pink20Puff150Rantaihtm20JodaHtm20SsnCho150Htm40",
							CreatedById: 536,
						},
						ShippingFee: 100,
						ExtTxId:     774925,
						Accepts: []*stock_iface.VariantItem{
							{
								VariantId: 13553,
								ItemPrice: 27000.695652173912,
								Count:     20,
							},
							{
								VariantId: 13856,
								ItemPrice: 25000.695652173912,
								Count:     60,
							},
							{
								VariantId: 6761,
								ItemPrice: 25000.695652173912,
								Count:     80,
							},
							{
								VariantId: 20466,
								ItemPrice: 26000.695652173912,
								Count:     15,
							},
							{
								VariantId: 5055,
								ItemPrice: 27500.695652173912,
								Count:     20,
							},
							{
								VariantId: 9402,
								ItemPrice: 30000.695652173912,
								Count:     145,
							},
							{
								VariantId: 17267,
								ItemPrice: 32000.695652173912,
								Count:     20,
							},
							{
								VariantId: 838,
								ItemPrice: 26000.695652173912,
								Count:     20,
							},
							{
								VariantId: 6689,
								ItemPrice: 30000.695652173912,
								Count:     150,
							},
							{
								VariantId: 2762,
								ItemPrice: 30000.695652173912,
								Count:     40,
							},
						},
					},
				})

				assert.Nil(t, err)
			})
		},
	)

}
