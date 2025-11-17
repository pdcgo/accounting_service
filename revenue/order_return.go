package revenue

import (
	"context"
	"fmt"
	"time"

	"connectrpc.com/connect"
	"github.com/pdcgo/accounting_service/accounting_core"
	"github.com/pdcgo/schema/services/revenue_iface/v1"
	"github.com/pdcgo/shared/db_models"
	"github.com/pdcgo/shared/interfaces/authorization_iface"
	"gorm.io/gorm"
)

// OrderReturn implements revenue_ifaceconnect.RevenueServiceHandler.
func (r *revenueServiceImpl) OrderReturn(
	ctx context.Context,
	req *connect.Request[revenue_iface.OrderReturnRequest],
) (*connect.Response[revenue_iface.OrderReturnResponse], error) {
	var err error

	result := revenue_iface.OrderReturnResponse{}
	pay := req.Msg

	identity := r.auth.AuthIdentityFromHeader(req.Header())
	agent := identity.Identity()
	identity.
		HasPermission(authorization_iface.CheckPermissionGroup{
			&db_models.Order{}: &authorization_iface.CheckPermission{
				DomainID: uint(pay.TeamId),
				Actions:  []authorization_iface.Action{authorization_iface.Update},
			},
		})

	err = identity.Err()
	if err != nil {
		return connect.NewResponse(&result), err
	}

	db := r.db.WithContext(ctx)
	err = accounting_core.OpenTransaction(ctx, db, func(tx *gorm.DB, bookmng accounting_core.BookManage) error {
		ref := accounting_core.NewRefID(&accounting_core.RefData{
			RefType: accounting_core.OrderReturnRef,
			ID:      uint(pay.OrderId),
		})
		tran := accounting_core.Transaction{
			RefID:       ref,
			TeamID:      uint(pay.TeamId),
			CreatedByID: agent.IdentityID(),
			Desc:        fmt.Sprintf("returning order %s", ref),
			Created:     time.Now(),
		}

		err = bookmng.
			NewTransaction().
			Create(&tran).
			Err()
		if err != nil {
			return err
		}

		// entry selling
		entry := bookmng.
			NewCreateEntry(uint(pay.TeamId), agent.IdentityID()).
			From(&accounting_core.EntryAccountPayload{
				Key:    accounting_core.SalesRevenueAccount,
				TeamID: uint(pay.TeamId),
			}, pay.OrderAmount).
			To(&accounting_core.EntryAccountPayload{
				Key:    accounting_core.SalesReturnRevenueAccount,
				TeamID: uint(pay.TeamId),
			}, pay.OrderAmount).
			From(&accounting_core.EntryAccountPayload{
				Key:    accounting_core.StockCostAccount,
				TeamID: uint(pay.WarehouseId),
			}, pay.StockAmount).
			To(&accounting_core.EntryAccountPayload{
				Key:    accounting_core.StockPendingAccount,
				TeamID: uint(pay.WarehouseId),
			}, pay.StockAmount)

		err = entry.
			Transaction(&tran).
			Commit().
			Err()

		if err != nil {
			return err
		}

		// entry gudang
		entry = bookmng.
			NewCreateEntry(uint(pay.WarehouseId), agent.IdentityID()).
			From(&accounting_core.EntryAccountPayload{
				Key:    accounting_core.StockCostAccount,
				TeamID: uint(pay.TeamId),
			}, pay.StockAmount).
			To(&accounting_core.EntryAccountPayload{
				Key:    accounting_core.StockPendingAccount,
				TeamID: uint(pay.TeamId),
			}, pay.StockAmount)

		err = entry.
			Transaction(&tran).
			Commit().
			Err()

		if err != nil {
			return err
		}
		return nil
	})

	return connect.NewResponse(&result), err
}
