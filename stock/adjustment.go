package stock

import (
	"context"
	"fmt"
	"time"

	"connectrpc.com/connect"
	"github.com/pdcgo/accounting_service/accounting_core"
	"github.com/pdcgo/schema/services/stock_iface/v1"
	"github.com/pdcgo/shared/interfaces/authorization_iface"
	"gorm.io/gorm"
)

type stockAdjustment struct {
	s     *stockServiceImpl
	ctx   context.Context
	req   *connect.Request[stock_iface.StockAdjustmentRequest]
	agent authorization_iface.Identity
}

func (i *stockAdjustment) adjustment() (*stock_iface.StockAdjustmentResponse, error) {
	var err error
	result := stock_iface.StockAdjustmentResponse{}

	db := i.s.db.WithContext(i.ctx)
	pay := i.req.Msg

	err = accounting_core.OpenTransaction(db, func(tx *gorm.DB, bookmng accounting_core.BookManage) error {
		ref := accounting_core.NewRefID(&accounting_core.RefData{
			RefType: accounting_core.StockAdjustmentRef,
			ID:      uint(pay.ExtTxId),
		})

		tran := accounting_core.Transaction{
			TeamID:      uint(pay.TeamId),
			RefID:       ref,
			CreatedByID: i.agent.IdentityID(),
			Desc:        fmt.Sprintf("stock diterima %s", ref),
			Created:     time.Now(),
		}

		err = bookmng.
			NewTransaction().
			Create(&tran).
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

		var lost, broken, lostCharge, brokenCharge float64

		// good lost
		if len(pay.Losts) != 0 {
			for _, l := range pay.Losts {
				amount := l.ItemPrice * float64(l.Count)
				lost += amount
				if l.ChargeWarehouse {
					lostCharge += amount
				}
			}
		}

		// good broken
		if len(pay.Brokens) != 0 {
			for _, b := range pay.Brokens {
				amount := b.ItemPrice * float64(b.Count)
				broken += amount
				if b.ChargeWarehouse {
					brokenCharge += amount
				}
			}
		}

		if broken > 0 {
			entryWare.
				From(&accounting_core.EntryAccountPayload{
					Key:    accounting_core.StockReadyAccount,
					TeamID: uint(pay.TeamId),
				}, broken).
				To(&accounting_core.EntryAccountPayload{
					Key:    accounting_core.StockBrokenAccount,
					TeamID: uint(pay.TeamId),
				}, broken)

			entrySel.
				From(&accounting_core.EntryAccountPayload{
					Key:    accounting_core.StockReadyAccount,
					TeamID: uint(pay.WarehouseId),
				}, broken).
				To(&accounting_core.EntryAccountPayload{
					Key:    accounting_core.StockBrokenAccount,
					TeamID: uint(pay.WarehouseId),
				}, broken)
		}

		if lost > 0 {
			entryWare.
				From(&accounting_core.EntryAccountPayload{
					Key:    accounting_core.StockReadyAccount,
					TeamID: uint(pay.TeamId),
				}, broken).
				To(&accounting_core.EntryAccountPayload{
					Key:    accounting_core.StockLostAccount,
					TeamID: uint(pay.TeamId),
				}, broken)

			entrySel.
				From(&accounting_core.EntryAccountPayload{
					Key:    accounting_core.StockReadyAccount,
					TeamID: uint(pay.WarehouseId),
				}, broken).
				To(&accounting_core.EntryAccountPayload{
					Key:    accounting_core.StockLostAccount,
					TeamID: uint(pay.WarehouseId),
				}, broken)
		}

		if lostCharge > 0 {
			entryWare.
				To(&accounting_core.EntryAccountPayload{
					Key:    accounting_core.StockLostCostAccount,
					TeamID: uint(pay.TeamId),
				}, lostCharge).
				To(&accounting_core.EntryAccountPayload{
					Key:    accounting_core.PayableAccount,
					TeamID: uint(pay.TeamId),
				}, lostCharge)

			entrySel.
				From(&accounting_core.EntryAccountPayload{
					Key:    accounting_core.StockLostCostAccount,
					TeamID: uint(pay.WarehouseId),
				}, lostCharge).
				To(&accounting_core.EntryAccountPayload{
					Key:    accounting_core.ReceivableAccount,
					TeamID: uint(pay.WarehouseId),
				}, lostCharge)
		}

		if brokenCharge > 0 {
			entryWare.
				To(&accounting_core.EntryAccountPayload{
					Key:    accounting_core.StockBrokenCostAccount,
					TeamID: uint(pay.TeamId),
				}, brokenCharge).
				To(&accounting_core.EntryAccountPayload{
					Key:    accounting_core.PayableAccount,
					TeamID: uint(pay.TeamId),
				}, brokenCharge)

			entrySel.
				From(&accounting_core.EntryAccountPayload{
					Key:    accounting_core.StockBrokenCostAccount,
					TeamID: uint(pay.WarehouseId),
				}, brokenCharge).
				To(&accounting_core.EntryAccountPayload{
					Key:    accounting_core.ReceivableAccount,
					TeamID: uint(pay.WarehouseId),
				}, brokenCharge)
		}

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

		if err != nil {
			return err
		}

		return nil
	})

	return &result, err
}
