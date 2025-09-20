package revenue

import (
	"context"
	"fmt"
	"log/slog"
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

// OrderCompleted implements revenue_ifaceconnect.RevenueServiceHandler.
func (r *revenueServiceImpl) OrderCompleted(context.Context, *connect.Request[revenue_iface.OrderCompletedRequest]) (*connect.Response[revenue_iface.OrderCompletedResponse], error) {
	panic("unimplemented")
}

// RevenueAdjustment implements revenue_ifaceconnect.RevenueServiceHandler.
func (r *revenueServiceImpl) RevenueAdjustment(context.Context, *connect.Request[revenue_iface.RevenueAdjustmentRequest]) (*connect.Response[revenue_iface.RevenueAdjustmentResponse], error) {
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

	err = db.Transaction(func(tx *gorm.DB) error {
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

			err = accounting_core.
				NewTransaction(tx).
				Create(&tran).
				Err()

			if err != nil {
				return err
			}
		}

		err = accounting_core.
			NewCreateEntry(tx, uint(pay.TeamId), agent.GetUserID()).
			From(&accounting_core.EntryAccountPayload{
				Key:    accounting_core.SellingReceivableAccount,
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

// OnOrder implements revenue_ifaceconnect.RevenueServiceHandler.
func (r *revenueServiceImpl) OnOrder(
	ctx context.Context,
	stream *connect.ClientStream[revenue_iface.OnOrderRequest],
) (*connect.Response[revenue_iface.OnOrderResponse], error) {
	var err error
	res := connect.NewResponse(&revenue_iface.OnOrderResponse{})
	db := r.db.WithContext(ctx)

	// cancel
	// created

	for stream.Receive() {
		pay := stream.Msg()
		identity := r.
			auth.
			AuthIdentityFromToken(pay.Token)
		agent := identity.Identity()

		err = identity.Err()
		if err != nil {
			slog.Error(err.Error(), slog.String("service", "revenue_service"))
			continue
		}

		switch pay.Event {
		case revenue_iface.OrderEvent_ORDER_EVENT_CREATED:
			err = db.Transaction(func(tx *gorm.DB) error {
				// creating transaction
				ref := accounting_core.NewRefID(&accounting_core.RefData{
					RefType: accounting_core.OrderRef,
					ID:      uint(pay.OrderId),
				})
				tran := accounting_core.Transaction{
					RefID:   ref,
					Desc:    fmt.Sprintf("order from %s", ref),
					Created: time.Now(),
				}
				err = accounting_core.
					NewTransaction(tx).
					Create(&tran).
					Err()
				if err != nil {
					return err
				}

				// creating fee ke gudang
				err = r.createWarehouseFee(tx, agent, pay, &tran)
				if err != nil {
					return err
				}

				// own product
				err = r.ownProductStock(tx, agent, pay, &tran)
				if err != nil {
					return err
				}

				// cross product
				err = r.crossProductStock(tx, agent, pay, &tran)
				if err != nil {
					return err
				}

				// revenue order
				err = r.revenueOrder(tx, agent, pay, &tran)
				if err != nil {
					return err
				}

				return nil
			})

		}

		if err != nil {
			slog.Error(err.Error(), slog.String("stream", "onorder"), slog.Any("payload", pay))
			continue
		}
	}

	return res, stream.Err()
}

func (r *revenueServiceImpl) revenueOrder(
	tx *gorm.DB,
	agent authorization_iface.Identity,
	pay *revenue_iface.OnOrderRequest,
	tran *accounting_core.Transaction,
) error {
	// var err error

	err := accounting_core.
		NewCreateEntry(tx, uint(pay.TeamId), agent.GetUserID()).
		To(&accounting_core.EntryAccountPayload{
			Key:    accounting_core.SalesRevenueAccount,
			TeamID: uint(pay.TeamId),
		}, pay.OrderAmount).
		To(&accounting_core.EntryAccountPayload{
			Key:    accounting_core.SellingReceivableAccount,
			TeamID: uint(pay.TeamId),
		}, pay.OrderAmount).
		Transaction(tran).
		Commit().
		Err()

	if err != nil {
		return err
	}

	return nil
}

func (r *revenueServiceImpl) crossProductStock(
	tx *gorm.DB,
	agent authorization_iface.Identity,
	pay *revenue_iface.OnOrderRequest,
	tran *accounting_core.Transaction,
) error {
	var err error
	if len(pay.BorrowStock) == 0 {
		return nil
	}

	for _, bor := range pay.BorrowStock {
		err = accounting_core.
			NewCreateEntry(tx, uint(bor.TeamId), agent.GetUserID()).
			From(&accounting_core.EntryAccountPayload{
				Key:    accounting_core.StockReadyAccount,
				TeamID: uint(pay.WarehouseId),
			}, bor.Amount).
			To(&accounting_core.EntryAccountPayload{
				Key:    accounting_core.StockBorrowCostAmount,
				TeamID: uint(pay.WarehouseId),
			}, bor.Amount).
			To(&accounting_core.EntryAccountPayload{
				Key:    accounting_core.BorrowStockRevenueAccount,
				TeamID: uint(pay.TeamId),
			}, bor.SellAmount).
			To(&accounting_core.EntryAccountPayload{
				Key:    accounting_core.ReceivableAccount,
				TeamID: uint(pay.TeamId),
			}, bor.SellAmount).
			Transaction(tran).
			Commit().
			Err()

		if err != nil {
			return err
		}

		err = accounting_core.
			NewCreateEntry(tx, uint(pay.TeamId), agent.GetUserID()).
			To(&accounting_core.EntryAccountPayload{
				Key:    accounting_core.StockBorrowCostAmount,
				TeamID: uint(bor.TeamId),
			}, bor.SellAmount).
			To(&accounting_core.EntryAccountPayload{
				Key:    accounting_core.PayableAccount,
				TeamID: uint(bor.TeamId),
			}, bor.SellAmount).
			Transaction(tran).
			Commit().
			Err()

		if err != nil {
			return err
		}

	}

	return nil
}

func (r *revenueServiceImpl) ownProductStock(
	tx *gorm.DB,
	agent authorization_iface.Identity,
	pay *revenue_iface.OnOrderRequest,
	tran *accounting_core.Transaction,
) error {
	var err error

	// sisi selling
	err = accounting_core.
		NewCreateEntry(tx, uint(pay.TeamId), agent.GetUserID()).
		From(&accounting_core.EntryAccountPayload{
			Key:    accounting_core.StockReadyAccount,
			TeamID: uint(pay.WarehouseId),
		}, pay.OwnStockAmount).
		To(&accounting_core.EntryAccountPayload{
			Key:    accounting_core.StockCostAccount,
			TeamID: uint(pay.WarehouseId),
		}, pay.OwnStockAmount).
		Transaction(tran).
		Commit().
		Err()

	if err != nil {
		return err
	}

	// sisi gudang
	err = accounting_core.
		NewCreateEntry(tx, uint(pay.WarehouseId), agent.GetUserID()).
		From(&accounting_core.EntryAccountPayload{
			Key:    accounting_core.StockReadyAccount,
			TeamID: uint(pay.TeamId),
		}, pay.OwnStockAmount).
		To(&accounting_core.EntryAccountPayload{
			Key:    accounting_core.StockCostAccount,
			TeamID: uint(pay.TeamId),
		}, pay.OwnStockAmount).
		Transaction(tran).
		Err()

	if err != nil {
		return err
	}

	return nil
}

func (r *revenueServiceImpl) createWarehouseFee(
	tx *gorm.DB,
	agent authorization_iface.Identity,
	pay *revenue_iface.OnOrderRequest,
	tran *accounting_core.Transaction,
) error {

	var err error

	// sisi gudang
	err = accounting_core.
		NewCreateEntry(tx, uint(pay.WarehouseId), agent.GetUserID()).
		To(&accounting_core.EntryAccountPayload{
			Key:    accounting_core.ServiceRevenueAccount, // revenue gudang [pendapatan jasa]
			TeamID: uint(pay.TeamId),
		}, pay.WarehouseFee).
		To(&accounting_core.EntryAccountPayload{
			Key:    accounting_core.ReceivableAccount,
			TeamID: uint(pay.TeamId),
		}, pay.WarehouseFee).
		Transaction(tran).
		Commit().
		Err()

	if err != nil {
		return err
	}

	// sisi selling
	err = accounting_core.
		NewCreateEntry(tx, uint(pay.TeamId), agent.GetUserID()).
		To(&accounting_core.EntryAccountPayload{
			Key:    accounting_core.WarehouseCostAccount,
			TeamID: uint(pay.WarehouseId),
		},
			pay.WarehouseFee).
		To(&accounting_core.EntryAccountPayload{
			Key:    accounting_core.PayableAccount,
			TeamID: uint(pay.WarehouseId),
		}, pay.WarehouseFee).
		Transaction(tran).
		Commit().
		Err()

	if err != nil {
		return err
	}

	return nil

}

func NewRevenueService(db *gorm.DB, auth authorization_iface.Authorization) *revenueServiceImpl {
	return &revenueServiceImpl{
		db:   db,
		auth: auth,
	}
}
