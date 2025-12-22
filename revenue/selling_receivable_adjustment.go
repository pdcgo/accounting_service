package revenue

import (
	"context"
	"errors"
	"fmt"
	"math"
	"time"

	"connectrpc.com/connect"
	"github.com/pdcgo/accounting_service/accounting_core"
	"github.com/pdcgo/schema/services/revenue_iface/v1"
	"gorm.io/gorm"
)

// SellingReceivableAdjustment implements revenue_ifaceconnect.RevenueServiceHandler.
func (r *revenueServiceImpl) SellingReceivableAdjustment(ctx context.Context, req *connect.Request[revenue_iface.SellingReceivableAdjustmentRequest]) (*connect.Response[revenue_iface.SellingReceivableAdjustmentResponse], error) {
	var err error

	identity := r.auth.
		AuthIdentityFromHeader(req.Header())

	agent := identity.
		Identity()

	err = identity.Err()

	if err != nil {
		return nil, err
	}

	db := r.db.WithContext(ctx)
	pay := req.Msg
	result := revenue_iface.SellingReceivableAdjustmentResponse{}

	err = accounting_core.OpenTransaction(ctx, db, func(tx *gorm.DB, bookmng accounting_core.BookManage) error {

		var ref accounting_core.RefID = accounting_core.NewStringRefID(&accounting_core.StringRefData{
			RefType: accounting_core.RevenueAdjustmentRef,
			ID:      pay.AdjRefId,
		})

		txmut := accounting_core.NewTransactionMutation(ctx, tx)
		txmut.
			ByRefID(ref, true)

		err = txmut.Err()
		if err != nil {
			if !errors.Is(err, accounting_core.ErrTransactionNotFound) {
				return err
			}
		} else {
			err = txmut.
				RollbackEntry(agent.IdentityID(), fmt.Sprintf("rollback %s with ref %s", pay.Desc, ref)).
				Err()
			if err != nil {
				return err
			}
		}

		if pay.OnlyRollback {
			return nil
		}

		if pay.Amount == 0 {
			return errors.New("amount is zero")
		}

		tran := accounting_core.Transaction{
			RefID:       ref,
			TeamID:      uint(pay.TeamId),
			CreatedByID: agent.IdentityID(),
			Desc:        pay.Desc,
			Created:     time.Now(),
		}

		err = bookmng.
			NewTransaction().
			Create(&tran).
			AddCustomerServiceID(agent.IdentityID()).
			AddShopID(uint(pay.ShopId)).
			Err()

		if err != nil {
			return err
		}

		entry := bookmng.
			NewCreateEntry(uint(pay.TeamId), agent.IdentityID())

		switch pay.Type {
		case revenue_iface.ReceivableAdjustmentType_RECEIVABLE_ADJUSTMENT_TYPE_RETURN_COST:
			err = returnCost(entry, pay)
		case revenue_iface.ReceivableAdjustmentType_RECEIVABLE_ADJUSTMENT_TYPE_REFUND_LOST:
			err = refundLost(entry, pay)
		default:
			return errors.New("unimplemented")
		}

		if err != nil {
			return err
		}

		err = entry.
			Transaction(&tran).
			Commit().
			Err()

		return err
	})

	return connect.NewResponse(&result), err

}

func refundLost(entry accounting_core.CreateEntry, pay *revenue_iface.SellingReceivableAdjustmentRequest) error {
	if pay.Amount < 0 {
		return errors.New("refund lost with value negative not implemented")
	}

	entry.
		To(&accounting_core.EntryAccountPayload{
			Key:    accounting_core.SellingReceivableAccount,
			TeamID: uint(pay.TeamId),
		}, pay.Amount).
		To(&accounting_core.EntryAccountPayload{
			Key:    accounting_core.OtherRevenueAccount,
			TeamID: uint(pay.TeamId),
		}, pay.Amount)

	return nil
}

func returnCost(entry accounting_core.CreateEntry, pay *revenue_iface.SellingReceivableAdjustmentRequest) error {
	if pay.Amount < 0 {
		amount := math.Abs(pay.Amount)
		entry.
			From(&accounting_core.EntryAccountPayload{
				Key:    accounting_core.SellingReceivableAccount,
				TeamID: uint(pay.TeamId),
			}, amount).
			To(&accounting_core.EntryAccountPayload{
				Key:    accounting_core.SellingReturnExpenseAccount,
				TeamID: uint(pay.TeamId),
			}, amount)

	} else {
		entry.
			To(&accounting_core.EntryAccountPayload{
				Key:    accounting_core.OtherRevenueAccount,
				TeamID: uint(pay.TeamId),
			}, pay.Amount).
			To(&accounting_core.EntryAccountPayload{
				Key:    accounting_core.SellingReceivableAccount,
				TeamID: uint(pay.TeamId),
			}, pay.Amount)
	}

	return nil
}
