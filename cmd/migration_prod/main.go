package main

import (
	"context"
	"log"
	"log/slog"
	"net/http"

	"connectrpc.com/connect"
	"github.com/pdcgo/accounting_service"
	"github.com/pdcgo/schema/services/accounting_iface/v1"
	"github.com/pdcgo/schema/services/accounting_iface/v1/accounting_ifaceconnect"
	"github.com/pdcgo/shared/configs"
	"github.com/pdcgo/shared/db_connect"
	"github.com/pdcgo/shared/db_models"
	"gorm.io/gorm"
)

func NewDatabase(cfg *configs.AppConfig) (*gorm.DB, error) {
	return db_connect.NewProductionDatabase("accounting_migration", &cfg.Database)
}

func NewSetupClient(
	cfg *configs.AppConfig,
) accounting_ifaceconnect.AccountingSetupServiceClient {
	acCfg := cfg.AccountingService
	log.Println("accounting service endpoint", acCfg.Endpoint)
	return accounting_ifaceconnect.NewAccountingSetupServiceClient(
		http.DefaultClient,
		acCfg.Endpoint,
		// "http://localhost:8081",
		connect.WithGRPC())
}

type Migration struct {
	Run func() error
}

func NewMigration(
	db *gorm.DB,
	setup accounting_ifaceconnect.AccountingSetupServiceClient,
	migrate accounting_service.MigrationHandler,

) *Migration {
	return &Migration{
		Run: func() error {
			var err error

			ctx := context.Background()

			// err = migrate()
			// if err != nil {
			// 	return err
			// }

			// stream, err := setup.Setup(ctx, &connect.Request[accounting_iface.SetupRequest]{
			// 	Msg: &accounting_iface.SetupRequest{
			// 		TeamId: 39,
			// 	},
			// })

			// if err != nil {
			// 	panic(err)
			// }

			// for stream.Receive() {
			// 	dd := stream.Msg()

			// 	debugtool.LogJson(dd)
			// }

			// log.Println(stream.Err())

			var teams []*db_models.Team
			err = db.
				Model(&db_models.Team{}).
				Find(&teams).
				Error

			if err != nil {
				return err
			}

			for _, team := range teams {
				log.Printf("Setup Accounting %s\n", team.Name)
				stream, err := setup.Setup(ctx, &connect.Request[accounting_iface.SetupRequest]{
					Msg: &accounting_iface.SetupRequest{
						TeamId: uint64(team.ID),
					},
				})

				if err != nil {
					slog.Error(err.Error())
					continue
				}

				for stream.Receive() {
					msg := stream.Msg()
					slog.Info(msg.Message)
				}
			}

			return nil
		},
	}
}

func main() {
	mig, err := InitializeMigration()
	if err != nil {
		panic(err)
	}

	err = mig.Run()
	if err != nil {
		panic(err)
	}
}
