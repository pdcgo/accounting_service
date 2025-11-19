package accounting_core_test

import (
	"testing"

	"github.com/pdcgo/accounting_service/accounting_core"
	"github.com/pdcgo/shared/pkg/moretest"
	"github.com/pdcgo/shared/pkg/moretest/moretest_mock"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

func TestJournalEntries(t *testing.T) {
	var db gorm.DB

	var migrate moretest.SetupFunc = func(t *testing.T) func() error {
		err := db.AutoMigrate(
			&accounting_core.JournalEntry{},
			&accounting_core.AccountDailyBalance{},
		)

		assert.Nil(t, err)

		return nil
	}

	var accounts moretest.SetupFunc = func(t *testing.T) func() error {

		err := accounting_core.
			NewCreateAccount(&db).
			Create(
				accounting_core.DebitBalance,
				accounting_core.ASSET,
				1,
				accounting_core.StockPendingAccount,
				"Pending Stock",
			)

		assert.Nil(t, err)

		err = accounting_core.
			NewCreateAccount(&db).
			Create(
				accounting_core.DebitBalance,
				accounting_core.ASSET,
				1,
				accounting_core.CashAccount,
				"Kas",
			)

		assert.Nil(t, err)
		return nil
	}

	moretest.Suite(t, "testing journal entries",
		moretest.SetupListFunc{
			moretest_mock.MockSqliteDatabase(&db),
			migrate,
			accounts,
		},
		func(t *testing.T) {

			err := accounting_core.OpenTransaction(t.Context(), &db, func(tx *gorm.DB, bookmng accounting_core.BookManage) error {
				tran := accounting_core.Transaction{
					ID: 1,
					RefID: accounting_core.NewRefID(&accounting_core.RefData{
						RefType: accounting_core.OrderRef,
						ID:      1,
					}),
					TeamID: 1,
					Desc:   "test creating transaction",
				}

				err := bookmng.
					NewTransaction().
					Create(&tran).
					Err()

				assert.Nil(t, err)

				entryCreate := bookmng.NewCreateEntry(1, 1)

				return entryCreate.
					To(&accounting_core.EntryAccountPayload{
						Key:    accounting_core.CashAccount,
						TeamID: 1,
					}, -1200).
					To(&accounting_core.EntryAccountPayload{
						Key:    accounting_core.StockPendingAccount,
						TeamID: 1,
					}, 1200).
					Transaction(&tran).
					Commit().
					Err()
			})

			assert.Nil(t, err)

			t.Run("testing getting entry", func(t *testing.T) {
				entries := []*accounting_core.JournalEntry{}

				err = db.
					Model(&accounting_core.JournalEntry{}).
					Where("transaction_id = ?", 1).
					Find(&entries).
					Error
				assert.Nil(t, err)

				assert.Len(t, entries, 2)

				// raw, _ := json.MarshalIndent(entries, "", "  ")
				// t.Error(string(raw))
			})

			t.Run("test rollback entry", func(t *testing.T) {
				err = accounting_core.
					NewTransactionMutation(t.Context(), &db).
					ByRefID(accounting_core.NewRefID(
						&accounting_core.RefData{
							RefType: accounting_core.OrderRef,
							ID:      1,
						},
					), true).
					RollbackEntry(1, "canceling").
					Err()
				assert.Nil(t, err)
			})

		},
	)
}
