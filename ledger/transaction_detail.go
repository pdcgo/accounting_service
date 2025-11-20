package ledger

import (
	"context"

	"connectrpc.com/connect"
	"github.com/pdcgo/accounting_service/accounting_core"
	"github.com/pdcgo/schema/services/accounting_iface/v1"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type EntryItemList []*accounting_core.JournalEntry

func (lst EntryItemList) toProto() []*accounting_iface.EntryItem {
	result := []*accounting_iface.EntryItem{}
	for _, item := range lst {
		result = append(result, &accounting_iface.EntryItem{
			Id:        uint64(item.ID),
			AccountId: uint64(item.AccountID),
			EntryTime: item.EntryTime.UnixMicro(),
			Desc:      item.Desc,
			Debit:     item.Debit,
			Credit:    item.Credit,
		})
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
		Entries: list.toProto(),
	}

	return connect.NewResponse(&result), nil

}
