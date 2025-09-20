package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/pdcgo/accounting_service"
	"github.com/pdcgo/shared/authorization"
	"github.com/pdcgo/shared/configs"
	"github.com/pdcgo/shared/db_connect"
	"github.com/pdcgo/shared/interfaces/authorization_iface"
	"github.com/pdcgo/shared/pkg/cloud_logging"
	"github.com/pdcgo/shared/pkg/ware_cache"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"gorm.io/gorm"
)

func NewCache() (ware_cache.Cache, error) {
	return ware_cache.NewBadgerCache("/tmp/cache")
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

func withCors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		w.Header().Set("Access-Control-Allow-Headers", "Connect-Protocol-Version, Referer, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, Accept, Origin, Cache-Control, X-Requested-With")
		w.Header().Set("Access-Control-Allow-Methods", "HEAD,PATCH,OPTIONS,GET,POST,PUT,DELETE")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

type App struct {
	Run func() error
}

func NewApp(
	mux *http.ServeMux,
	accountingRegister accounting_service.RegisterHandler,
	// auth authorization_iface.Authorization,
) *App {
	return &App{
		Run: func() error {

			// identity := auth.
			// 	AuthIdentityFromToken("eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJVc2VySUQiOjEyOCwiU3VwZXJVc2VyIjpmYWxzZSwiVmFsaWRVbnRpbCI6MTc1ODM0MjQ1MjM1Njk4OCwiRnJvbSI6InNlbGxpbmciLCJVc2VyQWdlbnQiOiIiLCJDcmVhdGVkQXQiOjE3NTgyNTYwNTIzNTY5ODh9.AtuZ_P25vHSW-JihBrhF2vXOg5Hl0vQMJR6LrK1XsEs")
			// err := identity.
			// 	HasPermission(authorization_iface.CheckPermissionGroup{
			// 		&db_models.InvTransaction{}: &authorization_iface.CheckPermission{
			// 			DomainID: 47,
			// 			Actions:  []authorization_iface.Action{authorization_iface.Create},
			// 		},
			// 	}).Err()

			// if err != nil {
			// 	var perm *authorization_iface.PermissionError
			// 	if errors.As(err, &perm) {

			// 	}
			// 	debugtool.LogJson(err)
			// 	log.Println(err)

			// 	agent := identity.Identity()
			// 	debugtool.LogJson(agent.GetUserID(), agent)
			// }

			// return nil
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
					withCors(mux),
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
