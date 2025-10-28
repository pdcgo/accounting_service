package accounting_core

import (
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func ParseDate(t time.Time) time.Time {
	y, m, d := t.Date()
	day := time.Time{}
	day = day.AddDate(y-1, int(m)-1, d-1)

	return day
}

type AccountKeyDailyBalance struct {
	ID            uint       `json:"id" gorm:"primarykey"`
	Day           time.Time  `json:"day" gorm:"index:account_key_journal,unique"`
	JournalTeamID uint       `json:"journal_team_id" gorm:"index:account_key_journal,unique"`
	AccountKey    AccountKey `json:"account_key" gorm:"index:account_key_journal,unique"`
	Debit         float64    `json:"debit"`
	Credit        float64    `json:"credit"`
	Balance       float64    `json:"balance"`
}

// AddBalance implements DailyBalance.
func (a *AccountKeyDailyBalance) AddBalance(balance float64) {
	a.Balance += balance
}

// After implements DailyBalance.
func (a *AccountKeyDailyBalance) After(tx *gorm.DB, lock bool) *gorm.DB {
	if lock {
		tx = tx.
			Clauses(
				clause.Locking{
					Strength: "UPDATE",
				},
			)
	}

	return tx.
		Model(&AccountKeyDailyBalance{}).
		Where("day > ?", a.Day).
		Where("account_key = ?", a.AccountKey).
		Where("journal_team_id = ?", a.JournalTeamID)
}

// Before implements DailyBalance.
func (a *AccountKeyDailyBalance) Before(tx *gorm.DB, lock bool) *gorm.DB {
	if lock {
		tx = tx.
			Clauses(
				clause.Locking{
					Strength: "UPDATE",
				},
			)
	}

	return tx.
		Model(&AccountKeyDailyBalance{}).
		Where("day < ?", a.Day).
		Where("account_key = ?", a.AccountKey).
		Where("journal_team_id = ?", a.JournalTeamID)
}

// Empty implements DailyBalance.
func (a *AccountKeyDailyBalance) Empty() DailyBalance {
	return &AccountKeyDailyBalance{}
}

// GetDebitCredit implements DailyBalance.
func (a *AccountKeyDailyBalance) GetDebitCredit() (debit float64, credit float64, balance float64) {
	return a.Debit, a.Credit, a.Balance
}

type AccountDailyBalance struct {
	ID            uint      `json:"id" gorm:"primarykey"`
	Day           time.Time `json:"day" gorm:"index:account_journal,unique"`
	AccountID     uint      `json:"account_id" gorm:"index:account_journal,unique"`
	JournalTeamID uint      `json:"journal_team_id" gorm:"index:account_journal,unique"`
	Debit         float64   `json:"debit"`
	Credit        float64   `json:"credit"`
	Balance       float64   `json:"balance"`

	Account *Account `gorm:"-"`
}

// AddBalance implements DailyBalance.
func (a *AccountDailyBalance) AddBalance(balance float64) {
	a.Balance += balance
}

// Empty implements report.DailyBalance.
func (a *AccountDailyBalance) Empty() DailyBalance {
	return &AccountDailyBalance{}
}

// After implements report.DailyBalance.
func (a *AccountDailyBalance) After(tx *gorm.DB, lock bool) *gorm.DB {
	if lock {
		tx = tx.
			Clauses(
				clause.Locking{
					Strength: "UPDATE",
				},
			)
	}

	return tx.
		Model(&AccountDailyBalance{}).
		Where("day > ?", a.Day).
		Where("account_id = ?", a.AccountID).
		Where("journal_team_id = ?", a.JournalTeamID)
}

// Before implements report.DailyBalance.
func (a *AccountDailyBalance) Before(tx *gorm.DB, lock bool) *gorm.DB {
	if lock {
		tx = tx.
			Clauses(
				clause.Locking{
					Strength: "UPDATE",
				},
			)
	}

	return tx.
		Model(&AccountDailyBalance{}).
		Where("day < ?", a.Day).
		Where("account_id = ?", a.AccountID).
		Where("journal_team_id = ?", a.JournalTeamID)

}

// GetDebitCredit implements DailyBalance.
func (a *AccountDailyBalance) GetDebitCredit() (debit float64, credit float64, balance float64) {
	return a.Debit, a.Credit, a.Balance
}

type ShopDailyBalance struct {
	ID            uint      `json:"id" gorm:"primarykey"`
	Day           time.Time `json:"day" gorm:"index:shop_daily_key_unique,unique"`
	ShopID        uint      `json:"shop_id" gorm:"index:shop_daily_key_unique,unique"`
	AccountID     uint      `json:"account_id" gorm:"index:shop_daily_key_unique,unique"`
	JournalTeamID uint      `json:"journal_team_id" gorm:"index:shop_daily_key_unique,unique"`
	Debit         float64   `json:"debit"`
	Credit        float64   `json:"credit"`
	Balance       float64   `json:"balance"`

	Account *Account `gorm:"-"`
}

// AddBalance implements DailyBalance.
func (s *ShopDailyBalance) AddBalance(balance float64) {
	s.Balance += balance
}

// Empty implements report.DailyBalance.
func (s *ShopDailyBalance) Empty() DailyBalance {
	return &ShopDailyBalance{}
}

// After implements report.DailyBalance.
func (s *ShopDailyBalance) After(tx *gorm.DB, lock bool) *gorm.DB {
	if lock {
		tx = tx.
			Clauses(
				clause.Locking{
					Strength: "UPDATE",
				},
			)
	}

	return tx.
		Model(&ShopDailyBalance{}).
		Where("day > ?", s.Day).
		Where("shop_id = ?", s.ShopID).
		Where("account_id = ?", s.AccountID).
		Where("journal_team_id = ?", s.JournalTeamID)
}

// Before implements report.DailyBalance.
func (s *ShopDailyBalance) Before(tx *gorm.DB, lock bool) *gorm.DB {
	if lock {
		tx = tx.
			Clauses(
				clause.Locking{
					Strength: "UPDATE",
				},
			)
	}

	return tx.
		Model(&ShopDailyBalance{}).
		Where("day < ?", s.Day).
		Where("shop_id = ?", s.ShopID).
		Where("account_id = ?", s.AccountID).
		Where("journal_team_id = ?", s.JournalTeamID)
}

// GetDebitCredit implements DailyBalance.
func (s *ShopDailyBalance) GetDebitCredit() (debit float64, credit float64, balance float64) {
	return s.Debit, s.Credit, s.Balance
}

type CsDailyBalance struct {
	ID            uint      `json:"id" gorm:"primarykey"`
	Day           time.Time `json:"day" gorm:"index:cs_daily_key_unique,unique"`
	CsID          uint      `json:"cs_id" gorm:"index:cs_daily_key_unique,unique"`
	AccountID     uint      `json:"account_id" gorm:"index:cs_daily_key_unique,unique"`
	JournalTeamID uint      `json:"journal_team_id" gorm:"index:cs_daily_key_unique,unique"`
	Debit         float64   `json:"debit"`
	Credit        float64   `json:"credit"`
	Balance       float64   `json:"balance"`

	Account *Account `gorm:"-"`
}

// AddBalance implements DailyBalance.
func (c *CsDailyBalance) AddBalance(balance float64) {
	c.Balance += balance
}

// Empty implements report.DailyBalance.
func (c *CsDailyBalance) Empty() DailyBalance {
	return &CsDailyBalance{}
}

// After implements report.DailyBalance.
func (c *CsDailyBalance) After(tx *gorm.DB, lock bool) *gorm.DB {
	if lock {
		tx = tx.
			Clauses(
				clause.Locking{
					Strength: "UPDATE",
				},
			)
	}

	return tx.
		Model(&CsDailyBalance{}).
		Where("day > ?", c.Day).
		Where("cs_id = ?", c.CsID).
		Where("account_id = ?", c.AccountID).
		Where("journal_team_id = ?", c.JournalTeamID)
}

// Before implements report.DailyBalance.
func (c *CsDailyBalance) Before(tx *gorm.DB, lock bool) *gorm.DB {
	if lock {
		tx = tx.
			Clauses(
				clause.Locking{
					Strength: "UPDATE",
				},
			)
	}

	return tx.
		Model(&CsDailyBalance{}).
		Where("day < ?", c.Day).
		Where("cs_id = ?", c.CsID).
		Where("account_id = ?", c.AccountID).
		Where("journal_team_id = ?", c.JournalTeamID)
}

// GetDebitCredit implements DailyBalance.
func (c *CsDailyBalance) GetDebitCredit() (debit float64, credit float64, balance float64) {
	return c.Debit, c.Credit, c.Balance
}

type SupplierDailyBalance struct {
	ID            uint      `json:"id" gorm:"primarykey"`
	Day           time.Time `json:"day" gorm:"index:sup_daily_key_unique,unique"`
	SupplierID    uint      `json:"supplier_id" gorm:"index:sup_daily_key_unique,unique"`
	AccountID     uint      `json:"account_id" gorm:"index:sup_daily_key_unique,unique"`
	JournalTeamID uint      `json:"journal_team_id" gorm:"index:sup_daily_key_unique,unique"`
	Debit         float64   `json:"debit"`
	Credit        float64   `json:"credit"`
	Balance       float64   `json:"balance"`

	Account *Account `gorm:"-"`
}

// AddBalance implements DailyBalance.
func (s *SupplierDailyBalance) AddBalance(balance float64) {
	s.Balance += balance
}

// Empty implements report.DailyBalance.
func (s *SupplierDailyBalance) Empty() DailyBalance {
	return &SupplierDailyBalance{}
}

// After implements report.DailyBalance.
func (s *SupplierDailyBalance) After(tx *gorm.DB, lock bool) *gorm.DB {
	if lock {
		tx = tx.
			Clauses(
				clause.Locking{
					Strength: "UPDATE",
				},
			)
	}

	return tx.
		Model(&SupplierDailyBalance{}).
		Where("day > ?", s.Day).
		Where("supplier_id = ?", s.SupplierID).
		Where("account_id = ?", s.AccountID).
		Where("journal_team_id = ?", s.JournalTeamID)
}

// Before implements report.DailyBalance.
func (s *SupplierDailyBalance) Before(tx *gorm.DB, lock bool) *gorm.DB {
	if lock {
		tx = tx.
			Clauses(
				clause.Locking{
					Strength: "UPDATE",
				},
			)
	}

	return tx.
		Model(&SupplierDailyBalance{}).
		Where("day < ?", s.Day).
		Where("supplier_id = ?", s.SupplierID).
		Where("account_id = ?", s.AccountID).
		Where("journal_team_id = ?", s.JournalTeamID)
}

// GetDebitCredit implements DailyBalance.
func (s *SupplierDailyBalance) GetDebitCredit() (debit float64, credit float64, balance float64) {
	return s.Debit, s.Credit, s.Balance
}

type CustomLabelDailyBalance struct {
	ID            uint      `json:"id" gorm:"primarykey"`
	Day           time.Time `json:"day" gorm:"index:custom_daily_key_unique,unique"`
	CustomID      uint      `json:"custom_id" gorm:"index:custom_daily_key_unique,unique"`
	AccountID     uint      `json:"account_id" gorm:"index:custom_daily_key_unique,unique"`
	JournalTeamID uint      `json:"journal_team_id" gorm:"index:custom_daily_key_unique,unique"`
	Debit         float64   `json:"debit"`
	Credit        float64   `json:"credit"`
	Balance       float64   `json:"balance"`

	Account *Account `gorm:"-"`
}

// AddBalance implements DailyBalance.
func (c *CustomLabelDailyBalance) AddBalance(balance float64) {
	c.Balance += balance
}

// Empty implements report.DailyBalance.
func (c *CustomLabelDailyBalance) Empty() DailyBalance {
	return &CustomLabelDailyBalance{}
}

// After implements report.DailyBalance.
func (c *CustomLabelDailyBalance) After(tx *gorm.DB, lock bool) *gorm.DB {
	if lock {
		tx = tx.
			Clauses(
				clause.Locking{
					Strength: "UPDATE",
				},
			)
	}

	return tx.
		Model(&CustomLabelDailyBalance{}).
		Where("day > ?", c.Day).
		Where("custom_id = ?", c.CustomID).
		Where("account_id = ?", c.AccountID).
		Where("journal_team_id = ?", c.JournalTeamID)
}

// Before implements report.DailyBalance.
func (c *CustomLabelDailyBalance) Before(tx *gorm.DB, lock bool) *gorm.DB {
	if lock {
		tx = tx.
			Clauses(
				clause.Locking{
					Strength: "UPDATE",
				},
			)
	}

	return tx.
		Model(&CustomLabelDailyBalance{}).
		Where("day < ?", c.Day).
		Where("custom_id = ?", c.CustomID).
		Where("account_id = ?", c.AccountID).
		Where("journal_team_id = ?", c.JournalTeamID)
}

// GetDebitCredit implements DailyBalance.
func (c *CustomLabelDailyBalance) GetDebitCredit() (debit float64, credit float64, balance float64) {
	return c.Debit, c.Credit, c.Balance
}
