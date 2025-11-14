package payment

import (
	"context"
	"time"

	"connectrpc.com/connect"
	"github.com/pdcgo/accounting_service/accounting_core"
	"github.com/pdcgo/accounting_service/accounting_model"
	"github.com/pdcgo/schema/services/payment_iface/v1"
	"github.com/pdcgo/shared/interfaces/authorization_iface"
	"gorm.io/gorm"
)

// PaymentCreate implements payment_ifaceconnect.PaymentServiceHandler.
func (p *paymentServiceImpl) PaymentCreate(
	ctx context.Context,
	req *connect.Request[payment_iface.PaymentCreateRequest],
) (*connect.Response[payment_iface.PaymentCreateResponse], error) {
	var err error
	result := payment_iface.PaymentCreateResponse{}

	return connect.NewResponse(&result), nil

	db := p.db.WithContext(ctx)
	pay := req.Msg

	identity := p.auth.AuthIdentityFromHeader(req.Header())
	agent := identity.Identity()

	err = identity.
		HasPermission(authorization_iface.CheckPermissionGroup{
			&accounting_model.Payment{}: &authorization_iface.CheckPermission{
				DomainID: uint(pay.FromTeamId),
				Actions:  []authorization_iface.Action{authorization_iface.Create},
			},
		}).
		Err()

	if err != nil {
		return connect.NewResponse(&result), err
	}

	err = accounting_core.OpenTransaction(db, func(tx *gorm.DB, bookmng accounting_core.BookManage) error {
		payment := accounting_model.Payment{
			FromTeamID:  uint(pay.FromTeamId),
			ToTeamID:    uint(pay.ToTeamId),
			Amount:      pay.Amount,
			Status:      payment_iface.PaymentStatus_PAYMENT_STATUS_PENDING,
			PaymentType: pay.PaymentType,
			CreatedByID: agent.IdentityID(),
			CreatedAt:   time.Now(),
		}

		err = tx.Save(&payment).Error
		if err != nil {
			return err
		}
		ref := accounting_core.NewRefID(&accounting_core.RefData{
			RefType: accounting_core.PaymentRef,
			ID:      payment.ID,
		})
		tran := accounting_core.Transaction{
			RefID:       ref,
			TeamID:      uint(pay.FromTeamId),
			CreatedByID: agent.IdentityID(),
			Desc:        pay.Description + string(ref),
			Created:     time.Now(),
		}

		err = bookmng.
			NewTransaction().
			Create(&tran).
			Err()

		if err != nil {
			return err
		}

		// sisi pengirim
		err = bookmng.
			NewCreateEntry(payment.FromTeamID, agent.IdentityID()).
			From(&accounting_core.EntryAccountPayload{
				Key:    accounting_core.CashAccount,
				TeamID: uint(pay.ToTeamId),
			}, pay.Amount).
			To(&accounting_core.EntryAccountPayload{
				Key:    accounting_core.PendingPaymentPayAccount,
				TeamID: uint(pay.ToTeamId),
			}, pay.Amount).
			Transaction(&tran).
			Commit().
			Err()

		if err != nil {
			return err
		}

		// sisi penerima
		err = bookmng.
			NewCreateEntry(payment.ToTeamID, agent.IdentityID()).
			From(&accounting_core.EntryAccountPayload{
				Key:    accounting_core.ReceivableAccount,
				TeamID: uint(pay.FromTeamId),
			}, pay.Amount).
			To(&accounting_core.EntryAccountPayload{
				Key:    accounting_core.PendingPaymentReceiveAccount,
				TeamID: uint(pay.FromTeamId),
			}, pay.Amount).
			Transaction(&tran).
			Commit().
			Err()

		if err != nil {
			return err
		}

		result.PaymentId = uint64(payment.ID)
		result.Status = payment_iface.PaymentStatus_PAYMENT_STATUS_PENDING
		return nil
	})

	if err != nil {
		return connect.NewResponse(&result), err
	}

	return connect.NewResponse(&result), nil
}
