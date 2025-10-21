package accounting_core_test

import (
	"testing"
	"time"

	"github.com/pdcgo/accounting_service/accounting_core"
	"github.com/stretchr/testify/assert"
)

func TestEntriesAccountBalance(t *testing.T) {
	entries := accounting_core.JournalEntriesList{
		{
			AccountID:     1,
			TeamID:        1,
			TransactionID: 1,
			Debit:         12000,
			Credit:        0,
			Account: &accounting_core.Account{
				ID:          1,
				AccountKey:  accounting_core.StockReadyAccount,
				TeamID:      1,
				Coa:         10,
				BalanceType: accounting_core.DebitBalance,
			},
		},
		{
			AccountID:     2,
			TeamID:        1,
			TransactionID: 1,
			Debit:         0,
			Credit:        12000,
			Account: &accounting_core.Account{
				ID:          2,
				AccountKey:  accounting_core.StockPendingAccount,
				TeamID:      1,
				Coa:         10,
				BalanceType: accounting_core.DebitBalance,
			},
		},
	}

	changes, err := entries.AccountBalance()
	assert.Nil(t, err)

	assert.Equal(t, 12000.00, changes[1].Change())
	assert.Equal(t, -12000.00, changes[2].Change())

	entries = append(entries, &accounting_core.JournalEntry{
		AccountID:     1,
		TeamID:        1,
		TransactionID: 1,
		Debit:         0,
		Credit:        12000,
		Account: &accounting_core.Account{
			ID:          1,
			AccountKey:  accounting_core.StockReadyAccount,
			TeamID:      1,
			Coa:         10,
			BalanceType: accounting_core.DebitBalance,
		},
		Rollback: true,
	})

	entries = append(entries, &accounting_core.JournalEntry{
		AccountID:     2,
		TeamID:        1,
		TransactionID: 1,
		Debit:         12000,
		Credit:        0,
		Account: &accounting_core.Account{
			ID:          2,
			AccountKey:  accounting_core.StockPendingAccount,
			TeamID:      1,
			Coa:         10,
			BalanceType: accounting_core.DebitBalance,
		},
		Rollback: true,
	})

	changes, err = entries.AccountBalance()
	assert.Nil(t, err)

	assert.Equal(t, 0.00, changes[1].Change())
	assert.Equal(t, 0.00, changes[2].Change())

	entries = append(entries, &accounting_core.JournalEntry{
		AccountID:     1,
		TeamID:        1,
		TransactionID: 1,
		Debit:         15000,
		Credit:        0,
		Account: &accounting_core.Account{
			ID:          1,
			AccountKey:  accounting_core.StockReadyAccount,
			TeamID:      1,
			Coa:         10,
			BalanceType: accounting_core.DebitBalance,
		},
	})

	entries = append(entries, &accounting_core.JournalEntry{
		AccountID:     2,
		TeamID:        1,
		TransactionID: 1,
		Debit:         0,
		Credit:        15000,
		Account: &accounting_core.Account{
			ID:          2,
			AccountKey:  accounting_core.StockPendingAccount,
			TeamID:      1,
			Coa:         10,
			BalanceType: accounting_core.DebitBalance,
		},
	})

	changes, err = entries.AccountBalance()
	assert.Nil(t, err)

	assert.Equal(t, 15000.00, changes[1].Change())
	assert.Equal(t, -15000.00, changes[2].Change())

	entries = append(entries, &accounting_core.JournalEntry{
		AccountID:     1,
		TeamID:        1,
		TransactionID: 1,
		Debit:         0,
		Credit:        15000,
		Account: &accounting_core.Account{
			ID:          1,
			AccountKey:  accounting_core.StockReadyAccount,
			TeamID:      1,
			Coa:         10,
			BalanceType: accounting_core.DebitBalance,
		},
		Rollback: true,
	})

	entries = append(entries, &accounting_core.JournalEntry{
		AccountID:     2,
		TeamID:        1,
		TransactionID: 1,
		Debit:         15000,
		Credit:        0,
		Account: &accounting_core.Account{
			ID:          2,
			AccountKey:  accounting_core.StockPendingAccount,
			TeamID:      1,
			Coa:         10,
			BalanceType: accounting_core.DebitBalance,
		},
		Rollback: true,
	})

	changes, err = entries.AccountBalance()
	assert.Nil(t, err)

	assert.Equal(t, 0.00, changes[1].Change())
	assert.Equal(t, 0.00, changes[2].Change())

}

func TestModelJournalEntrieeList(t *testing.T) {
	entries := accounting_core.JournalEntriesList{
		{
			ID:            17,
			AccountID:     57,
			TeamID:        5,
			TransactionID: 7,
			CreatedByID:   1,
			EntryTime:     time.Now(),
			Credit:        46_000,
			Account: &accounting_core.Account{
				ID:          57,
				AccountKey:  accounting_core.CashAccount,
				TeamID:      5,
				Coa:         10,
				BalanceType: accounting_core.DebitBalance,
			},
		},
		{
			ID:            18,
			AccountID:     102,
			TeamID:        5,
			TransactionID: 7,
			CreatedByID:   1,
			EntryTime:     time.Now(),
			Debit:         36_000,
			Account: &accounting_core.Account{
				ID:          102,
				AccountKey:  accounting_core.StockPendingAccount,
				TeamID:      4,
				Coa:         10,
				BalanceType: accounting_core.DebitBalance,
			},
		},
		{
			ID:            19,
			AccountID:     106,
			TeamID:        5,
			TransactionID: 7,
			CreatedByID:   1,
			EntryTime:     time.Now(),
			Debit:         10_000,
			Account: &accounting_core.Account{
				ID:          102,
				AccountKey:  accounting_core.StockPendingFeeAccount,
				TeamID:      4,
				Coa:         10,
				BalanceType: accounting_core.DebitBalance,
			},
		},
	}

	_, err := entries.AccountBalance()
	assert.Nil(t, err)

	// for _, ch := range accmap {
	// 	t.Error(ch.Account.AccountKey, ch.Change())
	// }
}
