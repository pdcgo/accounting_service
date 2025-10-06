package report

import (
	"context"

	"connectrpc.com/connect"
	"github.com/pdcgo/schema/services/common/v1"
	"github.com/pdcgo/schema/services/report_iface/v1"
	"gorm.io/gorm"
)

// BalanceDetail implements report_ifaceconnect.AccountReportServiceHandler.
func (a *accountReportImpl) BalanceDetail(
	ctx context.Context,
	req *connect.Request[report_iface.BalanceDetailRequest],
) (*connect.Response[report_iface.BalanceDetailResponse], error) {
	var err error

	result := report_iface.BalanceDetailResponse{
		Data:     []*report_iface.BalanceDetailItem{},
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
	view := NewBalanceDetailView(db, pay)
	result.PageInfo, err = view.Iterate(func(d *report_iface.BalanceDetailItem) error {
		d.LabelFilterType = pay.LabelFilterType
		result.Data = append(result.Data, d)
		return nil
	})

	return connect.NewResponse(&result), err
}

type balanceDetailViewImpl struct {
	tx  *gorm.DB
	db  *gorm.DB
	pay *report_iface.BalanceDetailRequest
	// err error
}

func (b *balanceDetailViewImpl) Iterate(handle func(d *report_iface.BalanceDetailItem) error) (*common.PageInfo, error) {
	var err error

	baseQuery := b.
		db.
		Table("(?) as base", b.dcQuery())

	page := common.PageInfo{}
	err = baseQuery.Count(&page.TotalItems).Error
	if err != nil {
		return &page, err
	}

	offset := (b.pay.Page.Page - 1) * b.pay.Page.Limit
	page.TotalPage = int64(page.TotalItems / b.pay.Page.Limit)
	page.CurrentPage = b.pay.Page.Page

	baseQuery = b.dcQuery()
	lbQuery := b.lastBalanceQuery()
	query := b.
		db.
		Table("(?) as base", baseQuery).
		Joins("full outer join (?) as bal on bal.label_id = base.label_id", lbQuery).
		Offset(int(offset)).
		Limit(int(b.pay.Page.Limit))

	rows, err := query.Rows()

	if err != nil {
		return &page, err
	}

	for rows.Next() {
		var d report_iface.BalanceDetailItem
		err = b.db.ScanRows(rows, &d)
		if err != nil {
			return &page, err
		}

		err = handle(&d)
		if err != nil {
			return &page, err
		}
	}

	return &page, nil
}

func (b *balanceDetailViewImpl) dcQuery() *gorm.DB {
	pay := b.pay
	query := b.
		db

	switch pay.LabelFilterType {
	case report_iface.LabelFilterType_LABEL_FILTER_TYPE_TEAM:
		query = query.
			Table("account_daily_balances adb").
			Joins("join accounts a on a.id = adb.account_id").
			Select([]string{
				"a.id as label_id",
				"sum(adb.debit) as debit",
				"sum(adb.credit) as credit",
			})

	case report_iface.LabelFilterType_LABEL_FILTER_TYPE_SHOP:
		query = query.
			Table("shop_daily_balances adb").
			Joins("join accounts a on a.id = adb.account_id").
			Select([]string{
				"adb.shop_id as label_id",
				"sum(adb.debit) as debit",
				"sum(adb.credit) as credit",
			})
	case report_iface.LabelFilterType_LABEL_FILTER_TYPE_SUPPLIER:
		query = query.
			Table("supplier_daily_balances adb").
			Joins("join accounts a on a.id = adb.account_id").
			Select([]string{
				"adb.supplier_id as label_id",
				"sum(adb.debit) as debit",
				"sum(adb.credit) as credit",
			})

	case report_iface.LabelFilterType_LABEL_FILTER_TYPE_CUSTOM:
		query = query.
			Table("custom_label_daily_balances adb").
			Joins("join accounts a on a.id = adb.account_id").
			Select([]string{
				"adb.custom_id as label_id",
				"sum(adb.debit) as debit",
				"sum(adb.credit) as credit",
			})

	case report_iface.LabelFilterType_LABEL_FILTER_TYPE_CS:
		query = query.
			Table("cs_daily_balances adb").
			Joins("join accounts a on a.id = adb.account_id").
			Select([]string{
				"adb.cs_id as label_id",
				"sum(adb.debit) as debit",
				"sum(adb.credit) as credit",
			})

	}

	query = query.
		Where("adb.journal_team_id = ?", pay.TeamId).
		Where("a.account_key = ?", pay.AccountKey)

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

	query = query.
		Group("label_id")

	return query

}

// query untuk getting last balance
func (b *balanceDetailViewImpl) lastBalanceQuery() *gorm.DB {
	pay := b.pay

	query := b.
		db

	switch pay.LabelFilterType {
	case report_iface.LabelFilterType_LABEL_FILTER_TYPE_TEAM:
		query = query.
			Table("account_daily_balances adb").
			Joins("join accounts a on a.id = adb.account_id").
			Select([]string{
				"a.id as label_id",
				"adb.day",
				"adb.balance",
			})

	case report_iface.LabelFilterType_LABEL_FILTER_TYPE_SHOP:
		query = query.
			Table("shop_daily_balances adb").
			Joins("join accounts a on a.id = adb.account_id").
			Select([]string{
				"adb.shop_id as label_id",
				"adb.day",
				"adb.balance",
			})
	case report_iface.LabelFilterType_LABEL_FILTER_TYPE_SUPPLIER:
		query = query.
			Table("supplier_daily_balances adb").
			Joins("join accounts a on a.id = adb.account_id").
			Select([]string{
				"adb.supplier_id as label_id",
				"adb.day",
				"adb.balance",
			})

	case report_iface.LabelFilterType_LABEL_FILTER_TYPE_CUSTOM:
		query = query.
			Table("custom_label_daily_balances adb").
			Joins("join accounts a on a.id = adb.account_id").
			Select([]string{
				"adb.custom_id as label_id",
				"adb.day",
				"adb.balance",
			})

	case report_iface.LabelFilterType_LABEL_FILTER_TYPE_CS:
		query = query.
			Table("cs_daily_balances adb").
			Joins("join accounts a on a.id = adb.account_id").
			Select([]string{
				"adb.cs_id as label_id",
				"adb.day",
				"adb.balance",
			})

	}

	query = query.
		Where("a.account_key = ?", pay.AccountKey).
		Where("adb.journal_team_id = ?", pay.TeamId)

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

	query = query.
		Order("label_id, adb.day desc")

	query = b.
		db.
		Table("(?) as daygroup", query).
		Select([]string{
			"distinct on (daygroup.label_id) daygroup.label_id",
			"daygroup.balance",
		})

	return query
}

func NewBalanceDetailView(db *gorm.DB, pay *report_iface.BalanceDetailRequest) *balanceDetailViewImpl {
	return &balanceDetailViewImpl{
		tx:  db,
		db:  db,
		pay: pay,
	}
}
