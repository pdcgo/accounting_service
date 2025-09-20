package accounting_core_test

import (
	"testing"
	"time"

	"github.com/pdcgo/accounting_service/accounting_core"
	"github.com/stretchr/testify/assert"
)

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
