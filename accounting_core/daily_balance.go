package accounting_core

import (
	"fmt"
	"time"

	"gorm.io/gorm"
)

type AccountCache interface {
	Get(accID uint) (*Account, error)
}

type DBAccount struct {
	Tx *gorm.DB
}

// Get implements AccountCache.
func (d *DBAccount) Get(accID uint) (*Account, error) {
	var acc Account
	err := d.Tx.Model(&Account{}).First(&acc, accID).Error
	return &acc, err
}

type MapAccount map[uint]*Account

// Get implements AccountCache.
func (m MapAccount) Get(accID uint) (*Account, error) {
	var err error
	account, ok := m[accID]
	if !ok {
		err = fmt.Errorf("update balance error acount not found %d", accID)
	}

	return account, err
}

type BeforeUpdateDaily func(daybalance *AccountDailyBalance) error

type BalanceCalculate struct {
	tx                *gorm.DB
	accountMap        AccountCache
	BeforeUpdateDaily BeforeUpdateDaily
}

func (b *BalanceCalculate) AddEntry(entry *JournalEntry) error {
	var err error

	y, m, d := entry.EntryTime.Date()
	day := time.Time{}
	day = day.AddDate(y-1, int(m)-1, d-1)

	var balance float64
	account, err := b.accountMap.Get(entry.AccountID)
	if err != nil {
		return err
	}

	switch account.BalanceType {
	case DebitBalance:
		balance = entry.Credit - entry.Debit
	case CreditBalance:
		balance = entry.Debit - entry.Credit
	}

	dayBalance := &AccountDailyBalance{
		Day:           day,
		AccountID:     entry.AccountID,
		JournalTeamID: entry.TeamID,
		Debit:         entry.Debit,
		Credit:        entry.Credit,
		Balance:       balance,
	}

	if b.BeforeUpdateDaily != nil {
		err = b.BeforeUpdateDaily(dayBalance)
		if err != nil {
			return err
		}
	}

	row := b.
		tx.
		Model(&AccountDailyBalance{}).
		Where("day = ?", dayBalance.Day).
		Where("account_id = ?", dayBalance.AccountID).
		Where("journal_team_id = ?", dayBalance.JournalTeamID).
		Updates(map[string]interface{}{
			"debit":   gorm.Expr("debit + ?", dayBalance.Debit),
			"credit":  gorm.Expr("credit + ?", dayBalance.Credit),
			"balance": gorm.Expr("balance + ?", dayBalance.Balance),
		})

	if row.RowsAffected == 0 {
		err = row.Error
		if err != nil {
			return err
		}

		err = b.
			tx.
			Save(dayBalance).
			Error

		if err != nil {
			return err
		}
	}

	return err
}

func NewBalanceCalculate(tx *gorm.DB, accmap AccountCache) *BalanceCalculate {
	return &BalanceCalculate{
		tx:         tx,
		accountMap: accmap,
	}
}
