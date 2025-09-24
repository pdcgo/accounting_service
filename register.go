package accounting_service

import (
	"context"
	"log"
	"log/slog"
	"net/http"

	"connectrpc.com/connect"
	"connectrpc.com/validate"
	"github.com/pdcgo/accounting_service/accounting_core"
	"github.com/pdcgo/accounting_service/accounting_model"
	"github.com/pdcgo/accounting_service/adjustment"
	"github.com/pdcgo/accounting_service/common"
	"github.com/pdcgo/accounting_service/ledger"
	"github.com/pdcgo/accounting_service/payment"
	"github.com/pdcgo/accounting_service/report"
	"github.com/pdcgo/accounting_service/revenue"
	"github.com/pdcgo/accounting_service/stock"
	"github.com/pdcgo/schema/services/accounting_iface/v1/accounting_ifaceconnect"
	"github.com/pdcgo/schema/services/common/v1/commonconnect"
	"github.com/pdcgo/schema/services/payment_iface/v1/payment_ifaceconnect"
	"github.com/pdcgo/schema/services/report_iface/v1/report_ifaceconnect"
	"github.com/pdcgo/schema/services/revenue_iface/v1/revenue_ifaceconnect"
	"github.com/pdcgo/schema/services/stock_iface/v1/stock_ifaceconnect"
	"github.com/pdcgo/shared/interfaces/authorization_iface"
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
) RegisterHandler {

	return func() {
		// validator
		interceptor, err := validate.NewInterceptor()
		if err != nil {
			log.Fatal(err)
		}

		logger := connect.WithInterceptors(&LoggingInterceptor{})

		validator := connect.WithInterceptors(interceptor)
		path, handler := accounting_ifaceconnect.NewAccountServiceHandler(NewAccountService(db, auth), validator, logger)
		mux.Handle(path, handler)
		path, handler = accounting_ifaceconnect.NewExpenseServiceHandler(NewExpenseService(db, auth), validator, logger)
		mux.Handle(path, handler)
		path, handler = commonconnect.NewUserServiceHandler(NewUserService(db), validator)
		mux.Handle(path, handler)
		path, handler = accounting_ifaceconnect.NewAccountingSetupServiceHandler(NewSetupService(db), validator, logger)
		mux.Handle(path, handler)
		path, handler = accounting_ifaceconnect.NewLedgerServiceHandler(ledger.NewLedgerService(db, auth), validator, logger)
		mux.Handle(path, handler)
		path, handler = revenue_ifaceconnect.NewRevenueServiceHandler(revenue.NewRevenueService(db, auth), validator, logger)
		mux.Handle(path, handler)
		path, handler = stock_ifaceconnect.NewStockServiceHandler(stock.NewStockService(db, auth), validator, logger)
		mux.Handle(path, handler)
		path, handler = report_ifaceconnect.NewAccountReportServiceHandler(report.NewAccountReportService(db, auth), validator, logger)
		mux.Handle(path, handler)
		path, handler = payment_ifaceconnect.NewPaymentServiceHandler(payment.NewPaymentService(db, auth), validator, logger)
		mux.Handle(path, handler)
		path, handler = accounting_ifaceconnect.NewAdjustmentServiceHandler(adjustment.NewAdjustmentService(db, auth), validator, logger)
		mux.Handle(path, handler)

		//  bagian common
		path, handler = commonconnect.NewTeamServiceHandler(common.NewTeamService(db), validator, logger)
		mux.Handle(path, handler)

	}

}

// LoggingInterceptor logs errors from RPC calls
type LoggingInterceptor struct{}

// WrapUnary satisfies connect.Interceptor
func (l *LoggingInterceptor) WrapUnary(next connect.UnaryFunc) connect.UnaryFunc {
	return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
		res, err := next(ctx, req)
		if err != nil {
			if cErr, ok := err.(*connect.Error); ok {
				slog.Error("rpc error",
					"procedure", req.Spec().Procedure,
					"code", cErr.Code().String(),
					"msg", cErr.Message(),
				)
			} else {
				slog.Error("request_error",
					"procedure", req.Spec().Procedure,
					"error", err,
					"message", err.Error(),
					"token", req.Header().Get("Authorization"),
					slog.Any("payload", req.Any()),
				)
			}
		}
		return res, err
	}
}

// WrapStreamingClient (optional: if you want streaming client logs)
func (l *LoggingInterceptor) WrapStreamingClient(next connect.StreamingClientFunc) connect.StreamingClientFunc {
	return next
}

// WrapStreamingHandler (optional: if you want streaming server logs)
func (l *LoggingInterceptor) WrapStreamingHandler(next connect.StreamingHandlerFunc) connect.StreamingHandlerFunc {
	return next
}
