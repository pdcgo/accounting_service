package expense_transaction

import (
	"fmt"
	"time"

	"github.com/pdcgo/accounting_service/accounting_core"
	"github.com/pdcgo/schema/services/accounting_iface/v1"
	"github.com/pdcgo/shared/interfaces/identity_iface"
	"gorm.io/gorm"
)

type CreatePayload struct {
	TeamID      uint
	ExpenseKey  accounting_core.AccountKey
	ExpenseType accounting_iface.ExpenseType
	Amount      float64
	Desc        string
}

type ExpenseTransaction interface {
	ExpenseCreate(payload *CreatePayload) error
}

type expenseTransactonImpl struct {
	agent identity_iface.Agent
	tx    *gorm.DB
}

// ExpenseCreate implements ExpenseTransaction.
func (e *expenseTransactonImpl) ExpenseCreate(payload *CreatePayload) error {
	var err error

	ref := accounting_core.NewStringRefID(&accounting_core.StringRefData{
		RefType: accounting_core.ExpenseRef,
		ID:      fmt.Sprintf("%d-%s-%d", payload.TeamID, payload.ExpenseType, time.Now().Unix()),
	})

	var tran accounting_core.Transaction = accounting_core.Transaction{
		Desc:        payload.Desc,
		Created:     time.Now(),
		RefID:       ref,
		TeamID:      payload.TeamID,
		CreatedByID: e.agent.GetUserID(),
	}
	err = accounting_core.OpenTransaction(e.tx, func(tx *gorm.DB, bookmng accounting_core.BookManage) error {
		err = bookmng.
			NewTransaction().
			Create(&tran).
			Labels([]*accounting_core.Label{
				{
					Key:   accounting_core.TeamIDLabel,
					Value: fmt.Sprintf("%d", payload.TeamID),
				},
				{
					Key:   accounting_core.FromLabel,
					Value: fmt.Sprintf("%d", payload.ExpenseType),
				},
			}).
			Err()

		if err != nil {
			return err
		}

		err = bookmng.
			NewCreateEntry(payload.TeamID, e.agent.GetUserID()).
			From(&accounting_core.EntryAccountPayload{
				Key:    accounting_core.CashAccount,
				TeamID: payload.TeamID,
			}, payload.Amount).
			To(&accounting_core.EntryAccountPayload{
				Key:    payload.ExpenseKey,
				TeamID: payload.TeamID,
			}, payload.Amount).
			Transaction(&tran).
			Commit().
			Err()

		if err != nil {
			return err
		}

		return nil
	})

	return err
}

func NewExpenseTransaction(tx *gorm.DB, agent identity_iface.Agent) ExpenseTransaction {
	return &expenseTransactonImpl{
		agent: agent,
		tx:    tx,
	}
}
