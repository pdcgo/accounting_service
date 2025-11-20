package stock

import (
	"context"
	"fmt"
	"time"

	"connectrpc.com/connect"
	"github.com/pdcgo/accounting_service/accounting_core"
	"github.com/pdcgo/schema/services/accounting_iface/v1"
	"github.com/pdcgo/schema/services/stock_iface/v1"
	"github.com/pdcgo/shared/db_models"
	"github.com/pdcgo/shared/interfaces/authorization_iface"
	"gorm.io/gorm"
)

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

	err = accounting_core.OpenTransaction(ctx, db, func(tx *gorm.DB, bookmng accounting_core.BookManage) error {
		tran := accounting_core.Transaction{
			TeamID:      uint(pay.TeamId),
			CreatedByID: agent.GetUserID(),
			Desc:        desc,
			Created:     time.Now(),
			RefID:       ref,
		}

		txcreate := bookmng.
			NewTransaction().
			Create(&tran).
			AddTypeLabel(
				[]*accounting_iface.TypeLabel{
					{
						Key:   accounting_iface.LabelKey_LABEL_KEY_WAREHOUSE_TRANSACTION_TYPE,
						Label: stock_iface.InboundSource_name[int32(pay.Source)],
					},
				},
			)

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
					Key:    accounting_core.StockPendingAccount,
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
