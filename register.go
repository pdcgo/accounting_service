package accounting_service

import (
	"log"
	"net/http"

	"github.com/pdcgo/accounting_service/accounting_core"
	"github.com/pdcgo/accounting_service/accounting_model"
	"github.com/pdcgo/accounting_service/adjustment"
	"github.com/pdcgo/accounting_service/ads_expense"
	"github.com/pdcgo/accounting_service/core"
	"github.com/pdcgo/accounting_service/ledger"
	"github.com/pdcgo/accounting_service/payment"
	"github.com/pdcgo/accounting_service/report"
	"github.com/pdcgo/accounting_service/revenue"
	"github.com/pdcgo/accounting_service/setup"
	"github.com/pdcgo/accounting_service/stock"
	"github.com/pdcgo/accounting_service/tag"
	"github.com/pdcgo/schema/services/accounting_iface/v1/accounting_ifaceconnect"
	"github.com/pdcgo/schema/services/common/v1/commonconnect"
	"github.com/pdcgo/schema/services/payment_iface/v1/payment_ifaceconnect"
	"github.com/pdcgo/schema/services/report_iface/v1/report_ifaceconnect"
	"github.com/pdcgo/schema/services/revenue_iface/v1/revenue_ifaceconnect"
	"github.com/pdcgo/schema/services/stock_iface/v1/stock_ifaceconnect"
	"github.com/pdcgo/shared/custom_connect"
	"github.com/pdcgo/shared/interfaces/authorization_iface"
	"github.com/pdcgo/shared/pkg/ware_cache"
	"gorm.io/gorm"
)

type MigrationHandler func() error

func NewMigrationHandler(
	db *gorm.DB,
) MigrationHandler {
	return func() error {
		log.Println("migrating account service")
		return db.AutoMigrate(
			&accounting_core.Account{},
			&accounting_core.JournalEntry{},
			&accounting_core.Transaction{},
			&accounting_core.Label{},
			&accounting_core.TransactionLabel{},
			&accounting_core.AccountDailyBalance{},
			&accounting_core.TransactionShop{},
			&accounting_core.TransactionSupplier{},
			&accounting_core.TransactionCustomerService{},
			&accounting_core.AccountingTag{},
			&accounting_core.TransactionTag{},
			&accounting_core.CsDailyBalance{},
			&accounting_core.ShopDailyBalance{},
			&accounting_core.SupplierDailyBalance{},
			&accounting_core.CustomLabelDailyBalance{},

			&accounting_model.BankAccountV2{},
			&accounting_model.BankAccountLabel{},
			&accounting_model.BankAccountLabelRelation{},
			&accounting_model.BankTransferHistory{},
			&accounting_model.Expense{},
			&accounting_model.Payment{},
		)
	}
}

type RegisterHandler func()

func NewRegister(
	db *gorm.DB,
	auth authorization_iface.Authorization,
	mux *http.ServeMux,
	defaultInterceptor custom_connect.DefaultInterceptor,
	cache ware_cache.Cache,
) RegisterHandler {

	return func() {

		path, handler := accounting_ifaceconnect.NewAccountServiceHandler(NewAccountService(db, auth), defaultInterceptor)
		mux.Handle(path, handler)
		path, handler = accounting_ifaceconnect.NewExpenseServiceHandler(NewExpenseService(db, auth), defaultInterceptor)
		mux.Handle(path, handler)
		path, handler = commonconnect.NewUserServiceHandler(NewUserService(db), defaultInterceptor)
		mux.Handle(path, handler)
		path, handler = accounting_ifaceconnect.NewAccountingSetupServiceHandler(setup.NewSetupService(db), defaultInterceptor)
		mux.Handle(path, handler)
		path, handler = accounting_ifaceconnect.NewLedgerServiceHandler(ledger.NewLedgerService(db, auth), defaultInterceptor)
		mux.Handle(path, handler)
		path, handler = revenue_ifaceconnect.NewRevenueServiceHandler(revenue.NewRevenueService(db, auth), defaultInterceptor)
		mux.Handle(path, handler)
		path, handler = stock_ifaceconnect.NewStockServiceHandler(stock.NewStockService(db, auth), defaultInterceptor)
		mux.Handle(path, handler)
		path, handler = report_ifaceconnect.NewAccountReportServiceHandler(report.NewAccountReportService(db, auth, cache), defaultInterceptor)
		mux.Handle(path, handler)
		path, handler = payment_ifaceconnect.NewPaymentServiceHandler(payment.NewPaymentService(db, auth), defaultInterceptor)
		mux.Handle(path, handler)
		path, handler = accounting_ifaceconnect.NewAdjustmentServiceHandler(adjustment.NewAdjustmentService(db, auth), defaultInterceptor)
		mux.Handle(path, handler)
		path, handler = accounting_ifaceconnect.NewCoreServiceHandler(core.NewCoreService(db, auth), defaultInterceptor)
		mux.Handle(path, handler)
		path, handler = accounting_ifaceconnect.NewAdsExpenseServiceHandler(ads_expense.NewAdsExpenseService(db, auth), defaultInterceptor)
		mux.Handle(path, handler)
		path, handler = accounting_ifaceconnect.NewTagServiceHandler(tag.NewTagService(db, auth), defaultInterceptor)
		mux.Handle(path, handler)
	}

}
