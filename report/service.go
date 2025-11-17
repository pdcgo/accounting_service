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
	cfg       *configs.DispatcherConfig
	accConfig *configs.AccountingService
	db        *gorm.DB
	auth      authorization_iface.Authorization
	cache     ware_cache.Cache
	dispather ReportDispatcher
}

// DailyUpdateBalanceAsync implements report_ifaceconnect.AccountReportServiceHandler.
func (a *accountReportImpl) DailyUpdateBalanceAsync(context.Context, *connect.Request[report_iface.DailyUpdateBalanceAsyncRequest]) (*connect.Response[report_iface.DailyUpdateBalanceAsyncResponse], error) {
	panic("unimplemented")
}

func NewAccountReportService(

	cfg *configs.DispatcherConfig,
	accConfig *configs.AccountingService,
	db *gorm.DB,
	auth authorization_iface.Authorization,
	cache ware_cache.Cache,
	dispather ReportDispatcher,
) *accountReportImpl {
	return &accountReportImpl{
		cfg:       cfg,
		accConfig: accConfig,
		db:        db,
		auth:      auth,
		cache:     cache,
		dispather: dispather,
	}
}

type ReportDispatcher func(ctx context.Context, req *cloudtaskspb.CreateTaskRequest, opts ...gax.CallOption) error
