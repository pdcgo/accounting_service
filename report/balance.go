package report

import (
	"context"

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
}

type balanceViewImpl struct {
	tx  *gorm.DB
	db  *gorm.DB
	pay *report_iface.BalanceRequest
	// err error
}

// Iterate implements BalanceView.
func (b *balanceViewImpl) Iterate(handle func(d *report_iface.AccountBalanceItem) error) error {
	var err error

	baseQuery := b.dcQuery()
	lbQuery := b.lastBalanceQuery()
	sbQuery := b.startBalanceQuery()

	query := b.
		db.
		Table("(?) as base", baseQuery).
		Joins("full outer join (?) as bal on bal.account_key = base.account_key", lbQuery).
		Joins("full outer join (?) as start on start.account_key = base.account_key", sbQuery)

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

func (b *balanceViewImpl) lastBalanceQuery() *gorm.DB {
	query := b.
		db.
		Table("account_key_daily_balances adb").
		Select([]string{
			"distinct on (adb.account_key) adb.account_key",
			"adb.balance",
		})

	pay := b.pay
	trange := b.pay.TimeRange
	if trange.EndDate.IsValid() {
		query = query.Where("adb.day <= ?",
			trange.EndDate.AsTime(),
		)
	}

	if trange.StartDate.IsValid() {
		query = query.Where("adb.day > ?",
			trange.StartDate.AsTime(),
		)
	}

	if pay.TeamId != 0 {
		query = query.
			Where("adb.journal_team_id = ?", pay.TeamId)
	}

	if len(pay.AccountKeys) != 0 {
		query = query.
			Where("adb.account_key in ?", pay.AccountKeys)
	}

	query = query.
		Order("adb.account_key, adb.day desc")

	return query
}

func (b *balanceViewImpl) startBalanceQuery() *gorm.DB {
	query := b.
		db.
		Table("account_key_daily_balances adb").
		Select([]string{
			"distinct on (adb.account_key) adb.account_key",
			"adb.start_balance",
		})

	pay := b.pay
	trange := b.pay.TimeRange
	if trange.EndDate.IsValid() {
		query = query.Where("adb.day < ?",
			trange.EndDate.AsTime(),
		)
	}

	if trange.StartDate.IsValid() {
		query = query.Where("adb.day > ?",
			trange.StartDate.AsTime(),
		)
	}

	if pay.TeamId != 0 {
		query = query.
			Where("adb.journal_team_id = ?", pay.TeamId)
	}

	if len(pay.AccountKeys) != 0 {
		query = query.
			Where("adb.account_key in ?", pay.AccountKeys)
	}

	query = query.
		Order("adb.account_key, adb.day asc")

	return query
}

// createQuery implements BalanceView.
func (b *balanceViewImpl) dcQuery() *gorm.DB {
	query := b.
		db.
		Table("account_key_daily_balances adb").
		Select([]string{
			"adb.account_key",
			"sum(adb.debit) as debit",
			"sum(adb.credit) as credit",
		})

	pay := b.pay

	query = query.
		Group("adb.account_key")

	if pay.TeamId != 0 {
		query = query.
			Where("adb.journal_team_id = ?", pay.TeamId)
	}

	trange := pay.TimeRange
	if trange.EndDate.IsValid() {
		end := accounting_core.ParseDate(trange.EndDate.AsTime())
		query = query.Where("adb.day <= ?",
			end,
		)
	}

	if trange.StartDate.IsValid() {
		start := accounting_core.ParseDate(trange.StartDate.AsTime())
		query = query.Where("adb.day > ?",
			start,
		)
	}

	if len(pay.AccountKeys) != 0 {
		query = query.
			Where("adb.account_key in ?", pay.AccountKeys)
	}

	return query
}

func NewBalanceView(db *gorm.DB, pay *report_iface.BalanceRequest) BalanceView {
	return &balanceViewImpl{
		tx:  db,
		db:  db,
		pay: pay,
	}
}
