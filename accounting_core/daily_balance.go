package accounting_core

import (
	"fmt"
	"time"

	"gorm.io/gorm"
)

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
		balance = entry.Debit - entry.Credit
	case CreditBalance:
		balance = entry.Credit - entry.Debit
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

// ------------------------------------------------- yg bawah yg baru ----------------------------------------

type DailyBalanceCalculate struct {
	tx         *gorm.DB
	labels     *TxLabelExtra
	accountMap AccountCache
}

func (d *DailyBalanceCalculate) UpdateDaily(
	entryfunc func() (
		entries JournalEntriesList,
		accMap AccountCache,
		err error,
	),
) error {
	var err error

	entries, accMap, err := entryfunc()
	if err != nil {
		return err
	}

	for _, entry := range entries {
		day := ParseDate(entry.EntryTime)
		var balance float64
		account, err := accMap.Get(entry.AccountID)
		if err != nil {
			return err
		}

		switch account.BalanceType {
		case DebitBalance:
			balance = entry.Debit - entry.Credit
		case CreditBalance:
			balance = entry.Credit - entry.Debit
		}

		dayBalance := &AccountDailyBalance{
			Day:           day,
			AccountID:     entry.AccountID,
			JournalTeamID: entry.TeamID,
			Debit:         entry.Debit,
			Credit:        entry.Credit,
			Balance:       balance,
		}

		err = d.updateDailyBalance(
			d.
				tx.
				Model(&AccountDailyBalance{}).
				Where("day = ?", dayBalance.Day).
				Where("account_id = ?", dayBalance.AccountID).
				Where("journal_team_id = ?", dayBalance.JournalTeamID),
			dayBalance,
		)

		if err != nil {
			return err
		}

		if d.labels.CsID != 0 {
			csDayBalance := &CsDailyBalance{
				Day:           day,
				CsID:          d.labels.CsID,
				AccountID:     entry.AccountID,
				JournalTeamID: entry.TeamID,
				Debit:         entry.Debit,
				Credit:        entry.Credit,
				Balance:       balance,
			}

			err = d.updateDailyBalance(
				d.
					tx.
					Model(&CsDailyBalance{}).
					Where("cs_id = ?", csDayBalance.CsID).
					Where("day = ?", csDayBalance.Day).
					Where("account_id = ?", csDayBalance.AccountID).
					Where("journal_team_id = ?", csDayBalance.JournalTeamID),
				csDayBalance,
			)
			if err != nil {
				return err
			}
		}

		if d.labels.ShopID != 0 {
			shopDayBalance := &ShopDailyBalance{
				Day:           day,
				ShopID:        d.labels.ShopID,
				AccountID:     entry.AccountID,
				JournalTeamID: entry.TeamID,
				Debit:         entry.Debit,
				Credit:        entry.Credit,
				Balance:       balance,
			}

			err = d.updateDailyBalance(
				d.
					tx.
					Model(&ShopDailyBalance{}).
					Where("shop_id = ?", shopDayBalance.ShopID).
					Where("day = ?", shopDayBalance.Day).
					Where("account_id = ?", shopDayBalance.AccountID).
					Where("journal_team_id = ?", shopDayBalance.JournalTeamID),
				shopDayBalance,
			)
			if err != nil {
				return err
			}

		}

		if d.labels.SupplierID != 0 {
			supplierDayBalance := &SupplierDailyBalance{
				Day:           day,
				SupplierID:    d.labels.SupplierID,
				AccountID:     entry.AccountID,
				JournalTeamID: entry.TeamID,
				Debit:         entry.Debit,
				Credit:        entry.Credit,
				Balance:       balance,
			}

			err = d.updateDailyBalance(
				d.
					tx.
					Model(&SupplierDailyBalance{}).
					Where("supplier_id = ?", supplierDayBalance.SupplierID).
					Where("day = ?", supplierDayBalance.Day).
					Where("account_id = ?", supplierDayBalance.AccountID).
					Where("journal_team_id = ?", supplierDayBalance.JournalTeamID),
				supplierDayBalance,
			)

			if err != nil {
				return err
			}
		}

		if d.labels.TagIDs != nil {
			for _, tagID := range d.labels.TagIDs {

				customDayBalance := &CustomLabelDailyBalance{
					Day:           day,
					CustomID:      tagID,
					AccountID:     entry.AccountID,
					JournalTeamID: entry.TeamID,
					Debit:         entry.Debit,
					Credit:        entry.Credit,
					Balance:       balance,
				}

				err = d.updateDailyBalance(
					d.
						tx.
						Model(&CustomLabelDailyBalance{}).
						Where("custom_id = ?", customDayBalance.CustomID).
						Where("day = ?", customDayBalance.Day).
						Where("account_id = ?", customDayBalance.AccountID).
						Where("journal_team_id = ?", customDayBalance.JournalTeamID),
					customDayBalance,
				)

				if err != nil {
					return err
				}
			}
		}

	}

	return nil
}

type DailyBalance interface {
	AddBalance(balance float64)
	GetDebitCredit() (debit float64, credit float64, balance float64)
	Before(tx *gorm.DB, lock bool) *gorm.DB
	After(tx *gorm.DB, lock bool) *gorm.DB
	Empty() DailyBalance
}

func (d *DailyBalanceCalculate) updateDailyBalance(query *gorm.DB, daily DailyBalance) error {
	var err error

	debit, credit, balance := daily.GetDebitCredit()

	row := query.
		Updates(map[string]interface{}{
			"debit":   gorm.Expr("debit + ?", debit),
			"credit":  gorm.Expr("credit + ?", credit),
			"balance": gorm.Expr("balance + ?", balance),
		})

	if row.RowsAffected == 0 {
		err = row.Error
		if err != nil {
			return err
		}

		err = d.
			tx.
			Save(daily).
			Error

		if err != nil {
			return err
		}
	}

	return err

}

type TransactionCalculate interface {
	GetLabelExtra() *TxLabelExtra
}

func NewDailyBalanceCalculate(
	tx *gorm.DB,
	labels *TxLabelExtra,
) *DailyBalanceCalculate {
	return &DailyBalanceCalculate{
		tx:         tx,
		labels:     labels,
		accountMap: &DBAccount{},
	}
}
