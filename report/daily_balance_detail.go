package report

import (
	"context"

	"connectrpc.com/connect"
	"github.com/pdcgo/schema/services/common/v1"
	"github.com/pdcgo/schema/services/report_iface/v1"
	"gorm.io/gorm"
)

// DailyBalanceDetail implements report_ifaceconnect.AccountReportServiceHandler.
func (a *accountReportImpl) DailyBalanceDetail(
	ctx context.Context,
	req *connect.Request[report_iface.DailyBalanceDetailRequest],
) (*connect.Response[report_iface.DailyBalanceDetailResponse], error) {
	var err error
	result := report_iface.DailyBalanceDetailResponse{
		Data:     []*report_iface.DailyBalanceDetailItem{},
		PageInfo: &common.PageInfo{},
	}

	pay := req.Msg

	err = a.
		auth.
		AuthIdentityFromHeader(req.Header()).
		Err()

	if err != nil {
		return connect.NewResponse(&result), err
	}

	db := a.db.WithContext(ctx)
	query := dailyBalanceDetailQ(db, pay)

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

	for _, data := range result.Data {
		data.LabelFilterType = pay.LabelFilterType
	}

	var itemcount int64
	err = dailyBalanceDetailQ(db, pay).
		Select([]string{
			"adb.id",
		}).
		Count(&itemcount).
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

func dailyBalanceDetailQ(db *gorm.DB, pay *report_iface.DailyBalanceDetailRequest) *gorm.DB {
	var query *gorm.DB

	switch pay.LabelFilterType {
	case report_iface.LabelFilterType_LABEL_FILTER_TYPE_TEAM:
		query = db.
			Table("account_daily_balances adb").
			Joins("join accounts a on a.id = adb.account_id").
			Select([]string{
				"adb.account_id as label_id",
				"(EXTRACT(EPOCH FROM adb.day) * 1000000)::BIGINT as day",
				"adb.debit as debit",
				"adb.credit as credit",
				"adb.balance as balance",
			}).
			Where("adb.account_id = ?", pay.LabelId)

	case report_iface.LabelFilterType_LABEL_FILTER_TYPE_SHOP:
		query = db.
			Table("shop_daily_balances adb").
			Joins("join accounts a on a.id = adb.account_id").
			Select([]string{
				"adb.shop_id as label_id",
				"(EXTRACT(EPOCH FROM adb.day) * 1000000)::BIGINT as day",
				"adb.debit as debit",
				"adb.credit as credit",
				"adb.balance as balance",
			}).
			Where("adb.shop_id = ?", pay.LabelId)

	case report_iface.LabelFilterType_LABEL_FILTER_TYPE_SUPPLIER:
		query = db.
			Table("supplier_daily_balances adb").
			Joins("join accounts a on a.id = adb.account_id").
			Select([]string{
				"adb.supplier_id as label_id",
				"(EXTRACT(EPOCH FROM adb.day) * 1000000)::BIGINT as day",
				"adb.debit as debit",
				"adb.credit as credit",
				"adb.balance as balance",
			}).
			Where("adb.supplier_id = ?", pay.LabelId)

	case report_iface.LabelFilterType_LABEL_FILTER_TYPE_CUSTOM:
		query = db.
			Table("custom_label_daily_balances adb").
			Joins("join accounts a on a.id = adb.account_id").
			Select([]string{
				"adb.custom_id as label_id",
				"(EXTRACT(EPOCH FROM adb.day) * 1000000)::BIGINT as day",
				"adb.debit as debit",
				"adb.credit as credit",
				"adb.balance as balance",
			}).
			Where("adb.custom_id = ?", pay.LabelId)

	case report_iface.LabelFilterType_LABEL_FILTER_TYPE_CS:
		query = query.
			Table("cs_daily_balances adb").
			Joins("join accounts a on a.id = adb.account_id").
			Select([]string{
				"adb.cs_id as label_id",
				"(EXTRACT(EPOCH FROM adb.day) * 1000000)::BIGINT as day",
				"adb.debit as debit",
				"adb.credit as credit",
				"adb.balance as balance",
			}).
			Where("adb.cs_id = ?", pay.LabelId)
	}

	query = query.
		Where("adb.journal_team_id = ?", pay.TeamId).
		Where("a.account_key = ?", pay.AccountKey)

	trange := pay.TimeRange
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

	return query.
		Order("adb.day desc")
}
