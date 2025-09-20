package report

import (
	"context"
	"time"

	"connectrpc.com/connect"
	"github.com/pdcgo/accounting_service/accounting_core"
	"github.com/pdcgo/schema/services/report_iface/v1"
	"gorm.io/gorm"
)

// Balance implements report_ifaceconnect.AccountReportServiceHandler.
func (a *accountReportImpl) Balance(
	ctx context.Context,
	req *connect.Request[report_iface.BalanceRequest],
) (*connect.Response[report_iface.BalanceResponse], error) {
	var err error
	result := report_iface.BalanceResponse{
		Data: []*report_iface.AccountBalanceItem{},
	}
	db := a.db.WithContext(ctx)
	pay := req.Msg

	view := NewBalanceView(db, pay)
	err = view.Iterate(func(d *report_iface.AccountBalanceItem) error {
		result.Data = append(result.Data, d)
		return nil
	})

	return connect.NewResponse(&result), err
}

type BalanceView interface {
	Iterate(handle func(d *report_iface.AccountBalanceItem) error) error
	Err() error
}

type balanceViewImpl struct {
	tx  *gorm.DB
	db  *gorm.DB
	pay *report_iface.BalanceRequest
	// err error
}

// Err implements BalanceView.
func (b *balanceViewImpl) Err() error {
	panic("unimplemented")
}

// Iterate implements BalanceView.
func (b *balanceViewImpl) Iterate(handle func(d *report_iface.AccountBalanceItem) error) error {
	var err error

	baseQuery := b.dcQuery()
	lbQuery := b.lastBalanceQuery()
	query := b.
		db.
		Table("(?) as base", baseQuery).
		Joins("full outer join (?) as bal on bal.account_key = base.account_key", lbQuery)

	rows, err := query.Rows()

	if err != nil {
		return err
	}

	for rows.Next() {
		var d report_iface.AccountBalanceItem
		err = b.db.ScanRows(rows, &d)
		if err != nil {
			return err
		}

		err = handle(&d)
		if err != nil {
			return err
		}
	}

	return nil
}

// createQuery implements BalanceView.
func (b *balanceViewImpl) dcQuery() *gorm.DB {
	query := b.
		db.
		Table("account_daily_balances adb").
		Joins("join accounts a on a.id = adb.account_id").
		Select([]string{
			"a.account_key",
			"sum(adb.debit) as debit",
			"sum(adb.credit) as credit",
		})

	pay := b.pay

	query = query.
		Group("a.account_key")

	if pay.TeamId != 0 {
		query = query.
			Where("adb.journal_team_id = ?", pay.TeamId)
	}

	trange := pay.TimeRange
	if trange.EndDate != 0 {
		end := accounting_core.ParseDate(time.UnixMicro(trange.EndDate))
		query = query.Where("adb.day <= ?",
			end,
		)
	}

	if trange.StartDate != 0 {
		start := accounting_core.ParseDate(time.UnixMicro(trange.StartDate))
		query = query.Where("adb.day > ?",
			start,
		)
	}

	if len(pay.AccountKeys) != 0 {
		query = query.
			Where("a.account_key in ?", pay.AccountKeys)
	}

	return query
}

func (b *balanceViewImpl) lastBalanceQuery() *gorm.DB {
	query := b.
		db.
		Table("account_daily_balances adb").
		Joins("join accounts a on a.id = adb.account_id").
		Select([]string{
			"a.account_key",
			"adb.day",
			"sum(adb.balance) as balance",
		})

	pay := b.pay
	trange := b.pay.TimeRange
	if trange.EndDate != 0 {
		query = query.Where("adb.day <= ?",
			time.UnixMicro(trange.EndDate),
		)
	}

	if trange.StartDate != 0 {
		query = query.Where("adb.day > ?",
			time.UnixMicro(trange.StartDate),
		)
	}

	if pay.TeamId != 0 {
		query = query.
			Where("adb.journal_team_id = ?", pay.TeamId)
	}

	if len(pay.AccountKeys) != 0 {
		query = query.
			Where("a.account_key in ?", pay.AccountKeys)
	}

	query = query.
		Group("a.account_key, adb.day")

	query = b.
		db.
		Table("(?) as daygroup", query).
		Select([]string{
			"distinct on (daygroup.account_key) daygroup.account_key",
			" daygroup.balance",
		})

	return query
}

// func (b *balanceViewImpl) setErr(err error) *balanceViewImpl {
// 	if b.err != nil {
// 		return b
// 	}

// 	if err != nil {
// 		b.err = err
// 	}

// 	return b
// }

func NewBalanceView(db *gorm.DB, pay *report_iface.BalanceRequest) BalanceView {
	return &balanceViewImpl{
		tx:  db,
		db:  db,
		pay: pay,
	}
}
