package accounting_service

import (
	"net/http"

	"connectrpc.com/connect"
	"github.com/pdcgo/accounting_service/account"
	"github.com/pdcgo/accounting_service/adjustment"
	"github.com/pdcgo/accounting_service/ads_expense"
	"github.com/pdcgo/accounting_service/core"
	"github.com/pdcgo/accounting_service/expense"
	"github.com/pdcgo/accounting_service/ledger"
	"github.com/pdcgo/accounting_service/payment"
	"github.com/pdcgo/accounting_service/report"
	"github.com/pdcgo/accounting_service/report/report_balance"
	"github.com/pdcgo/accounting_service/revenue"
	"github.com/pdcgo/accounting_service/setup"
	"github.com/pdcgo/accounting_service/statement"
	"github.com/pdcgo/accounting_service/stock"
	"github.com/pdcgo/accounting_service/tag"
	"github.com/pdcgo/accounting_service/transfer"
	"github.com/pdcgo/schema/services/accounting_iface/v1/accounting_ifaceconnect"
	"github.com/pdcgo/schema/services/payment_iface/v1/payment_ifaceconnect"
	"github.com/pdcgo/schema/services/report_iface/v1/report_ifaceconnect"
	"github.com/pdcgo/schema/services/revenue_iface/v1/revenue_ifaceconnect"
	"github.com/pdcgo/schema/services/stock_iface/v1/stock_ifaceconnect"
	"github.com/pdcgo/shared/configs"
	"github.com/pdcgo/shared/custom_connect"
	"github.com/pdcgo/shared/interfaces/authorization_iface"
	"github.com/pdcgo/shared/pkg/ware_cache"
	"gorm.io/gorm"
)

type RegisterHandler func()

func NewRegister(
	cfg *configs.AppConfig,
	db *gorm.DB,
	auth authorization_iface.Authorization,
	mux *http.ServeMux,
	defaultInterceptor custom_connect.DefaultInterceptor,
	cache ware_cache.Cache,
	dispather report.ReportDispatcher,
) RegisterHandler {

	return func() {

		sourceInterceptor := connect.WithInterceptors(&custom_connect.RequestSourceIntercept{})

		path, handler := accounting_ifaceconnect.NewAccountServiceHandler(
			account.NewAccountService(db, auth),
			defaultInterceptor,
			sourceInterceptor,
		)
		mux.Handle(path, handler)

		path, handler = accounting_ifaceconnect.NewExpenseServiceHandler(
			expense.NewExpenseService(db, auth),
			defaultInterceptor,
			sourceInterceptor,
		)
		mux.Handle(path, handler)

		path, handler = accounting_ifaceconnect.NewAccountingSetupServiceHandler(setup.NewSetupService(db), defaultInterceptor)
		mux.Handle(path, handler)
		path, handler = accounting_ifaceconnect.NewLedgerServiceHandler(
			ledger.NewLedgerService(db, auth),
			defaultInterceptor,
			sourceInterceptor,
		)
		mux.Handle(path, handler)
		path, handler = revenue_ifaceconnect.NewRevenueServiceHandler(revenue.NewRevenueService(db, auth), defaultInterceptor)
		mux.Handle(path, handler)
		path, handler = stock_ifaceconnect.NewStockServiceHandler(stock.NewStockService(db, auth), defaultInterceptor)
		mux.Handle(path, handler)

		// report
		path, handler = report_ifaceconnect.NewAccountReportServiceHandler(
			report.NewAccountReportService(
				&cfg.DispatcherConfig,
				&cfg.AccountingService,
				db,
				auth,
				cache,
				dispather),
			defaultInterceptor)
		mux.Handle(path, handler)
		path, handler = report_ifaceconnect.NewBalanceServiceHandler(
			report_balance.NewBalanceService(db, auth),
			defaultInterceptor,
			sourceInterceptor,
		)
		mux.Handle(path, handler)
		path, handler = payment_ifaceconnect.NewPaymentServiceHandler(payment.NewPaymentService(db, auth), defaultInterceptor)
		mux.Handle(path, handler)
		path, handler = accounting_ifaceconnect.NewAdjustmentServiceHandler(adjustment.NewAdjustmentService(db, auth), defaultInterceptor)
		mux.Handle(path, handler)
		path, handler = accounting_ifaceconnect.NewCoreServiceHandler(core.NewCoreService(db, auth), defaultInterceptor)
		mux.Handle(path, handler)
		path, handler = accounting_ifaceconnect.NewAdsExpenseServiceHandler(
			ads_expense.NewAdsExpenseService(db, auth),
			defaultInterceptor,
			sourceInterceptor,
		)
		mux.Handle(path, handler)

		path, handler = accounting_ifaceconnect.NewTagServiceHandler(tag.NewTagService(db, auth), defaultInterceptor)
		mux.Handle(path, handler)
		path, handler = accounting_ifaceconnect.NewTransferServiceHandler(transfer.NewTransferService(db, auth), defaultInterceptor)
		mux.Handle(path, handler)
		var ledgerClient accounting_ifaceconnect.LedgerServiceClient
		path, handler = accounting_ifaceconnect.NewStatementServiceHandler(
			statement.NewStatementService(ledgerClient),
			defaultInterceptor,
			sourceInterceptor,
		)
		mux.Handle(path, handler)
	}

}
