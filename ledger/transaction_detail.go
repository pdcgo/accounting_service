package ledger

import (
	"context"

	"connectrpc.com/connect"
	"github.com/pdcgo/accounting_service/accounting_core"
	"github.com/pdcgo/schema/services/accounting_iface/v1"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type EntryItemList []*accounting_core.JournalEntry

func (lst EntryItemList) toGroupProto() []*accounting_iface.BookEntryGroupItem {
	mapresult := map[uint64]*accounting_iface.BookEntryGroupItem{}
	result := []*accounting_iface.BookEntryGroupItem{}
	for _, item := range lst {
		acc := item.Account

		if mapresult[uint64(item.TeamID)] == nil {
			mapresult[uint64(item.TeamID)] = &accounting_iface.BookEntryGroupItem{
				TeamId:  uint64(item.TeamID),
				Entries: []*accounting_iface.EntryItem{},
			}
		}

		mapresult[uint64(item.TeamID)].Entries = append(mapresult[uint64(item.TeamID)].Entries, &accounting_iface.EntryItem{
			Id:        uint64(item.ID),
			TeamId:    uint64(item.TeamID),
			AccountId: uint64(item.AccountID),
			EntryTime: item.EntryTime.UnixMicro(),
			Desc:      item.Desc,
			Debit:     item.Debit,
			Credit:    item.Credit,
			Account: &accounting_iface.EntryAccount{
				Id:         uint64(acc.ID),
				TeamId:     uint64(acc.TeamID),
				AccountKey: string(acc.AccountKey),
				Name:       acc.Name,
			},
		})
	}

	for _, d := range mapresult {
		v := d
		result = append(result, v)
	}

	return result
}

// TransactionDetail implements accounting_ifaceconnect.LedgerServiceHandler.
func (l *ledgerServiceImpl) TransactionDetail(
	ctx context.Context,
	req *connect.Request[accounting_iface.TransactionDetailRequest],
) (*connect.Response[accounting_iface.TransactionDetailResponse], error) {
	var err error
	db := l.db.WithContext(ctx)

	var tran accounting_core.Transaction
	err = db.
		Model(&accounting_core.Transaction{}).
		First(&tran, req.Msg.Id).
		Error

	if err != nil {
		return connect.NewResponse(&accounting_iface.TransactionDetailResponse{}), err
	}

	list := EntryItemList{}

	err = db.
		Model(&accounting_core.JournalEntry{}).
		Where("transaction_id = ?", tran.ID).
		Preload("Account").
		Find(&list).
		Error

	if err != nil {
		return connect.NewResponse(&accounting_iface.TransactionDetailResponse{}), err
	}

	result := accounting_iface.TransactionDetailResponse{
		Transaction: &accounting_iface.Transaction{
			Id:          uint64(tran.ID),
			RefId:       string(tran.RefID),
			TeamId:      uint64(tran.TeamID),
			CreatedById: uint64(tran.CreatedByID),
			Desc:        tran.Desc,
			Created:     timestamppb.New(tran.Created),
		},
		Books: list.toGroupProto(),
	}

	return connect.NewResponse(&result), nil

}
