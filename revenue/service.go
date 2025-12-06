package revenue

import (
	"context"
	"fmt"
	"time"

	"connectrpc.com/connect"
	"github.com/pdcgo/accounting_service/accounting_core"
	"github.com/pdcgo/accounting_service/report"
	"github.com/pdcgo/schema/services/revenue_iface/v1"
	"github.com/pdcgo/shared/configs"
	"github.com/pdcgo/shared/interfaces/authorization_iface"
	"gorm.io/gorm"
)

type revenueServiceImpl struct {
	db                      *gorm.DB
	auth                    authorization_iface.Authorization
	accountingServiceConfig *configs.AccountingService
	cfg                     *configs.DispatcherConfig
	dispatcher              report.ReportDispatcher
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

	err = accounting_core.OpenTransaction(ctx, db, func(tx *gorm.DB, bookmng accounting_core.BookManage) error {
		ref := accounting_core.NewRefID(&accounting_core.RefData{
			RefType: accounting_core.WithdrawalRef,
			ID:      uint(pay.ShopId),
		})

		var tran accounting_core.Transaction

		txmut := accounting_core.
			NewTransactionMutation(ctx, tx).
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

func NewRevenueService(
	db *gorm.DB,
	auth authorization_iface.Authorization,
	accountingServiceConfig *configs.AccountingService,
	cfg *configs.DispatcherConfig,
	dispatcher report.ReportDispatcher,
) *revenueServiceImpl {
	return &revenueServiceImpl{
		db:                      db,
		auth:                    auth,
		accountingServiceConfig: accountingServiceConfig,
		cfg:                     cfg,
		dispatcher:              dispatcher,
	}
}
