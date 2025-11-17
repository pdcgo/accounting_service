package payment_transaction_test

import (
	"testing"

	"github.com/pdcgo/accounting_service/accounting_core"
	"github.com/pdcgo/accounting_service/accounting_transaction/payment_transaction"
	"github.com/pdcgo/shared/identity/mock_identity"
	"github.com/pdcgo/shared/pkg/moretest"
	"github.com/pdcgo/shared/pkg/moretest/moretest_mock"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

func TestStockOps(t *testing.T) {
	var db gorm.DB

	var migrate moretest.SetupFunc = func(t *testing.T) func() error {
		err := db.AutoMigrate(
			&accounting_core.Transaction{},
			&accounting_core.JournalEntry{},
			&accounting_core.AccountDailyBalance{},
		)

		assert.Nil(t, err)
		return nil
	}

	moretest.Suite(t, "testing stock operation",
		moretest.SetupListFunc{
			moretest_mock.MockSqliteDatabase(&db),
			migrate,
			func(t *testing.T) func() error { // seeding account
				accounts := []*accounting_core.Account{
					{
						AccountKey:  accounting_core.CashAccount,
						TeamID:      1,
						Coa:         accounting_core.ASSET,
						BalanceType: accounting_core.DebitBalance,
					},
					{
						AccountKey:  accounting_core.CashAccount,
						TeamID:      2,
						Coa:         accounting_core.ASSET,
						BalanceType: accounting_core.DebitBalance,
					},
					{
						AccountKey:  accounting_core.PayableAccount,
						TeamID:      2,
						Coa:         accounting_core.ASSET,
						BalanceType: accounting_core.DebitBalance,
					},
					{
						AccountKey:  accounting_core.ReceivableAccount,
						TeamID:      1,
						Coa:         accounting_core.ASSET,
						BalanceType: accounting_core.DebitBalance,
					},
				}

				err := db.Save(&accounts).Error
				assert.Nil(t, err)
				return nil
			},
		},
		func(t *testing.T) {
			agent := mock_identity.NewMockAgent(1, "test")
			paymentOps := payment_transaction.NewPaymentTransaction(t.Context(), &db, agent)

			t.Run("testing payment", func(t *testing.T) {
				err := paymentOps.Payment(&payment_transaction.PaymentPayload{
					FromTeamID: 1,
					ToTeamID:   2,
					Desc:       "pembayaran Fee Cod",
					Amount:     12000,
				})

				assert.Nil(t, err)
			})
		},
	)

}
