package stock

import (
	"context"
	"errors"
	"fmt"
	"time"

	"connectrpc.com/connect"
	"github.com/pdcgo/accounting_service/accounting_core"
	"github.com/pdcgo/schema/services/stock_iface/v1"
	"github.com/pdcgo/shared/db_models"
	"github.com/pdcgo/shared/interfaces/authorization_iface"
	"gorm.io/gorm"
)

type stockServiceImpl struct {
	db   *gorm.DB
	auth authorization_iface.Authorization
}

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
	err = accounting_core.OpenTransaction(db, func(tx *gorm.DB, bookmng accounting_core.BookManage) error {
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
			NewTransactionMutation(tx).
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

// InboundAccept implements stock_ifaceconnect.StockServiceHandler.
func (s *stockServiceImpl) InboundAccept(
	ctx context.Context,
	req *connect.Request[stock_iface.InboundAcceptRequest],
) (*connect.Response[stock_iface.InboundAcceptResponse], error) {
	var err error

	pay := req.Msg
	result := &stock_iface.InboundAcceptResponse{}

	identity := s.auth.AuthIdentityFromHeader(req.Header())
	agent := identity.Identity()
	err = identity.
		HasPermission(authorization_iface.CheckPermissionGroup{
			&db_models.InvTransaction{}: &authorization_iface.CheckPermission{
				DomainID: uint(pay.WarehouseId),
				Actions:  []authorization_iface.Action{authorization_iface.Update},
			},
		}).
		Err()

	if err != nil {
		return connect.NewResponse(result), err
	}

	handle := inboundAccept{
		s:     s,
		ctx:   ctx,
		req:   req,
		agent: agent,
	}

	result, err = handle.accept()
	return connect.NewResponse(result), err
}

// InboundCreate implements stock_ifaceconnect.StockServiceHandler.
func (s *stockServiceImpl) InboundCreate(
	ctx context.Context,
	req *connect.Request[stock_iface.InboundCreateRequest],
) (*connect.Response[stock_iface.InboundCreateResponse], error) {
	var err error

	pay := req.Msg
	result := stock_iface.InboundCreateResponse{}

	identity := s.auth.AuthIdentityFromHeader(req.Header())
	agent := identity.Identity()
	err = identity.
		HasPermission(authorization_iface.CheckPermissionGroup{
			&db_models.InvTransaction{}: &authorization_iface.CheckPermission{
				DomainID: uint(pay.TeamId),
				Actions:  []authorization_iface.Action{authorization_iface.Create},
			},
		}).
		Err()

	if err != nil {
		return connect.NewResponse(&result), err
	}
	db := s.db.WithContext(ctx)
	ref := accounting_core.NewRefID(&accounting_core.RefData{
		RefType: accounting_core.RestockRef,
		ID:      uint(pay.ExtTxId),
	})

	var desc string
	var extra *stock_iface.StockInfoExtra
	if pay.Extras != nil {
		extra = pay.Extras
		if extra.ExternalOrderId != "" {
			desc = fmt.Sprintf("[%s] restock for %s", ref, extra.ExternalOrderId)
		} else if extra.Receipt != "" {
			desc = fmt.Sprintf("[%s] restock for %s", ref, extra.Receipt)
		}

	} else {
		desc = fmt.Sprintf("[%s] restock created", ref)
	}

	err = accounting_core.OpenTransaction(db, func(tx *gorm.DB, bookmng accounting_core.BookManage) error {
		tran := accounting_core.Transaction{
			TeamID:      uint(pay.TeamId),
			CreatedByID: agent.GetUserID(),
			Desc:        desc,
			Created:     time.Now(),
			RefID:       ref,
		}

		txcreate := bookmng.
			NewTransaction().
			Create(&tran)

		if extra != nil {
			if extra.CreatedById != 0 {
				txcreate.AddCustomerServiceID(uint(extra.CreatedById))
			}
			if len(extra.Tags) != 0 {
				txcreate.AddTags(extra.Tags)
			}
		}

		err = txcreate.
			Err()

		if err != nil {
			return err
		}

		var goodAmount float64
		for _, vary := range pay.Products {
			goodAmount += vary.ItemPrice * float64(vary.Count)
		}

		var totalAmount float64 = goodAmount + pay.ShippingFee

		// sisi selling
		entry := bookmng.NewCreateEntry(uint(pay.TeamId), agent.GetUserID())

		switch pay.PaymentMethod {
		case stock_iface.PaymentMethod_PAYMENT_METHOD_CASH:
			entry.
				From(&accounting_core.EntryAccountPayload{
					Key:    accounting_core.CashAccount,
					TeamID: uint(pay.TeamId),
				}, totalAmount)

		case stock_iface.PaymentMethod_PAYMENT_METHOD_SHOPEEPAY:
			entry.
				From(&accounting_core.EntryAccountPayload{
					Key:    accounting_core.ShopeepayAccount,
					TeamID: uint(pay.TeamId),
				}, totalAmount)
		}

		entry.
			To(&accounting_core.EntryAccountPayload{
				Key:    accounting_core.StockPendingAccount,
				TeamID: uint(pay.WarehouseId),
			}, goodAmount)

		if pay.ShippingFee > 0 {
			entry.
				To(&accounting_core.EntryAccountPayload{
					Key:    accounting_core.StockPendingFeeAccount,
					TeamID: uint(pay.WarehouseId),
				}, pay.ShippingFee)
		}

		err = entry.
			Transaction(&tran).
			Commit().
			Err()
		return err
	})

	return connect.NewResponse(&result), err
}

// StockAdjustment implements stock_ifaceconnect.StockServiceHandler.
func (s *stockServiceImpl) StockAdjustment(
	ctx context.Context,
	req *connect.Request[stock_iface.StockAdjustmentRequest],
) (*connect.Response[stock_iface.StockAdjustmentResponse], error) {
	var err error

	pay := req.Msg
	result := &stock_iface.StockAdjustmentResponse{}

	identity := s.auth.AuthIdentityFromHeader(req.Header())
	agent := identity.Identity()
	err = identity.
		HasPermission(authorization_iface.CheckPermissionGroup{
			&db_models.InvTransaction{}: &authorization_iface.CheckPermission{DomainID: uint(pay.TeamId), Actions: []authorization_iface.Action{authorization_iface.Update}},
		}).
		Err()

	if err != nil {
		return connect.NewResponse(result), err
	}

	handle := stockAdjustment{
		s:     s,
		ctx:   ctx,
		req:   req,
		agent: agent,
	}

	result, err = handle.adjustment()
	return connect.NewResponse(result), err

}

func NewStockService(db *gorm.DB, auth authorization_iface.Authorization) *stockServiceImpl {
	return &stockServiceImpl{
		db:   db,
		auth: auth,
	}
}
