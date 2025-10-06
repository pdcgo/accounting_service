package accounting_core

import "gorm.io/gorm"

type BookManage interface {
	NewCreateEntry(teamID uint, createdByID uint) CreateEntry
	NewTransaction() CreateTransaction
	LabelExtra() *TxLabelExtra
	Entries() JournalEntriesList
}

type bookManageImpl struct {
	tx      *gorm.DB
	labels  *TxLabelExtra
	entries JournalEntriesList
}

// Entries implements BookManage.
func (h *bookManageImpl) Entries() JournalEntriesList {
	return h.entries
}

// LabelExtra implements BookManage.
func (h *bookManageImpl) LabelExtra() *TxLabelExtra {
	return h.labels
}

// CreateTransaction implements BookManage.
func (h *bookManageImpl) NewTransaction() CreateTransaction {
	return &createTansactionImpl{
		tx: h.tx,
		labelExtra: TxLabelExtra{
			TagIDs: []uint{},
		},
		afterTransactionCreate: h.afterTransactionCreate,
	}
}

// NewCreateEntry implements BookManage.
func (h *bookManageImpl) NewCreateEntry(teamID uint, createdByID uint) CreateEntry {
	return &createEntryImpl{
		tx:          h.tx,
		teamID:      teamID,
		createdByID: createdByID,
		entries:     map[uint]*JournalEntry{},
		accountMap:  map[uint]*Account{},
		afterCommit: h.afterCommit,
	}
}

func (h *bookManageImpl) afterTransactionCreate(labels *TxLabelExtra) error {
	h.labels = labels
	return nil
}
func (h *bookManageImpl) afterCommit(c *createEntryImpl) error {
	for _, entry := range c.entries {
		h.entries = append(h.entries, entry)
	}

	return nil
}

func OpenTransaction(tx *gorm.DB, handle func(tx *gorm.DB, bookmng BookManage) error) error {
	var err error
	err = tx.Transaction(func(tx *gorm.DB) error {
		hdlr := bookManageImpl{
			tx: tx,
		}

		err = handle(tx, &hdlr)
		if err != nil {
			return err
		}

		for _, handler := range customHandler {
			err = handler(&hdlr)
			if err != nil {
				return err
			}
		}

		return nil
	})

	if err != nil {
		return err
	}
	return nil
}
