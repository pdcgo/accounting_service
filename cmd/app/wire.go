//go:build wireinject
// +build wireinject

package main

import (
	"net/http"

	"github.com/google/wire"
	"github.com/pdcgo/accounting_service"
	"github.com/pdcgo/accounting_service/report"
	"github.com/pdcgo/shared/configs"
	"github.com/pdcgo/shared/custom_connect"
)

func InitializeApp() (*App, error) {
	wire.Build(
		configs.NewProductionConfig,
		custom_connect.NewDefaultClientInterceptor,
		http.NewServeMux,
		NewCache,
		NewDatabase,
		NewCloudTaskClient,
		report.NewCloudTaskReportDispatcher,
		NewAccountReportServiceClient,
		// accounting_core.NewAccountReportServiceClientDispatcher,
		custom_connect.NewDefaultInterceptor,
		NewAuthorization,
		accounting_service.NewRegister,
		custom_connect.NewRegisterReflect,
		NewApp,
	)

	return &App{}, nil
}
