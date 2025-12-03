package account

import (
	"context"
	"errors"
	"fmt"
	"time"

	"connectrpc.com/connect"
	"github.com/pdcgo/accounting_service/accounting_model"
	"github.com/pdcgo/schema/services/accounting_iface/v1"
	"github.com/pdcgo/schema/services/common/v1"
	"github.com/pdcgo/shared/custom_connect"
	"github.com/pdcgo/shared/interfaces/authorization_iface"
	"gorm.io/gorm"
)

// AccountMutationList implements accounting_ifaceconnect.AccountServiceHandler.
func (a *accountServiceImpl) AccountMutationList(
	ctx context.Context,
	req *connect.Request[accounting_iface.AccountMutationListRequest],
) (
	*connect.Response[accounting_iface.AccountMutationListResponse],
	error,
) {
	var err error
	result := accounting_iface.AccountMutationListResponse{
		Data:     []*accounting_iface.MutationItem{},
		PageInfo: &common.PageInfo{},
	}

	source, err := custom_connect.GetRequestSource(ctx)
	if err != nil {
		return nil, err
	}

	pay := req.Msg

	if pay.Page == nil {
		return connect.NewResponse(&result), errors.New("page must set")
	}

	if pay.Page.Limit == 0 {
		pay.Page.Limit = 100
	}
	if pay.Page.Page == 0 {
		pay.Page.Page = 1
	}

	db := a.db.WithContext(ctx)

	identity := a.
		auth.
		AuthIdentityFromHeader(req.Header())

	err = identity.
		HasPermission(authorization_iface.CheckPermissionGroup{
			&accounting_model.BankTransfer{}: &authorization_iface.CheckPermission{
				DomainID: uint(source.TeamId),
				Actions:  []authorization_iface.Action{authorization_iface.Read},
			},
		}).
		Err()

	if err != nil {
		return connect.NewResponse(&result), err
	}

	createQuery := func() *gorm.DB {

		query := db.
			Table("bank_transfer_histories bth").
			Select(
				"bth.id",
				"bth.team_id",
				"bth.from_account_id",
				"bth.to_account_id",
				"bth.amount",
				"bth.fee_amount",
				"bth.purpose",
				"bth.desc",
				"(EXTRACT(EPOCH FROM bth.transfer_at) * 1000000)::BIGINT AS transfer_at",
				"(EXTRACT(EPOCH FROM bth.created) * 1000000)::BIGINT AS created",
				fmt.Sprintf(`
				case 
					when fac.team_id = %d then 1
					else 2
				end as type`, pay.TeamId),
			).
			Joins("left join bank_account_v2 fac on fac.id = bth.from_account_id").
			Joins("left join bank_account_v2 tac on tac.id = bth.to_account_id").
			Where("fac.team_id = ? or tac.team_id = ?", pay.TeamId, pay.TeamId)

			// filtering
		if pay.AccountId != 0 {
			query = query.
				Where("fac.id = ? or tac.id = ?", pay.AccountId, pay.AccountId)
		}

		if pay.TimeRange != nil {
			trange := pay.TimeRange
			if trange.EndDate != 0 {
				query = query.Where("bth.created <= ?",
					time.UnixMicro(trange.EndDate).Local(),
				)
			}

			if trange.StartDate != 0 {
				query = query.Where("bth.created > ?",
					time.UnixMicro(trange.StartDate).Local(),
				)
			}

		}

		return query
	}

	query := createQuery()

	page := pay.Page.Page
	offset := (page - 1) * pay.Page.Limit
	err = query.
		Offset(int(offset)).
		Limit(int(pay.Page.Limit)).
		Find(&result.Data).
		Error

	if err != nil {
		return connect.NewResponse(&result), err
	}

	// pagination
	var itemcount int64
	err = createQuery().
		Select("count(1)").
		Count(&itemcount).Error
	if err != nil {
		return connect.NewResponse(&result), err
	}
	var total int64 = int64(itemcount / pay.Page.Limit)
	if total == 0 {
		total = 1
	}

	result.PageInfo = &common.PageInfo{
		CurrentPage: pay.Page.Page,
		TotalPage:   total,
		TotalItems:  itemcount,
	}

	// preload

	accountIDs := []uint64{}
	fromMap := map[uint64][]*accounting_iface.MutationItem{}
	toMap := map[uint64][]*accounting_iface.MutationItem{}
	for _, item := range result.Data {
		if toMap[item.ToAccountId] == nil {
			toMap[item.ToAccountId] = []*accounting_iface.MutationItem{}
		}

		if fromMap[item.FromAccountId] == nil {
			fromMap[item.FromAccountId] = []*accounting_iface.MutationItem{}
		}

		accountIDs = append(accountIDs, item.FromAccountId, item.ToAccountId)

		fromMap[item.FromAccountId] = append(fromMap[item.FromAccountId], item)
		toMap[item.ToAccountId] = append(toMap[item.ToAccountId], item)
	}

	accounts := []*accounting_iface.PublicAccountItem{}
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
		Where("bav.id in ?", accountIDs).
		Find(&accounts).
		Error

	if err != nil {
		return connect.NewResponse(&result), err
	}

	for _, d := range accounts {
		acc := d
		for _, item := range fromMap[acc.Id] {
			item.FromAccount = acc
		}

		for _, item := range toMap[acc.Id] {
			item.ToAccount = acc
		}

	}

	return connect.NewResponse(&result), err

}
