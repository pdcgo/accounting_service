//go:build wireinject
// +build wireinject

package main

import (
	"net/http"

	"github.com/google/wire"
	"github.com/pdcgo/accounting_service"
	"github.com/pdcgo/shared/configs"
)

func InitializeApp() (*App, error) {
	wire.Build(
		configs.NewProductionConfig,
		http.NewServeMux,
		NewCache,
		NewDatabase,
		NewAuthorization,
		accounting_service.NewRegister,
		NewApp,
	)

	return &App{}, nil
}
