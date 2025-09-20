//go:build wireinject
// +build wireinject

package main

import (
	"github.com/google/wire"
	"github.com/pdcgo/accounting_service"
	"github.com/pdcgo/shared/configs"
)

func InitializeMigration() (*Migration, error) {
	wire.Build(
		configs.NewProductionConfig,
		NewSetupClient,
		NewDatabase,
		accounting_service.NewMigrationHandler,
		NewMigration,
	)

	return &Migration{}, nil
}
