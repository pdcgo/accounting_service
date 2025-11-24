package account

import (
	"context"

	"connectrpc.com/connect"
	"github.com/pdcgo/schema/services/accounting_iface/v1"
)

// AccountByIDs implements accounting_ifaceconnect.AccountServiceHandler.
func (a *accountServiceImpl) AccountByIDs(ctx context.Context, req *connect.Request[accounting_iface.AccountByIDsRequest]) (*connect.Response[accounting_iface.AccountByIDsResponse], error) {
	var err error
	db := a.db.WithContext(ctx)

	result := accounting_iface.AccountByIDsResponse{
		Data: map[uint64]*accounting_iface.PublicAccountItem{},
	}

	list := []*accounting_iface.PublicAccountItem{}
	err = db.
		Table("bank_account_v2 bav").
		Joins("join account_types at2 on at2.id = bav.account_type_id").
		Select([]string{
			"bav.id",
			"bav.team_id",
			"bav.name",
			"bav.number_id",
			"at2.key as account_type",
		}).
		Find(&list).
		Error

	if err != nil {
		return connect.NewResponse(&result), err
	}

	for _, dd := range list {
		item := dd
		result.Data[item.Id] = item
	}

	return connect.NewResponse(&result), nil

}
