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

		// debugtool.LogJson(account)

		if !entry.Rollback {
			debit = entry.Debit
			credit = entry.Credit
		} else {
			debit = entry.Credit * -1
			credit = entry.Debit * -1
		}

		switch account.BalanceType {
		case accounting_core.DebitBalance:
			balance = entry.Debit - entry.Credit
		case accounting_core.CreditBalance:
			balance = entry.Credit - entry.Debit
		default:
			return &connect.Response[report_iface.DailyUpdateBalanceResponse]{}, errors.New("account not credit or debit")
		}

		keyDailyBalance := &accounting_core.AccountKeyDailyBalance{
			Day:           day,
			JournalTeamID: uint(entry.TeamId),
			AccountKey:    account.AccountKey,
			Debit:         debit,
			Credit:        credit,
			Balance:       balance,
		}

		err = a.updateDailyBalance(
			a.
				db.
				Model(&accounting_core.AccountKeyDailyBalance{}).
				Where("day = ?", keyDailyBalance.Day).
				Where("account_key = ?", keyDailyBalance.AccountKey).
				Where("journal_team_id = ?", keyDailyBalance.JournalTeamID),
			keyDailyBalance,
		)

		if err != nil {
			return &connect.Response[report_iface.DailyUpdateBalanceResponse]{}, err
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

func (a *accountReportImpl) updateDailyBalance(query *gorm.DB, daily accounting_core.DailyBalance) error {
	var err error
	var incBalance float64

	debit, credit, balance := daily.GetDebitCredit()
	incBalance += balance

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

		var beforeBalance float64

		err = a.db.Transaction(func(tx *gorm.DB) error {
			err = daily.
				Before(a.db, true).
				Select([]string{
					"balance",
				}).
				Order("day desc").
				Limit(1).
				Find(&beforeBalance).
				Error

			if err != nil {
				return err
			}

			daily.AddBalance(beforeBalance)
			err = a.
				db.
				Save(daily).
				Error

			if err != nil {
				return err
			}

			return nil
		})

		incBalance += beforeBalance
	}

	err = a.updateAfterIncrement(daily, incBalance)

	return err

}

func (a *accountReportImpl) updateAfterIncrement(daily accounting_core.DailyBalance, incAmount float64) error {
	err := daily.
		After(a.db, false).
		Updates(map[string]interface{}{
			"balance": gorm.Expr("balance + ?", incAmount),
		}).
		Error

	return err
}

// begin;
// lock table account_daily_balances in ACCESS exclusive mode;

// with entries as (
// 	select
// 		date(je.entry_time AT TIME ZONE 'Asia/Jakarta')::timestamp AT TIME ZONE 'Asia/Jakarta' + INTERVAL '7 hours' as day,
// 		je.account_id,
// 		je.team_id,
// 		case
// 			when je.rollback = true then je.credit * -1
// 			else je.debit
// 		end as debit,
// 		case
// 			when je.rollback = true then je.debit * -1
// 			else je.credit
// 		end as credit,

// 		case
// 			when a.balance_type = 'd' then je.debit - je.credit
// 			when a.balance_type = 'c' then je.credit - je.debit
// 		end as balance

// 	--	*
// 	from
// 		journal_entries je
// 	join accounts a on a.id = je.account_id
// ),

// summary as (
// 	select
// 		en.day,
// 		en.account_id,
// 		en.team_id,
// 		sum(en.debit) as debit,
// 		sum(en.credit) as credit,
// 		sum(en.balance) as balance

// 	from entries en
// 	group by en.day, en.account_id, en.team_id
// ),

// statdata as (
// 	select
// 		su.day,
// 		su.account_id,
// 		su.team_id,
// 		su.debit,
// 		su.credit,
// 		sum(su.balance) over (
// 			partition by su.account_id, su.team_id
// 			order by su.day asc
// 		) as balance

// 	from summary su
// --	where
// --		su.account_id = 251
// --	order by su.day desc
// )

// insert into account_daily_balances (day, account_id, journal_team_id, debit, credit, balance)
// select day, account_id, team_id, debit, credit, balance from statdata
// on conflict (day, account_id, journal_team_id)
// do update
// set
// 	debit=excluded.debit,
// 	credit=excluded.credit,
// 	balance=excluded.balance
// ;

// --SQL Error [23505]: ERROR: duplicate key value violates unique constraint "account_journal"
// --  Detail: Key (day, account_id, journal_team_id)=(2025-09-19 07:00:00+07, 211, 67) already exists.

// --	SQL Error [25P02]: ERROR: current transaction is aborted, commands ignored until end of transaction block
// --  ERROR: current transaction is aborted, commands ignored until end of transaction block
// --  ERROR: current transaction is aborted, commands ignored until end of transaction block
// --    ERROR: duplicate key value violates unique constraint "account_journal"
// --  Detail: Key (day, account_id, journal_team_id)=(2025-09-19 07:00:00+07, 211, 67) already exists.
// --    ERROR: duplicate key value violates unique constraint "account_journal"
// --  Detail: Key (day, account_id, journal_team_id)=(2025-09-19 07:00:00+07, 211, 67) already exists.

// commit;
