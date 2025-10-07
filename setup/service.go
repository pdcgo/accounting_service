package setup

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	"github.com/pdcgo/accounting_service/accounting_core"
	"github.com/pdcgo/accounting_service/accounting_model"
	"github.com/pdcgo/schema/services/accounting_iface/v1"
	"github.com/pdcgo/shared/authorization"
	"github.com/pdcgo/shared/db_models"
	"github.com/pdcgo/shared/interfaces/authorization_iface"
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
	dom := authorization.NewDomainV2(s.db, uint(pay.TeamId))
	err = dom.RoleAddPermission("owner", authorization_iface.RoleAddPermissionPayload{
		&accounting_model.BankTransfer{}: []*authorization_iface.RoleAddPermissionItem{
			{
				Action: authorization_iface.Create,
				Policy: authorization_iface.Allow,
			},
			{
				Action: authorization_iface.Read,
				Policy: authorization_iface.Allow,
			},
			{
				Action: authorization_iface.Delete,
				Policy: authorization_iface.Allow,
			},
			{
				Action: authorization_iface.Update,
				Policy: authorization_iface.Allow,
			},
		},
		&accounting_model.ExpenseEntity{}: []*authorization_iface.RoleAddPermissionItem{
			{
				Action: authorization_iface.Create,
				Policy: authorization_iface.Allow,
			},
			{
				Action: authorization_iface.Read,
				Policy: authorization_iface.Allow,
			},
			{
				Action: authorization_iface.Delete,
				Policy: authorization_iface.Allow,
			},
			{
				Action: authorization_iface.Update,
				Policy: authorization_iface.Allow,
			},
		},
		&accounting_model.Payment{}: []*authorization_iface.RoleAddPermissionItem{
			{
				Action: authorization_iface.Create,
				Policy: authorization_iface.Allow,
			},
			{
				Action: authorization_iface.Read,
				Policy: authorization_iface.Allow,
			},
			{
				Action: authorization_iface.Delete,
				Policy: authorization_iface.Allow,
			},
			{
				Action: authorization_iface.Update,
				Policy: authorization_iface.Allow,
			},
		},
	})

	if err != nil {
		streamlog(err.Error())
	}

	err = dom.RoleAddPermission("admin", authorization_iface.RoleAddPermissionPayload{
		&accounting_model.BankTransfer{}: []*authorization_iface.RoleAddPermissionItem{
			{
				Action: authorization_iface.Create,
				Policy: authorization_iface.Allow,
			},
			{
				Action: authorization_iface.Read,
				Policy: authorization_iface.Allow,
			},
			{
				Action: authorization_iface.Delete,
				Policy: authorization_iface.Allow,
			},
			{
				Action: authorization_iface.Update,
				Policy: authorization_iface.Allow,
			},
		},
		&accounting_model.ExpenseEntity{}: []*authorization_iface.RoleAddPermissionItem{
			{
				Action: authorization_iface.Create,
				Policy: authorization_iface.Allow,
			},
			{
				Action: authorization_iface.Read,
				Policy: authorization_iface.Allow,
			},
			{
				Action: authorization_iface.Delete,
				Policy: authorization_iface.Allow,
			},
			{
				Action: authorization_iface.Update,
				Policy: authorization_iface.Allow,
			},
		},
		&accounting_model.Payment{}: []*authorization_iface.RoleAddPermissionItem{
			{
				Action: authorization_iface.Create,
				Policy: authorization_iface.Allow,
			},
			{
				Action: authorization_iface.Read,
				Policy: authorization_iface.Allow,
			},
			{
				Action: authorization_iface.Delete,
				Policy: authorization_iface.Allow,
			},
			{
				Action: authorization_iface.Update,
				Policy: authorization_iface.Allow,
			},
		},
	})
	if err != nil {
		streamlog(err.Error())
	}
	streamlog("setup completed...")
	return nil
}

func NewSetupService(db *gorm.DB) *setupServiceImpl {
	return &setupServiceImpl{
		db: db,
	}
}
