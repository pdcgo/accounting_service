package accounting_core

import (
	"errors"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type CreateTransaction interface {
	Labels(labels []*Label) CreateTransaction
	Create(tran *Transaction) CreateTransaction
	Err() error
}

type createTansactionImpl struct {
	tx   *gorm.DB
	err  error
	tran *Transaction
}

// Create implements CreateTransaction.
func (c *createTansactionImpl) Create(tran *Transaction) CreateTransaction {
	tran.Created = time.Now()
	err := c.tx.Save(tran).Error
	if err != nil {
		return c.setErr(err)
	}

	c.tran = tran

	return c
}

// Err implements CreateTransaction.
func (c *createTansactionImpl) Err() error {
	return c.err
}

// Labels implements CreateTransaction.
func (c *createTansactionImpl) Labels(labels []*Label) CreateTransaction {
	if c.tran == nil {
		return c.setErr(errors.New("transaction nil"))
	}

	if c.tran.ID == 0 {
		return c.setErr(errors.New("transaction id is null"))
	}

	var err error
	for _, label := range labels {
		keyID := label.Hash()
		err = c.tx.Model(&Label{}).Where("id = ?", keyID).First(label).Error
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				err = c.tx.Save(label).Error
				if err != nil {
					return c.setErr(err)
				}
			} else {
				return c.setErr(err)
			}

		}

		rel := TransactionLabel{
			TransactionID: c.tran.ID,
			LabelID:       label.ID,
		}

		err = c.tx.Save(&rel).Error
		if err != nil {
			return c.setErr(err)
		}
	}

	return c
}
func (c *createTansactionImpl) setErr(err error) *createTansactionImpl {
	if c.err != nil {
		return c
	}

	if err != nil {
		c.err = err
	}

	return c
}

func NewTransaction(tx *gorm.DB) CreateTransaction {
	return &createTansactionImpl{
		tx: tx,
	}
}

var ErrTransactionNotLoaded = errors.New("transaction not loaded")

type TransactionMutation interface {
	ByRefID(refid RefID, lock bool) TransactionMutation
	CheckEntry() TransactionMutation
	RollbackEntry(userID uint, desc string) TransactionMutation
	IsExist() bool
	Data() *Transaction
	Err() error
}

type transactionMutationImpl struct {
	tx   *gorm.DB
	data *Transaction
	err  error
}

// CheckEntry implements TransactionMutation.
func (t *transactionMutationImpl) CheckEntry() TransactionMutation {
	var err error
	var entries JournalEntriesList

	if t.data == nil {
		return t.setErr(errors.New("data transaction data not loaded"))
	}

	err = t.
		tx.
		Model(&JournalEntry{}).
		Preload("Account").
		Where("transaction_id = ?", t.data.ID).
		Find(&entries).
		Error

	if err != nil {
		return t.setErr(err)
	}

	mapBalances, err := entries.AccountBalance()
	if err != nil {
		return t.setErr(err)
	}

	var debit, credit float64
	for _, balance := range mapBalances {
		debit += balance.Debit
		credit += balance.Credit
	}

	if debit != credit {
		return t.setErr(&ErrEntryInvalid{
			Debit:  debit,
			Credit: credit,
			List:   entries,
		})
	}

	return t
}

// Data implements TransactionMutation.
func (t *transactionMutationImpl) Data() *Transaction {
	return t.data
}

// Exist implements TransactionMutation.
func (t *transactionMutationImpl) IsExist() bool {
	return t.data.ID != 0
}

// ByRefID implements TransactionMutation.
func (t *transactionMutationImpl) ByRefID(refid RefID, lock bool) TransactionMutation {
	var err error
	tx := t.tx

	t.data = &Transaction{}

	if lock {
		tx = tx.Clauses(clause.Locking{
			Strength: "UPDATE",
		})
	}
	err = tx.Model(&Transaction{}).
		Where("ref_id = ?", refid).
		Find(t.data).
		Error

	if err != nil {
		return t.setErr(err)
	}
	return t
}

// Err implements TransactionMutation.
func (t *transactionMutationImpl) Err() error {
	return t.err
}

// RollbackEntry implements TransactionMutation.
func (t *transactionMutationImpl) RollbackEntry(userID uint, desc string) TransactionMutation {
	var err error
	entries := []*JournalEntry{}

	if t.data == nil {
		return t.setErr(ErrTransactionNotLoaded)
	}

	err = t.
		tx.
		Model(&JournalEntry{}).
		Where("transaction_id = ?", t.data.ID).
		Find(&entries).
		Error

	if err != nil {
		return t.setErr(err)
	}

	if len(entries) == 0 {
		return t.setErr(errors.New("entries on transaction is empty"))
	}

	teamEntries := map[uint]CreateEntry{}

	for _, entry := range entries {
		if teamEntries[entry.TeamID] == nil {
			teamEntries[entry.TeamID] = NewCreateEntry(t.tx, entry.TeamID, userID)
		}

		teamEntries[entry.TeamID].Set(entry.AccountID, entry.Debit, entry.Credit)
	}

	for _, nentries := range teamEntries {
		err = nentries.
			TransactionID(t.data.ID).
			Desc(desc).
			Commit().
			Err()

		if err != nil {
			return t.setErr(err)
		}
	}

	return t
}

func (t *transactionMutationImpl) setErr(err error) *transactionMutationImpl {
	if t.err != nil {
		return t
	}

	if err != nil {
		t.err = err
	}

	return t
}

func NewTransactionMutation(tx *gorm.DB) TransactionMutation {
	return &transactionMutationImpl{
		tx: tx,
	}
}
