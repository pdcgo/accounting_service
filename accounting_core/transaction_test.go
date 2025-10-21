package accounting_core_test

import (
	"testing"
	"time"

	"github.com/pdcgo/accounting_service/accounting_core"
	"github.com/pdcgo/accounting_service/accounting_mock"
	"github.com/pdcgo/shared/pkg/moretest"
	"github.com/pdcgo/shared/pkg/moretest/moretest_mock"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

func TestTransactionRollback(t *testing.T) {
	var db gorm.DB
	var migrate moretest.SetupFunc = func(t *testing.T) func() error {
		err := db.AutoMigrate(
			&accounting_core.Transaction{},
			&accounting_core.Account{},
			&accounting_core.JournalEntry{},
		)
		assert.Nil(t, err)

		return nil
	}

	moretest.Suite(t, "testing rollback",
		moretest.SetupListFunc{
			moretest_mock.MockSqliteDatabase(&db),
			migrate,
			accounting_mock.PopulateAccountKey(&db, 1),
		},
		func(t *testing.T) {
			ref := accounting_core.NewRefID(&accounting_core.RefData{
				RefType: accounting_core.OrderFundRef,
				ID:      1,
			})

			err := accounting_core.OpenTransaction(&db, func(tx *gorm.DB, bookmng accounting_core.BookManage) error {
				tran := accounting_core.Transaction{
					TeamID:  1,
					RefID:   ref,
					Created: time.Now(),
				}
				err := bookmng.
					NewTransaction().
					Create(&tran).
					Err()

				if err != nil {
					return err
				}

				err = bookmng.
					NewCreateEntry(1, 1).
					From(&accounting_core.EntryAccountPayload{
						Key:    accounting_core.StockPendingAccount,
						TeamID: 1,
					}, 1000000).
					To(&accounting_core.EntryAccountPayload{
						Key:    accounting_core.StockReadyAccount,
						TeamID: 1,
					}, 1000000).
					Transaction(&tran).
					Commit().
					Err()

				if err != nil {
					return err
				}

				return nil
			})

			assert.Nil(t, err)

			rollbackFunc := func(t *testing.T) {
				err := accounting_core.OpenTransaction(&db, func(tx *gorm.DB, bookmng accounting_core.BookManage) error {
					trmut := accounting_core.
						NewTransactionMutation(tx).
						ByRefID(ref, true)

					err = trmut.
						RollbackEntry(1, "testing rollback").
						CheckEntry().
						Err()

					if err != nil {
						return err
					}

					err = bookmng.
						NewCreateEntry(1, 1).
						From(&accounting_core.EntryAccountPayload{
							Key:    accounting_core.StockPendingAccount,
							TeamID: 1,
						}, 1200000).
						To(&accounting_core.EntryAccountPayload{
							Key:    accounting_core.StockReadyAccount,
							TeamID: 1,
						}, 1200000).
						Transaction(trmut.Data()).
						Commit().
						Err()

					if err != nil {
						return err
					}

					return err

				})

				assert.Nil(t, err)
			}

			t.Run("testing rollback", rollbackFunc)
			t.Run("testing rollback 2 kali", rollbackFunc)
			t.Run("testing rollback 3 kali", rollbackFunc)
			t.Run("testing rollback 4 kali", rollbackFunc)
			t.Run("testing rollback 5 kali", rollbackFunc)
			t.Run("testing entries", func(t *testing.T) {
				var entries accounting_core.JournalEntriesList
				err = db.
					Model(&accounting_core.JournalEntry{}).
					// Where("transaction_id = ?", ref).
					Order("id desc, account_id desc, rollback asc").
					Find(&entries).
					Error
				assert.Nil(t, err)

				for _, entry := range entries {
					if entry.Debit != 0 {
						assert.LessOrEqual(t, entry.Debit, 1200000.00)
					}

					if entry.Credit != 0 {
						assert.LessOrEqual(t, entry.Credit, 1200000.00)
					}
				}

				entries.PrintJournalEntries(&db)

			})
		},
	)
}
