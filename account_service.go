package accounting_service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/pdcgo/accounting_service/accounting_core"
	"github.com/pdcgo/accounting_service/accounting_model"
	"github.com/pdcgo/schema/services/accounting_iface/v1"
	"github.com/pdcgo/schema/services/common/v1"
	"github.com/pdcgo/shared/authorization"
	"github.com/pdcgo/shared/interfaces/authorization_iface"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const AccountInitialize authorization_iface.Action = "initialize_acc"

func NewAccountService(db *gorm.DB, auth authorization_iface.Authorization) *accountServiceImpl {
	return &accountServiceImpl{
		db:   db,
		auth: auth,
	}
}

type accountServiceImpl struct {
	auth authorization_iface.Authorization
	db   *gorm.DB
}

// TransferCancel implements accounting_ifaceconnect.AccountServiceHandler.
func (a *accountServiceImpl) TransferCancel(context.Context, *connect.Request[accounting_iface.TransferCancelRequest]) (*connect.Response[accounting_iface.TransferCancelResponse], error) {
	panic("unimplemented")
}

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
				DomainID: uint(pay.TeamId),
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
				query = query.Where("bth.transfer_at <= ?",
					time.UnixMicro(trange.EndDate).Local(),
				)
			}

			if trange.StartDate != 0 {
				query = query.Where("bth.transfer_at > ?",
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

// AccountBalanceInit implements accounting_ifaceconnect.AccountServiceHandler.
func (a *accountServiceImpl) AccountBalanceInit(
	ctx context.Context,
	req *connect.Request[accounting_iface.AccountBalanceInitRequest],
) (*connect.Response[accounting_iface.AccountBalanceInitResponse], error) {
	var err error
	result := accounting_iface.AccountBalanceInitResponse{}

	pay := req.Msg
	db := a.db.WithContext(ctx)
	identity := a.
		auth.
		AuthIdentityFromHeader(req.Header())
	agent := identity.Identity()

	err = identity.
		HasPermission(authorization_iface.CheckPermissionGroup{
			&accounting_model.BankTransfer{}: &authorization_iface.CheckPermission{
				DomainID: authorization.RootDomain,
				Actions:  []authorization_iface.Action{AccountInitialize},
			},
		}).
		Err()

	if err != nil {
		return connect.NewResponse(&result), err
	}

	err = accounting_core.OpenTransaction(db, func(tx *gorm.DB, bookmng accounting_core.BookManage) error {
		var account accounting_model.BankAccountV2
		trans := accounting_core.Transaction{
			CreatedByID: agent.GetUserID(),
			Desc:        pay.Desc,
			Created:     time.Now(),
		}
		err = tx.
			Clauses(clause.Locking{
				Strength: "UPDATE",
			}).
			Model(&accounting_model.BankAccountV2{}).
			First(&account, pay.AccountId).
			Error

		if err != nil {
			return err
		}

		if account.Balance != 0 {
			return errors.New("account balance not Empty")
		}

		err = bookmng.
			NewTransaction().
			Create(&trans).
			Err()

		if err != nil {
			return err
		}

		err = bookmng.
			NewCreateEntry(account.TeamID, agent.GetUserID()).
			From(&accounting_core.EntryAccountPayload{
				Key:    accounting_core.CapitalStartAccount,
				TeamID: authorization.RootDomain,
			}, pay.Amount).
			To(&accounting_core.EntryAccountPayload{
				Key:    accounting_core.CashAccount,
				TeamID: account.TeamID,
			}, pay.Amount).
			Transaction(&trans).
			Commit().
			Err()

		if err != nil {
			return err
		}

		account.Balance = pay.Amount
		err = tx.Save(&account).Error
		if err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return connect.NewResponse(&result), err
	}

	return connect.NewResponse(&result), nil
}

// TransferCreate implements accounting_ifaceconnect.AccountServiceHandler.
func (a *accountServiceImpl) TransferCreate(
	ctx context.Context,
	req *connect.Request[accounting_iface.TransferCreateRequest],
) (*connect.Response[accounting_iface.TransferCreateResponse], error) {
	var err error
	result := accounting_iface.TransferCreateResponse{}

	pay := req.Msg
	identity := a.
		auth.
		AuthIdentityFromHeader(req.Header())

	agent := identity.Identity()
	err = identity.
		HasPermission(authorization_iface.CheckPermissionGroup{
			&accounting_model.BankTransfer{}: &authorization_iface.CheckPermission{
				DomainID: uint(pay.TeamId),
				Actions:  []authorization_iface.Action{authorization_iface.Create},
			},
		}).
		Err()

	if err != nil {
		return connect.NewResponse(&result), err
	}
	db := a.db.WithContext(ctx)
	accounting_core.OpenTransaction(db, func(tx *gorm.DB, bookmng accounting_core.BookManage) error {
		var facc accounting_model.BankAccountV2
		var tacc accounting_model.BankAccountV2
		txclause := func() *gorm.DB {
			return tx.
				Clauses(clause.Locking{
					Strength: "UPDATE",
				})
		}

		err = txclause().
			First(&facc, pay.FromAccountId).
			Error
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return fmt.Errorf("account id %d not found", pay.FromAccountId)
			} else {
				return err
			}

		}

		if facc.TeamID != uint(pay.TeamId) {
			return fmt.Errorf("your team not have %s", facc.Name)
		}

		err = txclause().
			First(&tacc, pay.ToAccountId).
			Error
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return fmt.Errorf("account id %d not found", pay.ToAccountId)
			} else {
				return err
			}
		}

		if facc.ID == tacc.ID {
			return errors.New("account same")
		}

		trans := accounting_core.Transaction{
			CreatedByID: agent.IdentityID(),
			Desc:        pay.Desc,
			Created:     time.Now(),
		}

		err = bookmng.
			NewTransaction().
			Create(&trans).
			Err()

		if err != nil {
			return err
		}

		hist := accounting_model.BankTransferHistory{
			TeamID:        uint(pay.TeamId),
			FromAccountID: uint(pay.FromAccountId),
			ToAccountID:   uint(pay.ToAccountId),
			TxID:          trans.ID,
			Amount:        pay.Amount,
			FeeAmount:     pay.FeeAmount,
			Desc:          pay.Desc,
			TransferAt:    time.UnixMicro(pay.TransferAt).Local(),
			Created:       time.Now(),
		}

		err = tx.Save(&hist).Error
		if err != nil {
			return err
		}
		// book from
		entry := bookmng.
			NewCreateEntry(uint(pay.TeamId), agent.GetUserID()).
			From(&accounting_core.EntryAccountPayload{
				Key:    accounting_core.CashAccount,
				TeamID: facc.TeamID,
			}, pay.Amount+pay.FeeAmount).
			To(&accounting_core.EntryAccountPayload{
				Key:    accounting_core.CashAccount,
				TeamID: tacc.TeamID,
			}, pay.Amount)

		if pay.FeeAmount != 0 {
			entry.
				To(&accounting_core.EntryAccountPayload{
					Key:    accounting_core.BankFeeAccount,
					TeamID: facc.TeamID,
				}, pay.FeeAmount)
		}
		err = entry.
			Transaction(&trans).
			Commit().
			Err()
		if err != nil {
			return err
		}

		// book to
		entry = bookmng.
			NewCreateEntry(tacc.TeamID, agent.GetUserID()).
			From(&accounting_core.EntryAccountPayload{
				Key:    accounting_core.CashAccount,
				TeamID: facc.TeamID,
			}, pay.Amount).
			To(&accounting_core.EntryAccountPayload{
				Key:    accounting_core.CashAccount,
				TeamID: tacc.TeamID,
			}, pay.Amount)

		err = entry.
			Transaction(&trans).
			Commit().
			Err()
		if err != nil {
			return err
		}

		return nil
	})

	return connect.NewResponse(&result), err
}

// AccountTypeList implements accounting_ifaceconnect.AccountServiceHandler.
func (a *accountServiceImpl) AccountTypeList(
	ctx context.Context,
	req *connect.Request[accounting_iface.AccountTypeListRequest],
) (*connect.Response[accounting_iface.AccountTypeListResponse], error) {
	var err error
	result := accounting_iface.AccountTypeListResponse{
		Data: []*accounting_iface.AccountTypeItem{},
	}

	err = a.
		db.
		WithContext(ctx).
		Table("account_types").
		Select([]string{
			"key",
			"name",
		}).
		Find(&result.Data).
		Error

	if err != nil {
		return connect.NewResponse(&result), err
	}

	return connect.NewResponse(&result), nil
}

// AccountCreate implements accounting_ifaceconnect.AccountServiceHandler.
func (a *accountServiceImpl) AccountCreate(
	ctx context.Context,
	req *connect.Request[accounting_iface.AccountCreateRequest],
) (*connect.Response[accounting_iface.AccountCreateResponse], error) {
	var err error
	res := connect.NewResponse(&accounting_iface.AccountCreateResponse{})

	pay := req.Msg
	identity := a.
		auth.
		AuthIdentityFromHeader(req.Header())
	err = identity.
		HasPermission(authorization_iface.CheckPermissionGroup{
			&accounting_model.BankAccountV2{}: &authorization_iface.CheckPermission{
				DomainID: uint(pay.TeamId),
				Actions:  []authorization_iface.Action{authorization_iface.Create},
			},
		}).
		Err()

	if err != nil {
		return res, err
	}

	err = a.
		db.
		WithContext(ctx).
		Transaction(func(tx *gorm.DB) error {
			acc := accounting_model.BankAccountV2{
				TeamID:        uint(pay.TeamId),
				Name:          pay.Name,
				NumberID:      pay.NumberId,
				AccountTypeID: uint(pay.AccountTypeId),
				CreatedAt:     time.Now(),
			}
			err = tx.Save(&acc).Error
			if err != nil {
				if errors.Is(err, gorm.ErrDuplicatedKey) {
					return errors.New("account duplicated")
				}
				return err
			}

			if len(pay.Labels) != 0 {
				for _, label := range pay.Labels {
					lab := accounting_model.BankAccountLabel{
						Key:   label.Key,
						Value: label.Name,
					}

					err = tx.Where("key = ?", lab.Key).Find(&lab).Error
					if err != nil {
						return err
					}
					if lab.ID == 0 {
						err = tx.Save(&lab).Error
						if err != nil {
							return err
						}
					}

					rel := accounting_model.BankAccountLabelRelation{
						AccountID: acc.ID,
						LabelID:   lab.ID,
					}

					err = tx.Save(&rel).Error
					if err != nil {
						return err
					}
				}

			}

			return err
		})

	if err != nil {
		return res, err
	}

	return res, nil

}

// AccountDelete implements accounting_ifaceconnect.AccountServiceHandler.
func (a *accountServiceImpl) AccountDelete(
	ctx context.Context,
	req *connect.Request[accounting_iface.AccountDeleteRequest],
) (*connect.Response[accounting_iface.AccountDeleteResponse], error) {

	var err error
	res := connect.NewResponse(&accounting_iface.AccountDeleteResponse{})

	pay := req.Msg
	identity := a.
		auth.
		AuthIdentityFromHeader(req.Header())
	err = identity.
		HasPermission(authorization_iface.CheckPermissionGroup{
			&accounting_model.BankAccountV2{}: &authorization_iface.CheckPermission{
				DomainID: uint(pay.TeamId),
				Actions:  []authorization_iface.Action{authorization_iface.Create},
			},
		}).
		Err()

	if err != nil {
		return res, err
	}

	err = a.
		db.
		WithContext(ctx).
		Transaction(func(tx *gorm.DB) error {
			for _, acc := range pay.AccountIds {
				err = tx.
					Model(&accounting_model.BankAccountV2{}).
					Where("id = ?", acc).
					Updates(map[string]interface{}{
						"deleted":   true,
						"number_id": gorm.Expr("number_id || ?", "_deleted"),
					}).
					Error
				if err != nil {
					return err
				}
			}
			return nil
		})

	return res, nil
}

type RelKeyName struct {
	RelID uint64
	common.KeyName
}

// AccountList implements accounting_ifaceconnect.AccountServiceHandler.
func (a *accountServiceImpl) AccountList(
	ctx context.Context,
	req *connect.Request[accounting_iface.AccountListRequest],
) (*connect.Response[accounting_iface.AccountListResponse], error) {
	var err error
	result := accounting_iface.AccountListResponse{
		Data: []*accounting_iface.AccountItem{},
	}

	pay := req.Msg
	identity := a.
		auth.
		AuthIdentityFromHeader(req.Header())
	err = identity.
		HasPermission(authorization_iface.CheckPermissionGroup{
			&accounting_model.BankAccountV2{}: &authorization_iface.CheckPermission{
				DomainID: uint(pay.TeamId),
				Actions:  []authorization_iface.Action{authorization_iface.Create},
			},
		}).
		Err()

	if err != nil {
		return connect.NewResponse(&result), err
	}

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

	err = query.
		Find(&result.Data).
		Error

	if err != nil {
		return connect.NewResponse(&result), err
	}

	accIDs := []uint64{}
	mapAcc := map[uint64]*accounting_iface.AccountItem{}
	for _, dd := range result.Data {
		item := dd
		accIDs = append(accIDs, item.Id)
		mapAcc[item.Id] = item
		item.Labels = []*common.KeyName{}
	}

	items := []*RelKeyName{}
	err = db.
		Table("bank_account_labels bal").
		Select([]string{
			"balr.account_id as rel_id",
			"bal.key",
			"bal.value as name",
		}).
		Joins("join bank_account_label_relations balr on balr.label_id = bal.id").
		Where("balr.account_id in ?", accIDs).
		Find(&items).
		Error

	if err != nil {
		return connect.NewResponse(&result), err
	}

	for _, dd := range items {
		item := dd
		mapAcc[item.RelID].Labels = append(mapAcc[item.RelID].Labels, &item.KeyName)
	}
	return connect.NewResponse(&result), err
}

// AccountUpdate implements accounting_ifaceconnect.AccountServiceHandler.
func (a *accountServiceImpl) AccountUpdate(
	ctx context.Context,
	req *connect.Request[accounting_iface.AccountUpdateRequest],
) (*connect.Response[accounting_iface.AccountUpdateResponse], error) {
	// var err error
	result := accounting_iface.AccountUpdateResponse{}

	return connect.NewResponse(&result), errors.New("not_implemented")
}

// LabelList implements accounting_ifaceconnect.AccountServiceHandler.
func (a *accountServiceImpl) LabelList(
	ctx context.Context,
	req *connect.Request[accounting_iface.LabelListRequest],
) (*connect.Response[accounting_iface.LabelListResponse], error) {
	var err error
	result := accounting_iface.LabelListResponse{
		Data: []*common.KeyName{},
	}

	pay := req.Msg

	query := a.
		db.
		WithContext(ctx).
		Table("bank_account_labels d").
		Select([]string{"key", "value as name"})

	if pay.Keyword != "" {
		query = query.
			Where("LOWER(d.value) LIKE ?", "%"+strings.ToLower(pay.Keyword)+"%")
	}

	err = query.
		Find(&result.Data).
		Error

	if err != nil {
		return connect.NewResponse(&result), err
	}

	return connect.NewResponse(&result), nil

}
