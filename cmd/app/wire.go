//go:build wireinject
// +build wireinject

package main

import (
	"net/http"

	"github.com/google/wire"
	"github.com/pdcgo/accounting_service"
	"github.com/pdcgo/accounting_service/accounting_core"
	"github.com/pdcgo/shared/configs"
	"github.com/pdcgo/shared/custom_connect"
)

func InitializeApp() (*App, error) {
	wire.Build(
		configs.NewProductionConfig,
		http.NewServeMux,
		NewCache,
		NewDatabase,
		NewCloudTaskClient,
		accounting_core.NewAccountReportServiceClientDispatcher,
		custom_connect.NewDefaultInterceptor,
		NewAuthorization,
		accounting_service.NewRegister,
		NewApp,
	)

	return &App{}, nil
}
