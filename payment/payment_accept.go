package payment

import (
	"context"
	"errors"
	"fmt"

	"connectrpc.com/connect"
	"github.com/pdcgo/accounting_service/accounting_core"
	"github.com/pdcgo/accounting_service/accounting_model"
	"github.com/pdcgo/schema/services/common/v1"
	"github.com/pdcgo/schema/services/payment_iface/v1"
	"github.com/pdcgo/shared/authorization"
	"github.com/pdcgo/shared/interfaces/authorization_iface"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// PaymentAccept implements payment_ifaceconnect.PaymentServiceHandler.
func (p *paymentServiceImpl) PaymentAccept(
	ctx context.Context,
	req *connect.Request[payment_iface.PaymentAcceptRequest],
) (*connect.Response[payment_iface.PaymentAcceptResponse], error) {
	var err error
	result := payment_iface.PaymentAcceptResponse{}
	db := p.db.WithContext(ctx)
	pay := req.Msg

	identity := p.auth.AuthIdentityFromHeader(req.Header())
	agent := identity.Identity()

	var domainID uint
	switch pay.RequestFrom {
	case common.RequestFrom_REQUEST_FROM_ADMIN:
		domainID = authorization.RootDomain
	default:
		domainID = uint(pay.TeamId)
	}

	err = identity.
		HasPermission(authorization_iface.CheckPermissionGroup{
			&accounting_model.Payment{}: &authorization_iface.CheckPermission{
				DomainID: domainID,
				Actions:  []authorization_iface.Action{authorization_iface.Update},
			},
		}).
		Err()

	if err != nil {
		return connect.NewResponse(&result), err
	}

	err = accounting_core.OpenTransaction(db, func(tx *gorm.DB, bookmng accounting_core.BookManage) error {
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

		if payment.ToTeamID != uint(pay.TeamId) {
			return errors.New("payment not you own")
		}
		if payment.Status != payment_iface.PaymentStatus_PAYMENT_STATUS_PENDING {
			return errors.New("payment not pending")
		}

		ref := accounting_core.NewRefID(&accounting_core.RefData{
			RefType: accounting_core.PaymentRef,
			ID:      uint(pay.PaymentId),
		})

		txmut := accounting_core.
			NewTransactionMutation(tx).
			ByRefID(ref, true)
		err = txmut.
			Err()

		if err != nil {
			return err
		}

		trans := txmut.Data()

		desc := accounting_core.EntryDescOption(fmt.Sprintf("accept payment %s", ref))

		// sisi pengirim
		err = bookmng.
			NewCreateEntry(payment.FromTeamID, agent.IdentityID()).
			From(&accounting_core.EntryAccountPayload{
				Key:    accounting_core.PaymentInTransitAccount,
				TeamID: payment.ToTeamID,
			}, payment.Amount, desc).
			From(&accounting_core.EntryAccountPayload{
				Key:    accounting_core.PayableAccount,
				TeamID: payment.ToTeamID,
			}, payment.Amount, desc).
			Transaction(trans).
			Commit().
			Err()

		if err != nil {
			return err
		}

		// sisi penerima
		err = bookmng.
			NewCreateEntry(payment.ToTeamID, agent.IdentityID()).
			From(&accounting_core.EntryAccountPayload{
				Key:    accounting_core.PaymentInTransitAccount,
				TeamID: payment.FromTeamID,
			}, payment.Amount, desc).
			To(&accounting_core.EntryAccountPayload{
				Key:    accounting_core.CashAccount,
				TeamID: payment.ToTeamID,
			}, payment.Amount, desc).
			Transaction(trans).
			Commit().
			Err()

		if err != nil {
			return err
		}

		payment.Status = payment_iface.PaymentStatus_PAYMENT_STATUS_ACCEPTED
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
