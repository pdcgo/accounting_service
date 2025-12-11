package accounting_core

import (
	"context"
	"errors"
	"fmt"

	"github.com/pdcgo/schema/services/report_iface/v1"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/gorm"
)

type BookManage interface {
	NewCreateEntry(teamID uint, createdByID uint) CreateEntry
	NewTransaction() CreateTransaction
	LabelExtra() *TxLabelExtra
	Entries() JournalEntriesList
	DailyUpdateData() (*report_iface.DailyUpdateBalanceRequest, error)
}

// var _ BookManage = (*bookManageImpl)(nil)

type bookManageImpl struct {
	tx      *gorm.DB
	labels  *TxLabelExtra
	entries JournalEntriesList
}

// DailyUpdateData implements BookManage.
func (h *bookManageImpl) DailyUpdateData() (*report_iface.DailyUpdateBalanceRequest, error) {
	label := &report_iface.TxLabelExtra{}

	blabel := h.LabelExtra()
	if blabel != nil {
		tagIDs := []uint64{}

		for _, tagID := range blabel.TagIDs {
			tagIDs = append(tagIDs, uint64(tagID))
		}

		label = &report_iface.TxLabelExtra{
			ShopId:     uint64(blabel.ShopID),
			CsId:       uint64(blabel.CsID),
			TagIds:     tagIDs,
			SupplierId: uint64(blabel.SupplierID),
		}
	}

	entries := []*report_iface.EntryPayload{}
	for _, entry := range h.Entries() {
		entries = append(entries, &report_iface.EntryPayload{
			Id:            uint64(entry.ID),
			TransactionId: uint64(entry.TransactionID),
			Desc:          entry.Desc,
			AccountId:     uint64(entry.AccountID),
			TeamId:        uint64(entry.TeamID),
			Debit:         entry.Debit,
			Credit:        entry.Credit,
			EntryTime:     timestamppb.New(entry.EntryTime),
			Rollback:      entry.Rollback,
		})
	}

	msg := report_iface.DailyUpdateBalanceRequest{
		Entries:    entries,
		LabelExtra: label,
	}

	return &msg, nil
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

var ErrSkipTransaction = errors.New("skip transaction")

func OpenTransaction(ctx context.Context, tx *gorm.DB, handle func(tx *gorm.DB, bookmng BookManage) error) error {
	var err error

	var updata *report_iface.DailyUpdateBalanceRequest

	err = tx.Transaction(func(tx *gorm.DB) error {
		hdlr := bookManageImpl{
			tx: tx,
		}

		err = handle(tx, &hdlr)
		if err != nil {
			return err
		}

		if len(hdlr.entries) == 0 {
			return errors.New("entries empty in ending transaction")
		}

		for _, entry := range hdlr.entries {
			if entry.ID == 0 {
				// hdlr.entries.PrintJournalEntries(tx)
				return fmt.Errorf("theres entry not save desc %s", entry.Desc)
			}
		}

		updata, err = hdlr.DailyUpdateData()
		if err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		if errors.Is(err, ErrSkipTransaction) {
			return nil
		}

		return err
	}

	for _, handler := range customHandler {
		err = handler(ctx, updata)
		if err != nil {
			return err
		}
	}

	return nil
}
