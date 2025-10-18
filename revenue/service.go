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

type revenueServiceImpl struct {
	db   *gorm.DB
	auth authorization_iface.Authorization
}

// RevenueAdjustment implements revenue_ifaceconnect.RevenueServiceHandler.
func (r *revenueServiceImpl) RevenueAdjustment(context.Context, *connect.Request[revenue_iface.RevenueAdjustmentRequest]) (*connect.Response[revenue_iface.RevenueAdjustmentResponse], error) {
	panic("unimplemented")
}

// OrderCompleted implements revenue_ifaceconnect.RevenueServiceHandler.
func (r *revenueServiceImpl) OrderCompleted(context.Context, *connect.Request[revenue_iface.OrderCompletedRequest]) (*connect.Response[revenue_iface.OrderCompletedResponse], error) {
	panic("unimplemented")
}

// Withdrawal implements revenue_ifaceconnect.RevenueServiceHandler.
func (r *revenueServiceImpl) Withdrawal(
	ctx context.Context,
	req *connect.Request[revenue_iface.WithdrawalRequest],
) (*connect.Response[revenue_iface.WithdrawalResponse], error) {
	var err error
	res := connect.NewResponse(&revenue_iface.WithdrawalResponse{})
	db := r.db.WithContext(ctx)
	pay := req.Msg

	identity := r.
		auth.
		AuthIdentityFromHeader(req.Header())
	agent := identity.Identity()

	err = identity.Err()
	if err != nil {
		return res, err
	}

	err = accounting_core.OpenTransaction(db, func(tx *gorm.DB, bookmng accounting_core.BookManage) error {
		ref := accounting_core.NewRefID(&accounting_core.RefData{
			RefType: accounting_core.WithdrawalRef,
			ID:      uint(pay.ShopId),
		})

		var tran accounting_core.Transaction

		txmut := accounting_core.
			NewTransactionMutation(tx).
			ByRefID(ref, true)

		err = txmut.
			Err()
		if err != nil {
			return err
		}

		texist := txmut.IsExist()

		if texist {
			err = txmut.
				RollbackEntry(agent.GetUserID(), fmt.Sprintf("reset withdrawal %d at %s", pay.ShopId, time.Now().String())).
				Err()

			if err != nil {
				return err
			}

			tran = *txmut.Data()

		} else {
			tran = accounting_core.Transaction{
				Desc:        fmt.Sprintf("withdrawal for %d", pay.ShopId),
				TeamID:      uint(pay.TeamId),
				RefID:       ref,
				CreatedByID: agent.GetUserID(),
				Created:     time.Now(),
			}

			err = bookmng.
				NewTransaction().
				Create(&tran).
				Err()

			if err != nil {
				return err
			}
		}

		err = bookmng.
			NewCreateEntry(uint(pay.TeamId), agent.GetUserID()).
			From(&accounting_core.EntryAccountPayload{
				Key:    accounting_core.SellingEstReceivableAccount,
				TeamID: uint(pay.TeamId),
			}, pay.Amount).
			To(&accounting_core.EntryAccountPayload{
				Key:    accounting_core.CashAccount,
				TeamID: uint(pay.TeamId),
			}, pay.Amount).
			Transaction(&tran).
			Commit().
			Err()

		if err != nil {
			return err
		}

		return nil
	})

	return res, err
}

// OrderCancel implements revenue_ifaceconnect.RevenueServiceHandler.
func (r *revenueServiceImpl) OrderCancel(
	ctx context.Context,
	req *connect.Request[revenue_iface.OrderCancelRequest],
) (*connect.Response[revenue_iface.OrderCancelResponse], error) {
	var err error
	res := connect.NewResponse(&revenue_iface.OrderCancelResponse{})
	db := r.db.WithContext(ctx)
	pay := req.Msg

	identity := r.
		auth.
		AuthIdentityFromHeader(req.Header()).
		HasPermission(authorization_iface.CheckPermissionGroup{
			&db_models.Order{}: &authorization_iface.CheckPermission{
				DomainID: uint(pay.TeamId),
				Actions:  []authorization_iface.Action{authorization_iface.Update},
			},
		})
	agent := identity.Identity()

	err = identity.Err()
	if err != nil {
		return res, err
	}

	err = db.Transaction(func(tx *gorm.DB) error {
		ref := accounting_core.NewRefID(&accounting_core.RefData{
			RefType: accounting_core.OrderRef,
			ID:      uint(pay.OrderId),
		})
		return accounting_core.
			NewTransactionMutation(tx).
			ByRefID(accounting_core.RefID(ref), true).
			RollbackEntry(agent.GetUserID(), fmt.Sprintf("cancelling order %s", ref)).
			Err()
	})

	return res, err
}

func NewRevenueService(db *gorm.DB, auth authorization_iface.Authorization) *revenueServiceImpl {
	return &revenueServiceImpl{
		db:   db,
		auth: auth,
	}
}
