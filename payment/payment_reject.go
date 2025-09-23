package payment

import (
	"context"
	"errors"
	"fmt"

	"connectrpc.com/connect"
	"github.com/pdcgo/accounting_service/accounting_core"
	"github.com/pdcgo/accounting_service/accounting_model"
	"github.com/pdcgo/schema/services/payment_iface/v1"
	"github.com/pdcgo/shared/interfaces/authorization_iface"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// PaymentReject implements payment_ifaceconnect.PaymentServiceHandler.
func (p *paymentServiceImpl) PaymentReject(
	ctx context.Context,
	req *connect.Request[payment_iface.PaymentRejectRequest],
) (*connect.Response[payment_iface.PaymentRejectResponse], error) {
	var err error

	db := p.db.WithContext(ctx)
	result := payment_iface.PaymentRejectResponse{}
	pay := req.Msg

	identity := p.
		auth.
		AuthIdentityFromHeader(req.Header())

	err = identity.
		HasPermission(authorization_iface.CheckPermissionGroup{
			&accounting_model.Payment{}: &authorization_iface.CheckPermission{
				DomainID: uint(pay.TeamId),
				Actions:  []authorization_iface.Action{authorization_iface.Update},
			},
		}).
		Err()

	if err != nil {
		return connect.NewResponse(&result), err
	}

	agent := identity.Identity()

	err = db.Transaction(func(tx *gorm.DB) error {
		var payment accounting_model.Payment
		err = tx.
			Clauses(clause.Locking{
				Strength: "UPDATE",
			}).
			Model(&accounting_model.Payment{}).
			First(&payment, pay.PaymentId).
			Error
		if err != nil {
			return err
		}

		if payment.FromTeamID != uint(pay.TeamId) {
			return errors.New("payment not you own")
		}

		if payment.Status != payment_iface.PaymentStatus_PAYMENT_STATUS_PENDING {
			return errors.New("payment not pending")
		}

		ref := accounting_core.NewRefID(&accounting_core.RefData{
			RefType: accounting_core.PaymentRef,
			ID:      uint(pay.PaymentId),
		})
		err = accounting_core.
			NewTransactionMutation(tx).
			ByRefID(ref, true).
			RollbackEntry(agent.IdentityID(), fmt.Sprintf("reject %s", ref)).
			Err()

		if err != nil {
			return err
		}

		payment.Status = payment_iface.PaymentStatus_PAYMENT_STATUS_REJECTED
		err = tx.Save(&payment).Error
		if err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return connect.NewResponse(&result), err
	}

	return connect.NewResponse(&result), nil
}
