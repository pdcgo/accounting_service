package report

import (
	"context"
	"errors"
	"fmt"
	"time"

	"connectrpc.com/connect"
	"github.com/pdcgo/accounting_service/accounting_core"
	"github.com/pdcgo/schema/services/report_iface/v1"
	"github.com/pdcgo/shared/pkg/ware_cache"
	"gorm.io/gorm"
)

func (a *accountReportImpl) getAccount(ctx context.Context, accID uint) (*accounting_core.Account, error) {
	var err error
	var acc accounting_core.Account
	key := fmt.Sprintf("accounting/account/%d", accID)

	err = a.cache.Get(ctx, key, &acc)
	if err != nil {
		if !errors.Is(err, ware_cache.ErrCacheMiss) {
			return &acc, err
		}
		err = a.db.Model(&accounting_core.Account{}).First(&acc, accID).Error
		if err != nil {
			return &acc, err
		}

		err = a.cache.Add(ctx, &ware_cache.CacheItem{
			Key:        key,
			Expiration: time.Minute * 15,
			Data:       &acc,
		})
		if err != nil {
			return &acc, err
		}
	}

	if acc.ID == 0 {
		return &acc, fmt.Errorf("something problem to get account %d", accID)
	}

	return &acc, nil
}

// DailyUpdateBalance implements report_ifaceconnect.AccountReportServiceHandler.
func (a *accountReportImpl) DailyUpdateBalance(
	ctx context.Context,
	req *connect.Request[report_iface.DailyUpdateBalanceRequest],
) (*connect.Response[report_iface.DailyUpdateBalanceResponse], error) {

	var err error
	// identity := a.auth.AuthIdentityFromHeader(req.Header())
	// agent := identity.Identity()
	// err = identity.
	// 	Err()
	// if err != nil {
	// 	return connect.NewResponse(&report_iface.DailyUpdateBalanceResponse{}), err
	// }

	// if !agent.IsSuperUser() {
	// 	return &connect.Response[report_iface.DailyUpdateBalanceResponse]{}, errors.New("not allowed")
	// }

	pay := req.Msg
	labels := pay.LabelExtra
	for _, entry := range pay.Entries {
		day := accounting_core.ParseDate(entry.EntryTime.AsTime())
		var balance, debit, credit float64
		account, err := a.getAccount(ctx, uint(entry.AccountId))
		if err != nil {
			return &connect.Response[report_iface.DailyUpdateBalanceResponse]{}, err
		}

		if !entry.Rollback {
			debit = entry.Debit
			credit = entry.Credit
		} else {
			debit = entry.Debit * -1
			credit = entry.Credit * -1
		}

		switch account.BalanceType {
		case accounting_core.DebitBalance:
			balance = debit - credit
		case accounting_core.CreditBalance:
			balance = credit - debit
		}

		dayBalance := &accounting_core.AccountDailyBalance{
			Day:           day,
			AccountID:     uint(entry.AccountId),
			JournalTeamID: uint(entry.TeamId),
			Debit:         debit,
			Credit:        credit,
			Balance:       balance,
		}

		err = a.updateDailyBalance(
			a.
				db.
				Model(&accounting_core.AccountDailyBalance{}).
				Where("day = ?", dayBalance.Day).
				Where("account_id = ?", dayBalance.AccountID).
				Where("journal_team_id = ?", dayBalance.JournalTeamID),
			dayBalance,
		)

		if err != nil {
			return &connect.Response[report_iface.DailyUpdateBalanceResponse]{}, err
		}

		if labels.CsId != 0 {
			csDayBalance := &accounting_core.CsDailyBalance{
				Day:           day,
				CsID:          uint(labels.CsId),
				AccountID:     uint(entry.AccountId),
				JournalTeamID: uint(entry.TeamId),
				Debit:         debit,
				Credit:        credit,
				Balance:       balance,
			}

			err = a.updateDailyBalance(
				a.
					db.
					Model(&accounting_core.CsDailyBalance{}).
					Where("cs_id = ?", csDayBalance.CsID).
					Where("day = ?", csDayBalance.Day).
					Where("account_id = ?", csDayBalance.AccountID).
					Where("journal_team_id = ?", csDayBalance.JournalTeamID),
				csDayBalance,
			)
			if err != nil {
				return &connect.Response[report_iface.DailyUpdateBalanceResponse]{}, err
			}
		}

		if labels.ShopId != 0 {
			shopDayBalance := &accounting_core.ShopDailyBalance{
				Day:           day,
				ShopID:        uint(labels.ShopId),
				AccountID:     uint(entry.AccountId),
				JournalTeamID: uint(entry.TeamId),
				Debit:         debit,
				Credit:        credit,
				Balance:       balance,
			}

			err = a.updateDailyBalance(
				a.
					db.
					Model(&accounting_core.ShopDailyBalance{}).
					Where("shop_id = ?", shopDayBalance.ShopID).
					Where("day = ?", shopDayBalance.Day).
					Where("account_id = ?", shopDayBalance.AccountID).
					Where("journal_team_id = ?", shopDayBalance.JournalTeamID),
				shopDayBalance,
			)
			if err != nil {
				return &connect.Response[report_iface.DailyUpdateBalanceResponse]{}, err
			}

		}

		if labels.SupplierId != 0 {
			supplierDayBalance := &accounting_core.SupplierDailyBalance{
				Day:           day,
				SupplierID:    uint(labels.SupplierId),
				AccountID:     uint(entry.AccountId),
				JournalTeamID: uint(entry.TeamId),
				Debit:         debit,
				Credit:        credit,
				Balance:       balance,
			}

			err = a.updateDailyBalance(
				a.
					db.
					Model(&accounting_core.SupplierDailyBalance{}).
					Where("supplier_id = ?", supplierDayBalance.SupplierID).
					Where("day = ?", supplierDayBalance.Day).
					Where("account_id = ?", supplierDayBalance.AccountID).
					Where("journal_team_id = ?", supplierDayBalance.JournalTeamID),
				supplierDayBalance,
			)

			if err != nil {
				return &connect.Response[report_iface.DailyUpdateBalanceResponse]{}, err
			}
		}

		if labels.TagIds != nil {
			for _, tagID := range labels.TagIds {

				customDayBalance := &accounting_core.CustomLabelDailyBalance{
					Day:           day,
					CustomID:      uint(tagID),
					AccountID:     uint(entry.AccountId),
					JournalTeamID: uint(entry.TeamId),
					Debit:         debit,
					Credit:        credit,
					Balance:       balance,
				}

				err = a.updateDailyBalance(
					a.
						db.
						Model(&accounting_core.CustomLabelDailyBalance{}).
						Where("custom_id = ?", customDayBalance.CustomID).
						Where("day = ?", customDayBalance.Day).
						Where("account_id = ?", customDayBalance.AccountID).
						Where("journal_team_id = ?", customDayBalance.JournalTeamID),
					customDayBalance,
				)

				if err != nil {
					return &connect.Response[report_iface.DailyUpdateBalanceResponse]{}, err
				}
			}
		}
	}

	return &connect.Response[report_iface.DailyUpdateBalanceResponse]{}, err
}

type DailyBalance interface {
	GetDebitCredit() (debit float64, credit float64, balance float64)
}

func (a *accountReportImpl) updateDailyBalance(query *gorm.DB, daily DailyBalance) error {
	var err error

	debit, credit, balance := daily.GetDebitCredit()

	row := query.
		Updates(map[string]interface{}{
			"debit":   gorm.Expr("debit + ?", debit),
			"credit":  gorm.Expr("credit + ?", credit),
			"balance": gorm.Expr("balance + ?", balance),
		})

	if row.RowsAffected == 0 {
		err = row.Error
		if err != nil {
			return err
		}

		err = a.
			db.
			Save(daily).
			Error

		if err != nil {
			return err
		}
	}

	return err

}
