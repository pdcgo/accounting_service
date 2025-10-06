package report

import (
	"context"

	"connectrpc.com/connect"
	"github.com/pdcgo/schema/services/common/v1"
	"github.com/pdcgo/schema/services/report_iface/v1"
	"github.com/pdcgo/shared/pkg/debugtool"
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

	debugtool.LogJson(pay)

	return connect.NewResponse(&result), err
}

type dailyBalanceDetailViewImpl struct {
	tx  *gorm.DB
	db  *gorm.DB
	pay *report_iface.DailyBalanceDetailRequest
	// err error
}

func NewDailyBalanceDetailView(db *gorm.DB, pay *report_iface.DailyBalanceDetailRequest) *dailyBalanceDetailViewImpl {
	return &dailyBalanceDetailViewImpl{
		tx:  db,
		db:  db,
		pay: pay,
	}
}

func (d *dailyBalanceDetailViewImpl) Iterate(handle func(d *report_iface.DailyBalanceDetailItem) error) (*common.PageInfo, error) {
	panic("unimplemented")
}

func (d *dailyBalanceDetailViewImpl) dcQuery() *gorm.DB {
	panic("unimplemented")
}

func (d *dailyBalanceDetailViewImpl) lastBalanceQuery() *gorm.DB {
	panic("unimplemented")
}
