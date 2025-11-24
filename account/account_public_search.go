package account

import (
	"context"

	"connectrpc.com/connect"
	"github.com/pdcgo/schema/services/accounting_iface/v1"
)

// AccountPublicSearch implements accounting_ifaceconnect.AccountServiceHandler.
func (a *accountServiceImpl) AccountPublicSearch(
	ctx context.Context, req *connect.Request[accounting_iface.AccountPublicSearchRequest]) (*connect.Response[accounting_iface.AccountPublicSearchResponse], error) {
	var err error
	pay := req.Msg
	db := a.db.WithContext(ctx)

	query := db.
		Table("bank_account_v2 bav").
		Joins("join account_types at2 on at2.id = bav.account_type_id").
		Select([]string{
			"bav.id",
			"bav.team_id",
			"bav.name",
			"bav.number_id",
			"at2.key as account_type",
		})

	if pay.TeamId != 0 {
		query = query.
			Where("bav.team_id = ?", pay.TeamId)
	}

	if pay.Keyword != "" {
		query = query.
			Where("LOWER(bav.name) LIKE ?", "%"+pay.Keyword+"%")
	}

	result := accounting_iface.AccountPublicSearchResponse{
		Data: []*accounting_iface.PublicAccountItem{},
	}

	err = query.
		Find(&result.Data).
		Error

	if err != nil {
		return connect.NewResponse(&result), err
	}

	return connect.NewResponse(&result), nil

}
