package adjustment

import (
	"context"
	"errors"
	"fmt"
	"math"
	"time"

	"connectrpc.com/connect"
	"github.com/pdcgo/accounting_service/accounting_core"
	"github.com/pdcgo/schema/services/accounting_iface/v1"
	"github.com/pdcgo/schema/services/common/v1"
	"github.com/pdcgo/shared/authorization"
	"github.com/pdcgo/shared/interfaces/authorization_iface"
	"gorm.io/gorm"
)

type AdjustmentAccess struct{}

// GetEntityID implements authorization.Entity.
func (a *AdjustmentAccess) GetEntityID() string {
	return "accounting/adjustment"
}

// AccountAdjustment implements accounting_ifaceconnect.AdjustmentServiceHandler.
func (a *adjServiceImpl) AccountAdjustment(
	ctx context.Context,
	req *connect.Request[accounting_iface.AccountAdjustmentRequest],
) (*connect.Response[accounting_iface.AccountAdjustmentResponse], error) {
	var err error
	result := accounting_iface.AccountAdjustmentResponse{}
	pay := req.Msg

	identity := a.
		auth.
		AuthIdentityFromHeader(req.Header())

	agent := identity.Identity()

	var domainID uint
	switch pay.RequestFrom {
	case common.RequestFrom_REQUEST_FROM_ADMIN:
		domainID = authorization.RootDomain
	case common.RequestFrom_REQUEST_FROM_SELLING, common.RequestFrom_REQUEST_FROM_WAREHOUSE:
		domainID = uint(pay.TeamId)

	}

	err = identity.
		HasPermission(authorization_iface.CheckPermissionGroup{
			&AdjustmentAccess{}: &authorization_iface.CheckPermission{
				DomainID: domainID,
			},
		}).
		Err()

	if err != nil {
		return &connect.Response[accounting_iface.AccountAdjustmentResponse]{}, err
	}

	// checking payload
	for _, adjTeam := range pay.Adjustments {
		// checking teamid
		adjcount := len(adjTeam.Adjs)
		if adjcount > 1 {
			if pay.RequestFrom != common.RequestFrom_REQUEST_FROM_ADMIN {
				return nil, errors.New("adjustment multiple team needs admin")
			}

			for _, ads := range adjTeam.Adjs {
				if adjTeam.Adjs[ads.TeamId] == nil {
					return nil, fmt.Errorf("adjustment for teamid %d not found", ads.TeamId)
				}
			}
		}

		if adjcount == 1 {
			for teamID := range adjTeam.Adjs {
				if pay.RequestFrom != common.RequestFrom_REQUEST_FROM_ADMIN {
					if pay.TeamId != teamID {
						return nil, errors.New("teamid adjustment not same")
					}
				}

			}

		}

	}

	db := a.db.WithContext(ctx)
	err = accounting_core.OpenTransaction(db, func(tx *gorm.DB, bookmng accounting_core.BookManage) error {

		for _, adjTeam := range pay.Adjustments {
			// create transaction
			ref := accounting_core.NewStringRefID(&accounting_core.StringRefData{
				RefType: accounting_core.AdjustmentRef,
				ID:      fmt.Sprintf("%d-%d", time.Now().Unix(), agent.IdentityID()),
			})

			tran := accounting_core.Transaction{
				RefID:       ref,
				TeamID:      uint(pay.TeamId),
				Desc:        adjTeam.Description,
				CreatedByID: agent.IdentityID(),
				Created:     time.Now(),
			}

			for bookTeamID, adj := range adjTeam.Adjs {
				// getting acccount id
				var acc, adjAcc accounting_core.Account
				err = tx.
					Model(&accounting_core.Account{}).
					Where("account_key = ?", adj.AccountKey).
					Where("team_id = ?", adj.TeamId).
					Find(&acc).
					Error

				if err != nil {
					return err
				}
				if acc.ID == 0 {
					return fmt.Errorf("account %s in team %d not found", adj.AccountKey, adj.TeamId)
				}

				var adkey accounting_core.AccountKey

				switch acc.Coa {
				case accounting_core.ASSET:
					adkey = accounting_core.AdjAssetAccount
				case accounting_core.EQUITY:
					adkey = accounting_core.AdjEquityAccount
				case accounting_core.EXPENSE:
					adkey = accounting_core.AdjExpenseAccount
				case accounting_core.LIABILITY:
					adkey = accounting_core.AdjLiabilityAccount
				case accounting_core.REVENUE:
					adkey = accounting_core.AdjRevenueAccount
				}

				err = tx.
					Model(&accounting_core.Account{}).
					Where("account_key = ? AND team_id = ?", adkey, adj.TeamId).
					Find(&adjAcc).
					Error

				if err != nil {
					return err
				}

				if adjAcc.ID == 0 {
					adjAcc = accounting_core.Account{
						AccountKey:  adkey,
						Coa:         acc.Coa,
						TeamID:      uint(adj.TeamId),
						BalanceType: acc.BalanceType,
						Name:        fmt.Sprintf("adjustment %s", acc.Name),
						Created:     time.Now(),
					}
					err = tx.
						Save(&adjAcc).
						Error

					if err != nil {
						return err
					}
				}

				err = bookmng.
					NewTransaction().
					Create(&tran).
					AddTags([]string{fmt.Sprintf("adj_from_%s", pay.RequestFrom)}).
					AddCustomerServiceID(agent.IdentityID()).
					Err()

				if err != nil {
					return err
				}

				entry := bookmng.NewCreateEntry(uint(bookTeamID), agent.IdentityID())

				if adj.Amount > 0 {
					entry.
						From(&accounting_core.EntryAccountPayload{
							Key:    adjAcc.AccountKey,
							TeamID: uint(adj.TeamId),
						}, adj.Amount).
						To(&accounting_core.EntryAccountPayload{
							Key:    acc.AccountKey,
							TeamID: uint(adj.TeamId),
						}, adj.Amount)

				}

				if adj.Amount < 0 {
					amount := math.Abs(adj.Amount)
					entry.
						From(&accounting_core.EntryAccountPayload{
							Key:    acc.AccountKey,
							TeamID: uint(adj.TeamId),
						}, amount).
						To(&accounting_core.EntryAccountPayload{
							Key:    adjAcc.AccountKey,
							TeamID: uint(adj.TeamId),
						}, amount)
				}

				err = entry.
					Transaction(&tran).
					Commit().
					Err()

				if err != nil {
					return err
				}

			}
		}
		return nil
	})

	return connect.NewResponse(&result), err

}
