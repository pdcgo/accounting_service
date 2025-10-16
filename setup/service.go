package setup

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	"github.com/pdcgo/accounting_service/accounting_core"
	"github.com/pdcgo/schema/services/accounting_iface/v1"
	"github.com/pdcgo/shared/db_models"
	"gorm.io/gorm"
)

type setupServiceImpl struct {
	db *gorm.DB
}

// Setup implements accounting_ifaceconnect.AccountingSetupServiceHandler.
func (s *setupServiceImpl) Setup(
	ctx context.Context,
	req *connect.Request[accounting_iface.SetupRequest],
	stream *connect.ServerStream[accounting_iface.SetupResponse],
) error {
	var err error
	pay := req.Msg
	streamlog := func(msg string) {
		stream.Send(&accounting_iface.SetupResponse{
			Message: msg,
		})
	}

	var team db_models.Team
	err = s.
		db.
		Model(&db_models.Team{}).
		First(&team, pay.TeamId).
		Error

	if err != nil {
		return err
	}
	streamlog(fmt.Sprintf("setup %s with team_id %d", team.Name, pay.TeamId))

	accs := accounting_core.DefaultSeedAccount()
	for _, acc := range accs {
		streamlog(fmt.Sprintf("checking account %s", acc.AccountKey))

		var old accounting_core.Account
		err = s.
			db.
			Model(&accounting_core.Account{}).
			Where("team_id = ?", pay.TeamId).
			Where("account_key = ?", acc.AccountKey).
			Find(&old).
			Error

		if err != nil {
			return err
		}

		if old.ID != 0 {
			old.BalanceType = acc.BalanceType
			old.Coa = acc.Coa
			err = s.db.Save(&old).Error
			if err != nil {
				return err
			}
			continue
		}

		streamlog(fmt.Sprintf("creating account %s", acc.AccountKey))
		err = accounting_core.
			NewCreateAccount(s.db).
			Create(
				acc.BalanceType,
				acc.Coa,
				uint(pay.TeamId),
				acc.AccountKey,
				fmt.Sprintf("%s (%s)", acc.AccountKey, team.Name),
			)

		if err != nil {
			return err
		}

	}

	streamlog("creating permission")
	var tdata db_models.Team
	err = s.db.Model(&db_models.Team{}).First(&tdata, pay.TeamId).Error
	if err != nil {
		streamlog(err.Error())
		return err
	}
	RegisterPermission(s.db, uint(pay.TeamId), tdata.Type, streamlog)
	streamlog("setup completed...")
	return nil
}

func NewSetupService(db *gorm.DB) *setupServiceImpl {
	return &setupServiceImpl{
		db: db,
	}
}
