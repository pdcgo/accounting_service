package accounting_service

import (
	"log"
	"log/slog"
	"net/http"

	"connectrpc.com/connect"
	"github.com/pdcgo/accounting_service/accounting_core"
	"github.com/pdcgo/accounting_service/accounting_model"
	"github.com/pdcgo/accounting_service/adjustment"
	"github.com/pdcgo/accounting_service/ads_expense"
	"github.com/pdcgo/accounting_service/core"
	"github.com/pdcgo/accounting_service/expense"
	"github.com/pdcgo/accounting_service/ledger"
	"github.com/pdcgo/accounting_service/payment"
	"github.com/pdcgo/accounting_service/report"
	"github.com/pdcgo/accounting_service/revenue"
	"github.com/pdcgo/accounting_service/setup"
	"github.com/pdcgo/accounting_service/stock"
	"github.com/pdcgo/accounting_service/tag"
	"github.com/pdcgo/accounting_service/transfer"
	"github.com/pdcgo/schema/services/accounting_iface/v1"
	"github.com/pdcgo/schema/services/accounting_iface/v1/accounting_ifaceconnect"
	"github.com/pdcgo/schema/services/common/v1"
	"github.com/pdcgo/schema/services/payment_iface/v1/payment_ifaceconnect"
	"github.com/pdcgo/schema/services/report_iface/v1/report_ifaceconnect"
	"github.com/pdcgo/schema/services/revenue_iface/v1/revenue_ifaceconnect"
	"github.com/pdcgo/schema/services/stock_iface/v1"
	"github.com/pdcgo/schema/services/stock_iface/v1/stock_ifaceconnect"
	"github.com/pdcgo/shared/custom_connect"
	"github.com/pdcgo/shared/interfaces/authorization_iface"
	"github.com/pdcgo/shared/pkg/ware_cache"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type SeedHandler func() error

func NewSeedHandler(
	db *gorm.DB,
) SeedHandler {
	return func() error {
		log.Println("seeding account service")
		// domain := authorization.NewDomainV2(db, authorization.RootDomain)
		// domain.RoleAddPermission()
		return nil
	}
}

type MigrationHandler func() error

func NewMigrationHandler(
	db *gorm.DB,
) MigrationHandler {
	return func() error {
		log.Println("migrating account service")
		err := db.AutoMigrate(
			&accounting_core.Account{},
			&accounting_core.JournalEntry{},
			&accounting_core.Transaction{},
			&accounting_core.AccountDailyBalance{},
			&accounting_core.AccountKeyDailyBalance{},
			&accounting_core.TransactionShop{},
			&accounting_core.TransactionSupplier{},
			&accounting_core.TransactionCustomerService{},
			&accounting_core.AccountingTag{},
			&accounting_core.TransactionTag{},
			&accounting_core.CsDailyBalance{},
			&accounting_core.ShopDailyBalance{},
			&accounting_core.SupplierDailyBalance{},
			&accounting_core.CustomLabelDailyBalance{},
			&accounting_core.TypeLabel{},
			&accounting_core.TransactionTypeLabel{},
			&accounting_core.TypeLabelDailyBalance{},

			&accounting_model.BankAccountV2{},
			&accounting_model.BankAccountLabel{},
			&accounting_model.BankAccountLabelRelation{},
			&accounting_model.BankTransferHistory{},
			&accounting_model.Expense{},
			&accounting_model.Payment{},
		)
		if err != nil {
			return err

		}

		// adding type label
		mplabels := []common.MarketplaceType{
			common.MarketplaceType_MARKETPLACE_TYPE_CUSTOM,
			common.MarketplaceType_MARKETPLACE_TYPE_LAZADA,
			common.MarketplaceType_MARKETPLACE_TYPE_MENGANTAR,
			common.MarketplaceType_MARKETPLACE_TYPE_TOKOPEDIA,
			common.MarketplaceType_MARKETPLACE_TYPE_TIKTOK,
			common.MarketplaceType_MARKETPLACE_TYPE_SHOPEE,
		}

		for _, mp := range mplabels {
			mplabel := &accounting_core.TypeLabel{
				Key:   accounting_iface.LabelKey_LABEL_KEY_MARKETPLACE,
				Label: common.MarketplaceType_name[int32(mp)],
			}

			err = db.
				Clauses(clause.OnConflict{DoNothing: true}).
				Create(&mplabel).Error

			if err != nil {
				slog.Error(err.Error())
			}

			// 	err = db.
			//  Model().
		}

		inboundsources := []stock_iface.InboundSource{
			stock_iface.InboundSource_INBOUND_SOURCE_RESTOCK,
			stock_iface.InboundSource_INBOUND_SOURCE_RETURN,
			stock_iface.InboundSource_INBOUND_SOURCE_TRANSFER,
		}

		for _, src := range inboundsources {
			srco := &accounting_core.TypeLabel{
				Key:   accounting_iface.LabelKey_LABEL_KEY_WAREHOUSE_TRANSACTION_TYPE,
				Label: stock_iface.InboundSource_name[int32(src)],
			}

			err = db.
				Clauses(clause.OnConflict{DoNothing: true}).
				Create(&srco).Error

			if err != nil {
				slog.Error(err.Error())
			}
		}

		return nil
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
		sourceInterceptor := connect.WithInterceptors(&custom_connect.RequestSourceIntercept{})

		path, handler := accounting_ifaceconnect.NewAccountServiceHandler(NewAccountService(db, auth), defaultInterceptor)
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
		path, handler = report_ifaceconnect.NewAccountReportServiceHandler(
			report.NewAccountReportService(db, auth, cache),
			defaultInterceptor)
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
	}

}
