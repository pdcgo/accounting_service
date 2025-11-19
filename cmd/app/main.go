package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"

	cloudtasks "cloud.google.com/go/cloudtasks/apiv2"
	"connectrpc.com/connect"
	"github.com/pdcgo/accounting_service"
	"github.com/pdcgo/accounting_service/accounting_core"
	"github.com/pdcgo/schema/services/report_iface/v1/report_ifaceconnect"
	"github.com/pdcgo/shared/authorization"
	"github.com/pdcgo/shared/configs"
	"github.com/pdcgo/shared/custom_connect"
	"github.com/pdcgo/shared/db_connect"
	"github.com/pdcgo/shared/interfaces/authorization_iface"
	"github.com/pdcgo/shared/pkg/cloud_logging"
	"github.com/pdcgo/shared/pkg/ware_cache"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"gorm.io/gorm"
)

func NewCache(cfg *configs.AppConfig) (ware_cache.Cache, error) {
	return ware_cache.NewCustomCache(cfg.CacheService.Endpoint), nil
}

func NewCloudTaskClient() (*cloudtasks.Client, error) {
	return cloudtasks.NewClient(context.Background())
}

func NewAccountReportServiceClient(
	cfg *configs.AppConfig,
	defaultInterceptor custom_connect.DefaultClientInterceptor,
) report_ifaceconnect.AccountReportServiceClient {
	return report_ifaceconnect.NewAccountReportServiceClient(
		http.DefaultClient,
		cfg.AccountingService.Endpoint,
		connect.WithGRPC(),
		defaultInterceptor,
	)
}

func NewAuthorization(
	cfg *configs.AppConfig,
	db *gorm.DB,
	cache ware_cache.Cache,
) authorization_iface.Authorization {
	return authorization.NewAuthorization(cache, db, cfg.JwtSecret)
}

func NewDatabase(cfg *configs.AppConfig) (*gorm.DB, error) {
	return db_connect.NewProductionDatabase("accounting_service", &cfg.Database)
}

type App struct {
	Run func() error
}

func NewApp(
	mux *http.ServeMux,
	accountingRegister accounting_service.RegisterHandler,
	reportClient report_ifaceconnect.AccountReportServiceClient,
	// cache ware_cache.Cache
	// auth authorization_iface.Authorization,
) *App {
	return &App{
		Run: func() error {
			cancel, err := custom_connect.InitTracer("accounting-service")
			if err != nil {
				return err
			}

			defer cancel(context.Background())

			accounting_core.RegisterCustomHandler(
				"task_daily_update",
				accounting_core.NewDailyBalanceHandler(reportClient),
			)

			accountingRegister()

			port := os.Getenv("PORT")
			if port == "" {
				port = "8081"
			}

			host := os.Getenv("HOST")
			listen := fmt.Sprintf("%s:%s", host, port)
			log.Println("listening on", listen)

			http.ListenAndServe(
				listen,
				// Use h2c so we can serve HTTP/2 without TLS.
				h2c.NewHandler(
					custom_connect.WithCORS(mux),
					&http2.Server{}),
			)

			return nil
		},
	}
}

func main() {
	cloud_logging.SetCloudLoggingDefault()
	app, err := InitializeApp()
	if err != nil {
		panic(err)
	}

	err = app.Run()
	if err != nil {
		panic(err)
	}
}
