package stock

import (
	"context"
	"fmt"
	"time"

	"connectrpc.com/connect"
	"github.com/pdcgo/accounting_service/accounting_core"
	"github.com/pdcgo/schema/services/accounting_iface/v1"
	"github.com/pdcgo/schema/services/stock_iface/v1"
	"github.com/pdcgo/shared/interfaces/authorization_iface"
	"gorm.io/gorm"
)

type inboundAccept struct {
	s     *stockServiceImpl
	ctx   context.Context
	req   *connect.Request[stock_iface.InboundAcceptRequest]
	agent authorization_iface.Identity
}

func (i *inboundAccept) extPriceFee() float64 {
	var totalGood int64
	pay := i.req.Msg
	if pay.ShippingFee == 0 && pay.WarehouseCodFee == 0 {
		return 0
	}

	for _, ac := range pay.Accepts {
		totalGood += ac.Count
	}

	for _, l := range pay.Losts {
		totalGood += l.Count
	}

	for _, b := range pay.Brokens {
		totalGood += b.Count
	}

	return (pay.ShippingFee + pay.WarehouseCodFee) / float64(totalGood)
}

func (i *inboundAccept) accept() (*stock_iface.InboundAcceptResponse, error) {
	var err error

	pay := i.req.Msg
	result := stock_iface.InboundAcceptResponse{}

	db := i.s.db.WithContext(i.ctx)
	err = accounting_core.OpenTransaction(i.ctx, db, func(tx *gorm.DB, bookmng accounting_core.BookManage) error {
		var ref accounting_core.RefID

		switch pay.Source {
		case stock_iface.InboundSource_INBOUND_SOURCE_RESTOCK:
			ref = accounting_core.NewRefID(&accounting_core.RefData{
				RefType: accounting_core.StockAcceptRef,
				ID:      uint(pay.ExtTxId),
			})
		case stock_iface.InboundSource_INBOUND_SOURCE_RETURN:
			ref = accounting_core.NewRefID(&accounting_core.RefData{
				RefType: accounting_core.StockReturnAcceptRef,
				ID:      uint(pay.ExtTxId),
			})
		case stock_iface.InboundSource_INBOUND_SOURCE_TRANSFER:
			ref = accounting_core.NewRefID(&accounting_core.RefData{
				RefType: accounting_core.StockTransferAcceptRef,
				ID:      uint(pay.ExtTxId),
			})
		default:
			return fmt.Errorf("%s not supported", pay.Source)
		}

		extra := &stock_iface.StockInfoExtra{}
		desc := fmt.Sprintf("stock diterima %s", ref)

		if pay.Extras != nil {
			extra = pay.Extras
			if extra.Receipt != "" {
				desc += fmt.Sprintf(" dengan resi %s", extra.Receipt)
			}
			if extra.ExternalOrderId != "" {
				desc += fmt.Sprintf(" dengan orderid %s", extra.ExternalOrderId)
			}
		}

		tran := accounting_core.Transaction{
			TeamID:      uint(pay.TeamId),
			RefID:       ref,
			CreatedByID: i.agent.IdentityID(),
			Desc:        desc,
			Created:     time.Now(),
		}

		err = bookmng.
			NewTransaction().
			Create(&tran).
			AddTags(extra.Tags).
			AddTypeLabel([]*accounting_iface.TypeLabel{
				{
					Key:   accounting_iface.LabelKey_LABEL_KEY_WAREHOUSE_TRANSACTION_TYPE,
					Label: stock_iface.InboundSource_name[int32(pay.Source)],
				},
			}).
			Err()

		if err != nil {
			return err
		}

		// sisi selling
		entrySel := bookmng.
			NewCreateEntry(uint(pay.TeamId), i.agent.IdentityID())

		// sisi gudang
		entryWare := bookmng.
			NewCreateEntry(uint(pay.WarehouseId), i.agent.IdentityID())

		if pay.ShippingFee != 0 {
			entryWare.
				From(&accounting_core.EntryAccountPayload{
					Key:    accounting_core.StockPendingAccount,
					TeamID: uint(pay.TeamId),
				}, pay.ShippingFee)

			entrySel.
				From(&accounting_core.EntryAccountPayload{
					Key:    accounting_core.StockPendingAccount,
					TeamID: uint(pay.WarehouseId),
				}, pay.ShippingFee)
		}

		if pay.WarehouseCodFee != 0 {
			entryWare.
				From(&accounting_core.EntryAccountPayload{
					Key:    accounting_core.CashAccount,
					TeamID: uint(pay.WarehouseId),
				}, pay.WarehouseCodFee).
				From(&accounting_core.EntryAccountPayload{
					Key:    accounting_core.StockCodFeeAccount,
					TeamID: uint(pay.WarehouseId),
				}, pay.WarehouseCodFee).
				To(&accounting_core.EntryAccountPayload{
					Key:    accounting_core.ReceivableAccount,
					TeamID: uint(pay.TeamId),
				}, pay.WarehouseCodFee)

			entrySel.
				To(&accounting_core.EntryAccountPayload{
					Key:    accounting_core.PayableAccount,
					TeamID: uint(pay.WarehouseId),
				}, pay.WarehouseCodFee)

		}

		var pending, warehouse_charge, accept, lost, broken, ext_price float64
		ext_price = i.extPriceFee()

		// accept
		for _, acc := range pay.Accepts {
			amount := (acc.ItemPrice + ext_price) * float64(acc.Count)
			accept += amount
			pending += acc.ItemPrice * float64(acc.Count)
		}

		// good lost
		if len(pay.Losts) != 0 {
			for _, l := range pay.Losts {
				amount := (l.ItemPrice + ext_price) * float64(l.Count)
				lost += amount
				pending += l.ItemPrice * float64(l.Count)
				if l.ChargeWarehouse {
					warehouse_charge += amount
				}
			}
		}

		// good broken
		if len(pay.Brokens) != 0 {
			for _, b := range pay.Brokens {
				amount := (b.ItemPrice + ext_price) * float64(b.Count)
				broken += amount
				pending += b.ItemPrice * float64(b.Count)
				if b.ChargeWarehouse {
					warehouse_charge += amount
				}
			}
		}

		entryWare.
			From(&accounting_core.EntryAccountPayload{
				Key:    accounting_core.StockPendingAccount,
				TeamID: uint(pay.TeamId),
			}, pending).
			To(&accounting_core.EntryAccountPayload{
				Key:    accounting_core.StockReadyAccount,
				TeamID: uint(pay.TeamId),
			}, accept)

		entrySel.
			From(&accounting_core.EntryAccountPayload{
				Key:    accounting_core.StockPendingAccount,
				TeamID: uint(pay.WarehouseId),
			}, pending).
			To(&accounting_core.EntryAccountPayload{
				Key:    accounting_core.StockReadyAccount,
				TeamID: uint(pay.WarehouseId),
			}, accept)

		if broken > 0 {
			entryWare.
				To(&accounting_core.EntryAccountPayload{
					Key:    accounting_core.StockBrokenAccount,
					TeamID: uint(pay.TeamId),
				}, broken)

			entrySel.
				To(&accounting_core.EntryAccountPayload{
					Key:    accounting_core.StockBrokenAccount,
					TeamID: uint(pay.WarehouseId),
				}, broken)
		}

		if lost > 0 {
			entryWare.
				To(&accounting_core.EntryAccountPayload{
					Key:    accounting_core.StockLostAccount,
					TeamID: uint(pay.TeamId),
				}, lost)

			entrySel.
				To(&accounting_core.EntryAccountPayload{
					Key:    accounting_core.StockLostAccount,
					TeamID: uint(pay.WarehouseId),
				}, lost)
		}

		// if warehouse_charge > 0 {
		// 	entryWare.
		// 		To(&accounting_core.EntryAccountPayload{
		// 			Key:    accounting_core.PayableAccount,
		// 			TeamID: uint(pay.TeamId),
		// 		}, warehouse_charge)

		// 	entrySel.
		// 		To(&accounting_core.EntryAccountPayload{
		// 			Key:    accounting_core.ReceivableAccount,
		// 			TeamID: uint(pay.WarehouseId),
		// 		}, warehouse_charge)
		// }

		err = entryWare.
			Transaction(&tran).
			Commit().
			Err()

		if err != nil {
			return err
		}

		err = entrySel.
			Transaction(&tran).
			Commit().
			Err()

		return err
	})

	return &result, err
}
