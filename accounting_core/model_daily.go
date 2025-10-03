package accounting_core

import "time"

func ParseDate(t time.Time) time.Time {
	y, m, d := t.Date()
	day := time.Time{}
	day = day.AddDate(y-1, int(m)-1, d-1)

	return day
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

// GetDebitCredit implements DailyBalance.
func (s *ShopDailyBalance) GetDebitCredit() (debit float64, credit float64, balance float64) {
	panic("unimplemented")
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

// GetDebitCredit implements DailyBalance.
func (c *CsDailyBalance) GetDebitCredit() (debit float64, credit float64, balance float64) {
	panic("unimplemented")
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

// GetDebitCredit implements DailyBalance.
func (s *SupplierDailyBalance) GetDebitCredit() (debit float64, credit float64, balance float64) {
	panic("unimplemented")
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
