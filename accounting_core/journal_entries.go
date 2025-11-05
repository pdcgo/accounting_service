package accounting_core

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"time"

	"gorm.io/gorm"
)

var Precision = 5

func RoundUp(x float64, n int) float64 {
	pow := math.Pow(10, float64(n))
	result := math.Floor(x*pow) / pow
	return result
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

type CommitOption func(entry *JournalEntry) error

func RollbackOption() CommitOption {
	return func(entry *JournalEntry) error {
		entry.Rollback = true
		return nil
	}
}

func CustomTimeOption(t time.Time) CommitOption {
	return func(entry *JournalEntry) error {
		entry.EntryTime = t
		return nil
	}
}

type CreateEntry interface {
	Commit(opts ...CommitOption) CreateEntry
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
	afterCommit func(c *createEntryImpl) error
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
func (c *createEntryImpl) Commit(opts ...CommitOption) CreateEntry {
	if c.isEntryEmpty() {
		return c.setErr(ErrEmptyEntry)
	}
	var entries JournalEntriesList

	var debit, credit float64

	for _, entry := range c.entries {
		if entry.EntryTime.IsZero() {
			entry.EntryTime = time.Now()
		}

		// options commit
		for _, opt := range opts {
			err := opt(entry)
			if err != nil {
				return c.setErr(err)
			}
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
	if RoundUp(debit, Precision) != RoundUp(credit, Precision) {
		// log.Println(RoundUp(debit, precision), RoundUp(credit, precision))
		// entries.PrintJournalEntries(c.tx)
		return c.setErr(&ErrEntryInvalid{
			Debit:     debit,
			Credit:    credit,
			List:      entries,
			Precision: Precision,
		})
	}

	err := c.tx.Save(&entries).Error
	if err != nil {
		return c.setErr(err)
	}

	if c.afterCommit != nil {
		err = c.afterCommit(c)
		if err != nil {
			return c.setErr(err)
		}
	}

	return c
}

// func (c *createEntryImpl) updateBalance(entries JournalEntriesList) *createEntryImpl {
// 	var err error

// 	labels := &report_iface.TxLabelExtra{}
// 	if c.labelExtra != nil {
// 		tagIDs := []uint64{}
// 		for _, tagID := range c.labelExtra.TagIDs {
// 			tagIDs = append(tagIDs, uint64(tagID))
// 		}
// 		labels = &report_iface.TxLabelExtra{
// 			CsId:       uint64(c.labelExtra.CsID),
// 			ShopId:     uint64(c.labelExtra.ShopID),
// 			SupplierId: uint64(c.labelExtra.SupplierID),
// 			TagIds:     tagIDs,
// 		}
// 	}

// 	entryPayload := []*report_iface.EntryPayload{}
// 	for _, entry := range entries {
// 		entryPayload = append(entryPayload, &report_iface.EntryPayload{
// 			Id:            uint64(entry.ID),
// 			TransactionId: uint64(entry.TransactionID),
// 			Desc:          entry.Desc,
// 			AccountId:     uint64(entry.AccountID),
// 			Debit:         entry.Debit,
// 			Credit:        entry.Credit,
// 			TeamId:        uint64(entry.TeamID),
// 			EntryTime:     timestamppb.New(entry.EntryTime),
// 		})
// 	}

// 	err = onEntryCreate(&report_iface.DailyUpdateBalanceRequest{
// 		LabelExtra: labels,
// 		Entries:    entryPayload,
// 	})

// 	if err != nil {
// 		return c.setErr(err)
// 	}

// 	return c
// }

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

// func NewCreateEntry(tx *gorm.DB, teamID uint, createdByID uint) CreateEntry {
// 	return &createEntryImpl{
// 		tx:          tx,
// 		teamID:      teamID,
// 		createdByID: createdByID,
// 		entries:     map[uint]*JournalEntry{},
// 		accountMap:  map[uint]*Account{},
// 	}
// }
