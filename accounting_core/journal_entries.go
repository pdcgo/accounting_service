package accounting_core

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"time"

	"gorm.io/gorm"
)

var precision = 5

func RoundUp(x float64, n int) float64 {
	pow := math.Pow(10, float64(n))
	return math.Ceil(x*pow) / pow
}

var ErrEmptyEntry = errors.New("entry empty")

type ErrEntryInvalid struct {
	Debit     float64            `json:"debit"`
	Credit    float64            `json:"credit"`
	List      JournalEntriesList `json:"list"`
	Precision int                `json:"precision"`
}

// Error implements error.
func (e *ErrEntryInvalid) Error() string {
	raw, _ := json.Marshal(e)
	return "journal entry invalid" + string(raw)
}

type EntryAccountPayload struct {
	Key    AccountKey
	TeamID uint
}

type EntryOption func(entry *JournalEntry) error

func EntryDescOption(desc string) EntryOption {
	return func(entry *JournalEntry) error {
		entry.Desc = desc
		return nil
	}
}

type CreateEntry interface {
	Commit() CreateEntry
	Rollback(oldentries map[uint]*ChangeBalance, opts ...EntryOption) CreateEntry
	Desc(desc string) CreateEntry
	TransactionID(txID uint) CreateEntry
	Transaction(tx *Transaction) CreateEntry
	EntryTime(t time.Time) CreateEntry
	Set(accID uint, credit, debit float64) CreateEntry
	From(account *EntryAccountPayload, amount float64, opts ...EntryOption) CreateEntry
	To(account *EntryAccountPayload, amount float64, opts ...EntryOption) CreateEntry
	Err() error
}

type createEntryImpl struct {
	tx          *gorm.DB
	teamID      uint
	createdByID uint
	entries     map[uint]*JournalEntry
	accountMap  map[uint]*Account
	err         error
}

// Rollback implements CreateEntry.
func (c *createEntryImpl) Rollback(oldentries map[uint]*ChangeBalance, opts ...EntryOption) CreateEntry {
	for _, ch := range oldentries {
		amount := ch.Change()
		if amount > 0 {
			c.From(&EntryAccountPayload{
				Key:    ch.Account.AccountKey,
				TeamID: ch.Account.TeamID,
			}, amount, opts...)
		}

		if amount < 0 {
			c.To(&EntryAccountPayload{
				Key:    ch.Account.AccountKey,
				TeamID: ch.Account.TeamID,
			}, math.Abs(amount), opts...)
		}
	}

	return c
}

// Set implements CreateEntry.
func (c *createEntryImpl) Set(accID uint, credit float64, debit float64) CreateEntry {
	var err error
	if c.accountMap[accID] == nil {
		c.accountMap[accID] = &Account{}
		err = c.tx.Model(&Account{}).First(c.accountMap[accID], accID).Error
		if err != nil {
			return c.setErr(err)
		}
	}

	c.mergeEntry(accID, &JournalEntry{
		AccountID: accID,
		Credit:    credit,
		Debit:     debit,
	})
	return c
}

// EntryTime implements CreateEntry.
func (c *createEntryImpl) EntryTime(t time.Time) CreateEntry {
	if c.isEntryEmpty() {
		return c.setErr(ErrEmptyEntry)
	}

	for _, entry := range c.entries {
		entry.EntryTime = t
	}
	return c
}

// Transaction implements CreateEntry.
func (c *createEntryImpl) Transaction(tx *Transaction) CreateEntry {
	if c.isEntryEmpty() {
		return c.setErr(ErrEmptyEntry)
	}

	for _, entry := range c.entries {
		if entry.Desc == "" {
			entry.Desc = tx.Desc
		}

		entry.TransactionID = tx.ID
	}
	return c
}

// From implements CreateEntry.
func (c *createEntryImpl) From(account *EntryAccountPayload, amount float64, opts ...EntryOption) CreateEntry {
	return c.To(account, amount*-1, opts...)
}

// Commit implements CreateEntry.
func (c *createEntryImpl) Commit() CreateEntry {
	if c.isEntryEmpty() {
		return c.setErr(ErrEmptyEntry)
	}
	var entries JournalEntriesList

	var debit, credit float64

	for _, entry := range c.entries {
		if entry.EntryTime.IsZero() {
			entry.EntryTime = time.Now()
		}

		entry.TeamID = c.teamID
		entry.CreatedByID = c.createdByID

		debit += entry.Debit
		credit += entry.Credit

		if entry.Debit == entry.Credit {
			continue
			// return c.setErr(&ErrEntryInvalid{
			// 	Debit:  debit,
			// 	Credit: credit,
			// 	List:   entries,
			// })
		}

		if entry.Debit > 0 && entry.Credit > 0 {
			dentry := *entry
			dentry.Credit = 0
			entries = append(entries, &dentry)
			entry.Debit = 0
			entries = append(entries, entry)
			continue
		}

		entries = append(entries, entry)
	}

	// checking debit and credit balance
	if RoundUp(debit, precision) != RoundUp(credit, precision) {
		// entries.PrintJournalEntries(c.tx)
		return c.setErr(&ErrEntryInvalid{
			Debit:     debit,
			Credit:    credit,
			List:      entries,
			Precision: precision,
		})
	}

	err := c.tx.Save(&entries).Error
	if err != nil {
		return c.setErr(err)
	}

	return c.
		updateBalance(entries)
}

func (c *createEntryImpl) updateBalance(entries JournalEntriesList) *createEntryImpl {
	var err error

	for _, entry := range entries {
		y, m, d := entry.EntryTime.Date()
		day := time.Time{}
		day = day.AddDate(y-1, int(m)-1, d-1)

		var balance float64
		account, ok := c.accountMap[entry.AccountID]
		if !ok {
			err = fmt.Errorf("update balance error acount not found %d", entry.AccountID)
			return c.setErr(err)
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
			Credit:        entry.Debit,
			Balance:       balance,
		}
		row := c.
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
				return c.setErr(err)
			}

			err = c.
				tx.
				Save(dayBalance).
				Error

			if err != nil {
				return c.setErr(err)
			}
		}

	}

	return c
}

// Desc implements CreateEntry.
func (c *createEntryImpl) Desc(desc string) CreateEntry {
	if c.isEntryEmpty() {
		return c.setErr(ErrEmptyEntry)
	}

	for _, entry := range c.entries {
		if entry.Desc == "" {
			entry.Desc = desc
		}

	}
	return c
}

// Err implements CreateEntry.
func (c *createEntryImpl) Err() error {
	return c.err
}

// To implements CreateEntry.
func (c *createEntryImpl) To(account *EntryAccountPayload, amount float64, opts ...EntryOption) CreateEntry {
	acc, err := c.getAccount(account)
	if err != nil {
		return c.setErr(err)
	}

	entry := &JournalEntry{
		AccountID: acc.ID,
	}

	err = acc.SetAmountEntry(amount, entry)
	if err != nil {
		return c.setErr(err)
	}

	for _, opt := range opts {
		err = opt(entry)
		if err != nil {
			return c.setErr(err)
		}
	}

	c.mergeEntry(entry.AccountID, entry)

	return c
}

// TransactionID implements CreateEntry.
func (c *createEntryImpl) TransactionID(txID uint) CreateEntry {
	if c.isEntryEmpty() {
		return c.setErr(ErrEmptyEntry)
	}

	for _, entry := range c.entries {
		entry.TransactionID = txID
	}
	return c
}

func (c *createEntryImpl) getAccount(accp *EntryAccountPayload) (*Account, error) {
	var acc Account
	// var err error

	err := c.tx.Model(&Account{}).
		Where("account_key = ?", accp.Key).
		Where("team_id = ?", accp.TeamID).
		Find(&acc).
		Error

	if err != nil {
		return &acc, err
	}

	if acc.ID == 0 {
		return &acc, fmt.Errorf("account not found %s in team %d", accp.Key, accp.TeamID)
	}

	c.accountMap[acc.ID] = &acc
	return &acc, nil
}

func (c *createEntryImpl) mergeEntry(accID uint, entry *JournalEntry) {
	if c.entries[accID] != nil {
		c.entries[accID].Credit += entry.Credit
		c.entries[accID].Debit += entry.Debit
	} else {
		c.entries[accID] = entry
	}
}

func (c *createEntryImpl) isEntryEmpty() bool {
	return len(c.entries) == 0
}

func (c *createEntryImpl) setErr(err error) *createEntryImpl {
	if c.err != nil {
		return c
	}

	if err != nil {
		c.err = err
	}

	return c
}

func NewCreateEntry(tx *gorm.DB, teamID uint, createdByID uint) CreateEntry {
	return &createEntryImpl{
		tx:          tx,
		teamID:      teamID,
		createdByID: createdByID,
		entries:     map[uint]*JournalEntry{},
		accountMap:  map[uint]*Account{},
	}
}
