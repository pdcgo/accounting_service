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

// BalanceDetail implements report_ifaceconnect.AccountReportServiceHandler.
func (a *accountReportImpl) BalanceDetail(context.Context, *connect.Request[report_iface.BalanceDetailRequest]) (*connect.Response[report_iface.BalanceDetailResponse], error) {
	panic("unimplemented")
}

// DailyBalanceDetail implements report_ifaceconnect.AccountReportServiceHandler.
func (a *accountReportImpl) DailyBalanceDetail(context.Context, *connect.Request[report_iface.DailyBalanceDetailRequest]) (*connect.Response[report_iface.DailyBalanceDetailResponse], error) {
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
