package account

import (
	"context"
	"errors"
	"fmt"
	"time"

	"connectrpc.com/connect"
	"github.com/pdcgo/accounting_service/accounting_core"
	"github.com/pdcgo/accounting_service/accounting_model"
	"github.com/pdcgo/schema/services/accounting_iface/v1"
	"github.com/pdcgo/shared/interfaces/authorization_iface"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func (a *accountServiceImpl) TransferCreate(
	ctx context.Context,
	req *connect.Request[accounting_iface.TransferCreateRequest],
) (*connect.Response[accounting_iface.TransferCreateResponse], error) {
	var err error
	result := accounting_iface.TransferCreateResponse{}

	pay := req.Msg
	identity := a.
		auth.
		AuthIdentityFromHeader(req.Header())

	agent := identity.Identity()
	err = identity.
		HasPermission(authorization_iface.CheckPermissionGroup{
			&accounting_model.BankTransfer{}: &authorization_iface.CheckPermission{
				DomainID: uint(pay.TeamId),
				Actions:  []authorization_iface.Action{authorization_iface.Create},
			},
		}).
		Err()

	if err != nil {
		return connect.NewResponse(&result), err
	}
	db := a.db.WithContext(ctx)
	err = accounting_core.OpenTransaction(ctx, db, func(tx *gorm.DB, bookmng accounting_core.BookManage) error {
		var facc accounting_model.BankAccountV2
		var tacc accounting_model.BankAccountV2
		txclause := func() *gorm.DB {
			return tx.
				Clauses(clause.Locking{
					Strength: "UPDATE",
				})
		}

		err = txclause().
			First(&facc, pay.FromAccountId).
			Error
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return fmt.Errorf("account id %d not found", pay.FromAccountId)
			} else {
				return err
			}

		}

		if facc.TeamID != uint(pay.TeamId) {
			return fmt.Errorf("your team not have %s", facc.Name)
		}

		err = txclause().
			First(&tacc, pay.ToAccountId).
			Error
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return fmt.Errorf("account id %d not found", pay.ToAccountId)
			} else {
				return err
			}
		}

		hist := accounting_model.BankTransferHistory{
			TeamID:        uint(pay.TeamId),
			FromAccountID: uint(pay.FromAccountId),
			ToAccountID:   uint(pay.ToAccountId),
			// TxID:          trans.ID,
			Amount:    pay.Amount,
			FeeAmount: pay.FeeAmount,
			Desc:      pay.Desc,
			// TransferAt:    time.UnixMicro(pay.TransferAt).Local(),
			Created: time.Now(),
		}

		err = tx.Save(&hist).Error
		if err != nil {
			return err
		}

		if facc.ID == tacc.ID {
			return errors.New("account same")
		}
		ref := accounting_core.NewRefID(&accounting_core.RefData{
			RefType: accounting_core.TransferRef,
			ID:      hist.ID,
		})

		trans := accounting_core.Transaction{
			CreatedByID: agent.IdentityID(),
			TeamID:      uint(pay.TeamId),
			RefID:       ref,
			Desc:        pay.Desc,
			Created:     time.Now(),
		}

		err = bookmng.
			NewTransaction().
			Create(&trans).
			Err()

		if err != nil {
			return err
		}

		hist.TxID = trans.ID
		err = tx.Save(&hist).Error
		if err != nil {
			return err
		}

		entryopt := accounting_core.IncludeDebitCreditEqual()

		// book from
		entry := bookmng.
			NewCreateEntry(uint(pay.TeamId), agent.GetUserID()).
			From(&accounting_core.EntryAccountPayload{
				Key:    accounting_core.CashAccount,
				TeamID: facc.TeamID,
			}, pay.Amount+pay.FeeAmount).
			To(&accounting_core.EntryAccountPayload{
				Key:    accounting_core.CashAccount,
				TeamID: tacc.TeamID,
			}, pay.Amount)

		if pay.FeeAmount != 0 {
			entry.
				To(&accounting_core.EntryAccountPayload{
					Key:    accounting_core.BankFeeAccount,
					TeamID: facc.TeamID,
				}, pay.FeeAmount)
		}
		err = entry.
			Transaction(&trans).
			Commit(entryopt).
			Err()
		if err != nil {
			return err
		}

		// book to
		entry = bookmng.
			NewCreateEntry(tacc.TeamID, agent.GetUserID()).
			From(&accounting_core.EntryAccountPayload{
				Key:    accounting_core.CashAccount,
				TeamID: facc.TeamID,
			}, pay.Amount).
			To(&accounting_core.EntryAccountPayload{
				Key:    accounting_core.CashAccount,
				TeamID: tacc.TeamID,
			}, pay.Amount)

		err = entry.
			Transaction(&trans).
			Commit(entryopt).
			Err()
		if err != nil {
			return err
		}

		return nil
	})

	return connect.NewResponse(&result), err
}
