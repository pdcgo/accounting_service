package report

import (
	"context"
	"fmt"
	"math"

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

	view := monthlyViewImpl{
		db:  db,
		pay: pay,
	}

	query := view.baseQ()

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
		Table("(?) as base", view.baseQ()).
		Select([]string{
			"count(1)",
		}).
		Find(&itemcount).
		Error
	if err != nil {
		return connect.NewResponse(&result), err
	}

	var total int64
	if itemcount < pay.Page.Limit {
		total = 1
	} else {
		total = int64(math.Ceil(float64(itemcount) / float64(pay.Page.Limit)))
	}

	result.PageInfo = &common.PageInfo{
		CurrentPage: pay.Page.Page,
		TotalPage:   total,
		TotalItems:  itemcount,
	}

	return connect.NewResponse(&result), err
}

type monthlyViewImpl struct {
	db  *gorm.DB
	pay *report_iface.MonthlyBalanceRequest
}

func (m *monthlyViewImpl) balanceQ() *gorm.DB {
	pay := m.pay
	keyBalanceQ := m.
		db.
		Table("account_key_daily_balances db").
		Select(`
		distinct on (
			db.account_key,
			date_trunc('month', db.day AT TIME ZONE 'Asia/Jakarta')
		)
		db.account_key,
		date_trunc('month', db.day AT TIME ZONE 'Asia/Jakarta') as month,
		db.balance,
		db.start_balance
	`).
		Where("db.journal_team_id = ?", pay.TeamId).
		Where("db.account_key = ?", pay.AccountKey)

	trange := pay.TimeRange
	if trange.EndDate.IsValid() {
		end := accounting_core.ParseDate(trange.EndDate.AsTime())
		keyBalanceQ = keyBalanceQ.Where("db.day <= ?",
			end,
		)
	}

	if trange.StartDate.IsValid() {
		start := accounting_core.ParseDate(trange.StartDate.AsTime())
		keyBalanceQ = keyBalanceQ.Where("db.day > ?",
			start,
		)
	}

	keyBalanceQ = keyBalanceQ.
		Order(`
		db.account_key,
		month,
		db.day desc
	`)

	bquery := m.
		db.
		Table("(?) as bal", keyBalanceQ).
		Select([]string{
			"(EXTRACT(EPOCH FROM bal.month) * 1000000)::BIGINT as month",
			// "bal.month",
			"sum(bal.balance) as balance",
			"sum(bal.start_balance) as start_balance",
		}).
		Group("bal.month")

	return bquery
}

func (m *monthlyViewImpl) debitCreditQ() *gorm.DB {
	pay := m.pay

	query := m.
		db.
		Table("account_key_daily_balances adb").
		Select([]string{
			"(EXTRACT(EPOCH FROM date_trunc('month', adb.day)) * 1000000)::BIGINT as month",
			"sum(adb.debit) as debit",
			"sum(adb.credit) as credit",
		}).
		Group("month")

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

func (m *monthlyViewImpl) baseQ() *gorm.DB {
	pay := m.pay

	debitCredit := m.debitCreditQ()
	balance := m.balanceQ()

	query := m.
		db.
		Table("(?) as dc", debitCredit).
		Select([]string{
			"dc.month",
			"dc.debit",
			"dc.credit",
			"bal.balance",
			"bal.start_balance",
		}).
		Joins("full outer join (?) as bal on bal.month = dc.month", balance)

	if pay.Sort != nil {
		var sorttype string
		switch pay.Sort.Type {
		case common.SortType_SORT_TYPE_ASC:
			sorttype = "asc"
		case common.SortType_SORT_TYPE_DESC:
			sorttype = "desc"
		default:
			sorttype = "desc"
		}

		query = query.
			Order(fmt.Sprintf("bal.month, dc.month %s", sorttype))

	} else {
		query = query.
			Order("bal.month, dc.month desc")
	}

	return query
}

// func createMonthlyReportQ(db *gorm.DB, pay *report_iface.MonthlyBalanceRequest) *gorm.DB {
// 	query := db.
// 		Table("account_key_daily_balances adb").
// 		Select([]string{
// 			"(EXTRACT(EPOCH FROM date_trunc('month', adb.day)) * 1000000)::BIGINT as month",
// 			"sum(adb.debit) as debit",
// 			"sum(adb.credit) as credit",
// 		}).
// 		Group("month")

// 	if pay.TeamId != 0 {
// 		query = query.
// 			Where("adb.journal_team_id = ?", pay.TeamId)
// 	}

// 	if pay.AccountKey != "" {
// 		query = query.
// 			Where("adb.account_key = ?", pay.AccountKey)
// 	}

// 	trange := pay.TimeRange
// 	if trange.EndDate.IsValid() {
// 		end := accounting_core.ParseDate(trange.EndDate.AsTime())
// 		query = query.Where("adb.day <= ?",
// 			end,
// 		)
// 	}

// 	if trange.StartDate.IsValid() {
// 		start := accounting_core.ParseDate(trange.StartDate.AsTime())
// 		query = query.Where("adb.day > ?",
// 			start,
// 		)
// 	}

// 	if pay.Sort != nil {
// 		var sorttype string
// 		switch pay.Sort.Type {
// 		case common.SortType_SORT_TYPE_ASC:
// 			sorttype = "asc"
// 		case common.SortType_SORT_TYPE_DESC:
// 			sorttype = "desc"
// 		default:
// 			sorttype = "desc"
// 		}

// 		query = query.
// 			Order(fmt.Sprintf("month %s", sorttype))

// 	} else {
// 		query = query.
// 			Order("month desc")
// 	}

// 	return query
// }
