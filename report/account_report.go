package report

import (
	"context"

	"connectrpc.com/connect"
	"github.com/pdcgo/schema/services/report_iface/v1"
	"github.com/pdcgo/shared/interfaces/authorization_iface"
	"github.com/pdcgo/shared/pkg/ware_cache"
	"gorm.io/gorm"
)

type accountReportImpl struct {
	db    *gorm.DB
	auth  authorization_iface.Authorization
	cache ware_cache.Cache
}

// MonthlyBalance implements report_ifaceconnect.AccountReportServiceHandler.
func (a *accountReportImpl) MonthlyBalance(context.Context, *connect.Request[report_iface.MonthlyBalanceRequest]) (*connect.Response[report_iface.MonthlyBalanceResponse], error) {
	panic("unimplemented")
}

// MonthlyBalanceDetail implements report_ifaceconnect.AccountReportServiceHandler.
func (a *accountReportImpl) MonthlyBalanceDetail(context.Context, *connect.Request[report_iface.MonthlyBalanceDetailRequest]) (*connect.Response[report_iface.MonthlyBalanceDetailResponse], error) {
	panic("unimplemented")
}

func NewAccountReportService(
	db *gorm.DB,
	auth authorization_iface.Authorization,
	cache ware_cache.Cache,
) *accountReportImpl {
	return &accountReportImpl{
		db:    db,
		auth:  auth,
		cache: cache,
	}
}
