package payment_transaction

import (
	"context"

	"github.com/pdcgo/accounting_service/accounting_core"
	"github.com/pdcgo/shared/interfaces/identity_iface"
	"gorm.io/gorm"
)

type PaymentPayload struct {
	// FromAccountID uint    `json:"from_account_id"`
	// ToAccountID   uint    `json:"to_account_id"`
	// TeamID        uint    `json:"team_id"`
	ToTeamID   uint    `json:"to_team_id"`
	FromTeamID uint    `json:"from_team_id"`
	Desc       string  `json:"desc"`
	Amount     float64 `json:"amount"`
}

type PaymentTransaction interface {
	Payment(payment *PaymentPayload) error
}

type paymentPaymentTransactionImpl struct {
	ctx   context.Context
	agent identity_iface.Agent
	tx    *gorm.DB
}

// Payment implements PaymentTransaction.
func (p *paymentPaymentTransactionImpl) Payment(payment *PaymentPayload) error {
	return accounting_core.OpenTransaction(p.ctx, p.tx, func(tx *gorm.DB, bookmng accounting_core.BookManage) error {
		var err error
		var tran accounting_core.Transaction

		err = bookmng.
			NewTransaction().
			Create(&tran).
			Err()

		if err != nil {
			return err
		}

		entry := bookmng.NewCreateEntry(payment.FromTeamID, p.agent.GetUserID())
		err = entry.
			To(&accounting_core.EntryAccountPayload{
				Key:    accounting_core.CashAccount,
				TeamID: payment.FromTeamID,
			}, payment.Amount).
			From(&accounting_core.EntryAccountPayload{
				Key:    accounting_core.PayableAccount,
				TeamID: payment.ToTeamID,
			}, payment.Amount).
			Transaction(&tran).
			Commit().
			Err()

		if err != nil {
			return err
		}

		entry = bookmng.NewCreateEntry(payment.ToTeamID, p.agent.GetUserID())
		err = entry.
			To(&accounting_core.EntryAccountPayload{
				Key:    accounting_core.CashAccount,
				TeamID: payment.ToTeamID,
			}, payment.Amount).
			From(&accounting_core.EntryAccountPayload{
				Key:    accounting_core.ReceivableAccount,
				TeamID: payment.FromTeamID,
			}, payment.Amount).
			Transaction(&tran).
			Commit().
			Err()

		if err != nil {
			return err
		}

		return nil
	})
}

func NewPaymentTransaction(ctx context.Context, tx *gorm.DB, agent identity_iface.Agent) PaymentTransaction {
	return &paymentPaymentTransactionImpl{
		ctx:   ctx,
		tx:    tx,
		agent: agent,
	}
}
