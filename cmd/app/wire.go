//go:build wireinject
// +build wireinject

package main

import (
	"net/http"

	"github.com/google/wire"
	"github.com/pdcgo/accounting_service"
	"github.com/pdcgo/shared/configs"
	"github.com/pdcgo/shared/custom_connect"
)

func InitializeApp() (*App, error) {
	wire.Build(
		configs.NewProductionConfig,
		http.NewServeMux,
		NewCache,
		NewDatabase,
		custom_connect.NewDefaultInterceptor,
		NewAuthorization,
		accounting_service.NewRegister,
		NewApp,
	)

	return &App{}, nil
}
