package accounting_core

import (
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/pdcgo/shared/db_models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type CreateTransaction interface {
	Labels(labels []*Label) CreateTransaction
	Create(tran *Transaction) CreateTransaction
	AddSupplierID(suplierID uint) CreateTransaction
	AddShopID(shopID uint) CreateTransaction
	AddCustomerServiceID(customerServiceID uint) CreateTransaction
	AddTags(tnames []string) CreateTransaction

	Err() error
}

var ErrTransactionNotCreated = errors.New("transaction not created")

type TxLabelExtra struct {
	ShopID     uint
	CsID       uint
	SupplierID uint
	TagIDs     []uint
}

type createTansactionImpl struct {
	tx                     *gorm.DB
	err                    error
	tran                   *Transaction
	labelExtra             TxLabelExtra
	afterTransactionCreate func(labels *TxLabelExtra) error
}

// AddCustomerServiceID implements CreateTransaction.
func (c *createTansactionImpl) AddCustomerServiceID(customerServiceID uint) CreateTransaction {
	var err error
	if c.isTransactionEmpty() {
		return c.setErr(ErrTransactionNotCreated)
	}

	rel := TransactionCustomerService{
		TransactionID:     c.tran.ID,
		CustomerServiceID: customerServiceID,
	}

	err = c.tx.Save(&rel).Error
	if err != nil {
		return c.setErr(err)
	}

	c.labelExtra.CsID = customerServiceID
	return c
}

// AddShopID implements CreateTransaction.
func (c *createTansactionImpl) AddShopID(shopID uint) CreateTransaction {
	var err error
	if c.isTransactionEmpty() {
		return c.setErr(ErrTransactionNotCreated)
	}

	if shopID == 0 {
		return c.setErr(errors.New("shop id is null"))
	}

	var shop db_models.Marketplace
	err = c.
		tx.
		Model(&db_models.Marketplace{}).
		First(&shop, shopID).
		Error

	if err != nil {
		return c.setErr(err)
	}

	rel := TransactionShop{
		TransactionID: c.tran.ID,
		ShopID:        shopID,
	}

	err = c.tx.Save(&rel).Error
	if err != nil {
		return c.setErr(err)
	}

	c.labelExtra.ShopID = shopID
	return c.
		AddTags([]string{string(shop.MpType)})

}

// AddSupplierID implements CreateTransaction.
func (c *createTansactionImpl) AddSupplierID(suplierID uint) CreateTransaction {
	var err error
	if c.isTransactionEmpty() {
		return c.setErr(ErrTransactionNotCreated)
	}

	rel := TransactionSupplier{
		TransactionID: c.tran.ID,
		SupplierID:    suplierID,
	}

	err = c.tx.Save(&rel).Error
	if err != nil {
		return c.setErr(err)
	}

	c.labelExtra.SupplierID = suplierID
	return c
}

// AddTags implements CreateTransaction.
func (c *createTansactionImpl) AddTags(tnames []string) CreateTransaction {
	var err error
	if c.isTransactionEmpty() {
		return c.setErr(ErrTransactionNotCreated)
	}

	if len(tnames) == 0 {
		return c
	}

	for i, tname := range tnames {
		tname = SanityTag(tname)
		tnames[i] = tname
	}

	tmap := map[string]bool{}
	for _, tname := range tnames {
		tmap[tname] = false
	}

	tags := []*AccountingTag{}

	err = c.
		tx.
		Transaction(func(tx *gorm.DB) error {

			err = tx.
				Model(&AccountingTag{}).
				Where("name IN ?", tnames).
				Find(&tags).
				Error

			if err != nil {
				return err
			}

			for _, tag := range tags {
				tmap[tag.Name] = true
			}

			for _, tname := range tnames {
				if !tmap[tname] {
					tag := AccountingTag{
						Name: SanityTag(tname),
					}
					err = tx.Save(&tag).Error
					if err != nil {
						return err
					}

					tags = append(tags, &tag)
				}
			}

			for _, tag := range tags {
				rel := TransactionTag{
					TransactionID: c.tran.ID,
					TagID:         tag.ID,
				}
				err = tx.Save(&rel).Error
				if err != nil {
					return err
				}
			}

			return nil
		})

	if err != nil {
		return c.setErr(err)
	}
	for _, tag := range tags {
		c.labelExtra.TagIDs = append(c.labelExtra.TagIDs, tag.ID)
	}
	return c
}

func (c *createTansactionImpl) isTransactionEmpty() bool {
	if c.tran == nil {
		return true
	}

	if c.tran.ID == 0 {
		return true
	}

	return false

}

// Create implements CreateTransaction.
func (c *createTansactionImpl) Create(tran *Transaction) CreateTransaction {
	if tran.Created.IsZero() {
		tran.Created = time.Now()
	}

	err := c.tx.Save(tran).Error
	if err != nil {
		return c.setErr(err)
	}

	if c.afterTransactionCreate != nil {
		err = c.afterTransactionCreate(&c.labelExtra)
		if err != nil {
			return c.setErr(err)
		}
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

// func NewTransaction(tx *gorm.DB) CreateTransaction {
// 	return &createTansactionImpl{
// 		tx: tx,
// 		labelExtra: TxLabelExtra{
// 			TagIDs: []uint{},
// 		},
// 	}
// }

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

	if t.data.ID == 0 {
		return t.setErr(fmt.Errorf("transaction %s not found", refid))
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
	entries := JournalEntriesList{}

	if t.data == nil {
		return t.setErr(ErrTransactionNotLoaded)
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

	if len(entries) == 0 {
		return t.setErr(errors.New("entries on transaction is empty"))
	}

	// accbalance, _ := entries.AccountBalance()
	// debugtool.LogJson(accbalance)

	err = OpenTransaction(t.tx, func(tx *gorm.DB, bookmng BookManage) error {

		teamEntries := map[uint]JournalEntriesList{}
		teamBookEntry := map[uint]CreateEntry{}
		for _, dd := range entries {
			entry := dd
			if teamEntries[entry.TeamID] == nil {
				teamEntries[entry.TeamID] = JournalEntriesList{}
				teamBookEntry[entry.TeamID] = bookmng.NewCreateEntry(entry.TeamID, userID)
			}

			teamEntries[entry.TeamID] = append(teamEntries[entry.TeamID], entry)
		}

		for teamID, tentries := range teamEntries {
			var accBalance map[uint]*ChangeBalance
			accBalance, err = tentries.AccountBalance()
			if err != nil {
				return err
			}

			for accID, accentries := range accBalance {
				diff := accentries.Change()
				// fmt.Printf("%3.f -- %s\n", diff, accentries.Account.Name)

				switch accentries.Account.BalanceType {
				case DebitBalance:

					if diff > 0 {
						teamBookEntry[teamID].Set(accID, diff, 0)
					}
					if diff < 0 {
						teamBookEntry[teamID].Set(accID, 0, math.Abs(diff))
					}
				case CreditBalance:
					if diff > 0 {
						teamBookEntry[teamID].Set(accID, 0, diff)
					}
					if diff < 0 {
						teamBookEntry[teamID].Set(accID, math.Abs(diff), 0)
					}
				}
			}

		}

		for _, nentries := range teamBookEntry {
			err = nentries.
				TransactionID(t.data.ID).
				Desc(desc).
				Commit(RollbackOption()).
				Err()

			if err != nil {
				return err
			}
		}

		return nil
	})

	if err != nil {
		return t.setErr(err)
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
