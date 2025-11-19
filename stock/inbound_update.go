package stock

import (
	"context"
	"errors"
	"fmt"

	"connectrpc.com/connect"
	"github.com/pdcgo/accounting_service/accounting_core"
	"github.com/pdcgo/schema/services/stock_iface/v1"
	"github.com/pdcgo/shared/db_models"
	"github.com/pdcgo/shared/interfaces/authorization_iface"
	"gorm.io/gorm"
)

// InboundUpdate implements stock_ifaceconnect.StockServiceHandler.
func (s *stockServiceImpl) InboundUpdate(
	ctx context.Context,
	req *connect.Request[stock_iface.InboundUpdateRequest],
) (*connect.Response[stock_iface.InboundUpdateResponse], error) {
	var err error

	pay := req.Msg
	result := stock_iface.InboundUpdateResponse{}

	identity := s.auth.AuthIdentityFromHeader(req.Header())
	agent := identity.Identity()
	err = identity.
		HasPermission(authorization_iface.CheckPermissionGroup{
			&db_models.InvTransaction{}: &authorization_iface.CheckPermission{DomainID: uint(pay.TeamId), Actions: []authorization_iface.Action{authorization_iface.Update}},
		}).
		Err()

	if err != nil {
		return connect.NewResponse(&result), err
	}
	db := s.db.WithContext(ctx)
	err = accounting_core.OpenTransaction(ctx, db, func(tx *gorm.DB, bookmng accounting_core.BookManage) error {
		var ref accounting_core.RefID
		switch pay.Source {
		case stock_iface.InboundSource_INBOUND_SOURCE_RESTOCK:
			ref = accounting_core.NewRefID(&accounting_core.RefData{
				RefType: accounting_core.RestockRef,
				ID:      uint(pay.ExtTxId),
			})
		case stock_iface.InboundSource_INBOUND_SOURCE_RETURN:
			ref = accounting_core.NewRefID(&accounting_core.RefData{
				RefType: accounting_core.StockReturnRef,
				ID:      uint(pay.ExtTxId),
			})
		default:
			return errors.New("source not supported for updating inbound")
		}

		txmut := accounting_core.
			NewTransactionMutation(ctx, tx).
			ByRefID(ref, true)

		err = txmut.Err()
		if err != nil {
			return err
		}

		if !txmut.IsExist() {
			return errors.New("transaction inbound update not found")
		}

		txdata := txmut.Data()

		// getting entry
		var oldentries accounting_core.JournalEntriesList
		err = tx.
			Model(&accounting_core.JournalEntry{}).
			Preload("Account").
			Where("team_id = ?", pay.TeamId).
			Where("transaction_id = ?", txdata.ID).
			Order("id asc").
			Find(&oldentries).
			Error

		if err != nil {
			return err
		}

		// create resetting entry
		entry := bookmng.NewCreateEntry(uint(pay.TeamId), agent.IdentityID())
		mapBalance, err := oldentries.AccountBalance()
		if err != nil {
			return err
		}

		entry.
			Rollback(mapBalance)

		var totalPayment float64

		if len(pay.Products) > 0 {
			var goodAmount float64
			for _, prod := range pay.Products {
				goodAmount += prod.ItemPrice * float64(prod.Count)
			}
			entry.To(&accounting_core.EntryAccountPayload{
				Key:    accounting_core.StockPendingAccount,
				TeamID: uint(pay.WarehouseId),
			}, goodAmount)

			totalPayment += goodAmount
		} else {
			ch, err := oldentries.AccountBalanceKey(accounting_core.StockPendingAccount)
			if err != nil {
				return err
			}
			entry.To(&accounting_core.EntryAccountPayload{
				Key:    accounting_core.StockPendingAccount,
				TeamID: uint(pay.WarehouseId),
			}, ch.Change())

			totalPayment += ch.Change()
		}

		if pay.ShippingFee != 0 {
			entry.To(&accounting_core.EntryAccountPayload{
				Key:    accounting_core.StockPendingFeeAccount,
				TeamID: uint(pay.WarehouseId),
			}, pay.ShippingFee)

			totalPayment += pay.ShippingFee

		} else {
			ch, _ := oldentries.AccountBalanceKey(accounting_core.StockPendingFeeAccount)
			c := ch.Change()
			if c != 0 {
				entry.To(&accounting_core.EntryAccountPayload{
					Key:    accounting_core.StockPendingFeeAccount,
					TeamID: uint(pay.WarehouseId),
				}, c)

				totalPayment += c
			}

		}

		switch pay.PaymentMethod {
		case stock_iface.PaymentMethod_PAYMENT_METHOD_CASH:
			entry.
				From(&accounting_core.EntryAccountPayload{
					Key:    accounting_core.CashAccount,
					TeamID: uint(pay.TeamId),
				}, totalPayment)
		case stock_iface.PaymentMethod_PAYMENT_METHOD_SHOPEEPAY:
			entry.
				From(&accounting_core.EntryAccountPayload{
					Key:    accounting_core.ShopeepayAccount,
					TeamID: uint(pay.TeamId),
				}, totalPayment)
		}

		err = entry.
			Transaction(txdata).
			Desc(fmt.Sprintf("update %s", txdata.RefID)).
			Commit().
			Err()

		return err
	})

	return connect.NewResponse(&result), err

}
