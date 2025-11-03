package report

import (
	"context"

	"connectrpc.com/connect"
	"github.com/pdcgo/accounting_service/accounting_core"
	"github.com/pdcgo/schema/services/common/v1"
	"github.com/pdcgo/schema/services/report_iface/v1"
	"gorm.io/gorm"
)

// MonthlyBalance implements report_ifaceconnect.AccountReportServiceHandler.
func (a *accountReportImpl) MonthlyBalance(
	ctx context.Context,
	req *connect.Request[report_iface.MonthlyBalanceRequest]) (*connect.Response[report_iface.MonthlyBalanceResponse], error) {

	var err error
	result := report_iface.MonthlyBalanceResponse{
		Data:     []*report_iface.MonthlyAccountBalanceItem{},
		PageInfo: &common.PageInfo{},
	}
	pay := req.Msg

	db := a.db.WithContext(ctx)

	query := createMonthlyReportQ(db, pay)

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

	var itemcount int64

	err = db.
		Table("(?) as base", createMonthlyReportQ(db, pay)).
		Select([]string{
			"count(1)",
		}).
		Find(&itemcount).
		Error
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

	return connect.NewResponse(&result), err
}

func createMonthlyReportQ(db *gorm.DB, pay *report_iface.MonthlyBalanceRequest) *gorm.DB {
	query := db.
		Table("account_key_daily_balances adb").
		Select([]string{
			"(EXTRACT(EPOCH FROM date_trunc('month', adb.day)) * 1000000)::BIGINT as month",
			"sum(adb.debit) as debit",
			"sum(adb.credit) as credit",
			"sum(adb.balance) as balance",
		}).
		Group("month").
		Order("month desc")

	if pay.TeamId != 0 {
		query = query.
			Where("adb.journal_team_id = ?", pay.TeamId)
	}

	if pay.AccountKey != "" {
		query = query.
			Where("adb.account_key = ?", pay.AccountKey)
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

	return query
}
