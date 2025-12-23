package revenue

import (
	"context"

	"connectrpc.com/connect"
	"github.com/pdcgo/accounting_service/report"
	"github.com/pdcgo/schema/services/revenue_iface/v1"
	"github.com/pdcgo/shared/configs"
	"github.com/pdcgo/shared/interfaces/authorization_iface"
	"gorm.io/gorm"
)

type revenueServiceImpl struct {
	db                      *gorm.DB
	auth                    authorization_iface.Authorization
	accountingServiceConfig *configs.AccountingService
	cfg                     *configs.DispatcherConfig
	dispatcher              report.ReportDispatcher
}

// RevenueAdjustment implements revenue_ifaceconnect.RevenueServiceHandler.
func (r *revenueServiceImpl) RevenueAdjustment(context.Context, *connect.Request[revenue_iface.RevenueAdjustmentRequest]) (*connect.Response[revenue_iface.RevenueAdjustmentResponse], error) {
	panic("unimplemented")
}

// OrderCompleted implements revenue_ifaceconnect.RevenueServiceHandler.
func (r *revenueServiceImpl) OrderCompleted(context.Context, *connect.Request[revenue_iface.OrderCompletedRequest]) (*connect.Response[revenue_iface.OrderCompletedResponse], error) {
	panic("unimplemented")
}

func NewRevenueService(
	db *gorm.DB,
	auth authorization_iface.Authorization,
	accountingServiceConfig *configs.AccountingService,
	cfg *configs.DispatcherConfig,
	dispatcher report.ReportDispatcher,
) *revenueServiceImpl {
	return &revenueServiceImpl{
		db:                      db,
		auth:                    auth,
		accountingServiceConfig: accountingServiceConfig,
		cfg:                     cfg,
		dispatcher:              dispatcher,
	}
}
