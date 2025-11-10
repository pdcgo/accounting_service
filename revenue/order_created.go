package revenue

import (
	"context"
	"fmt"
	"time"

	"connectrpc.com/connect"
	"github.com/pdcgo/accounting_service/accounting_core"
	"github.com/pdcgo/schema/services/revenue_iface/v1"
	"github.com/pdcgo/shared/interfaces/authorization_iface"
	"gorm.io/gorm"
)

// OnOrder implements revenue_ifaceconnect.RevenueServiceHandler.
func (r *revenueServiceImpl) OnOrder(
	ctx context.Context,
	req *connect.Request[revenue_iface.OnOrderRequest],
) (*connect.Response[revenue_iface.OnOrderResponse], error) {
	var err error
	res := connect.NewResponse(&revenue_iface.OnOrderResponse{})
	db := r.db.WithContext(ctx)

	// pecah payload
	pay := req.Msg
	labelInfo := pay.LabelInfo
	identity := r.
		auth.
		AuthIdentityFromToken(pay.Token)
	agent := identity.Identity()

	err = identity.Err()
	if err != nil {
		return res, err
	}

	switch pay.Event {
	case revenue_iface.OrderEvent_ORDER_EVENT_CREATED:
		err = accounting_core.OpenTransaction(db, func(tx *gorm.DB, bookmng accounting_core.BookManage) error {
			// creating transaction
			ref := accounting_core.NewRefID(&accounting_core.RefData{
				RefType: accounting_core.OrderRef,
				ID:      uint(pay.OrderId),
			})

			desc := fmt.Sprintf("order from %s", ref)
			ordInfo := &revenue_iface.OrderInfo{}
			if pay.OrderInfo != nil {
				ordInfo = pay.OrderInfo
				if ordInfo.ExternalOrderId != "" {
					desc = fmt.Sprintf("%s dengan orderid %s", desc, ordInfo.ExternalOrderId)
				}

				if ordInfo.Receipt != "" {
					desc = fmt.Sprintf("%s dengan resi %s", desc, ordInfo.Receipt)
				}

			}

			tran := accounting_core.Transaction{
				RefID:   ref,
				Desc:    desc,
				Created: time.Now(),
			}

			txcreate := bookmng.
				NewTransaction().
				Create(&tran).
				AddShopID(uint(labelInfo.ShopId)).
				AddCustomerServiceID(agent.IdentityID()).
				// AddTypeLabel([]*accounting_iface.TypeLabel{

				// }).
				AddTags(labelInfo.Tags)

			err = txcreate.
				Err()

			if err != nil {
				return err
			}

			// creating fee ke gudang
			err = r.createWarehouseFee(bookmng, agent, pay, &tran)
			if err != nil {
				return err
			}

			// own product
			err = r.ownProductStock(bookmng, agent, pay, &tran)
			if err != nil {
				return err
			}

			// cross product
			err = r.crossProductStock(bookmng, agent, pay, &tran)
			if err != nil {
				return err
			}

			// revenue order
			err = r.revenueOrder(bookmng, agent, pay, &tran)
			if err != nil {
				return err
			}

			return nil
		})

	}

	if err != nil {
		return res, err
	}

	return res, nil
}

func (r *revenueServiceImpl) ownProductStock(
	bookmng accounting_core.BookManage,
	agent authorization_iface.Identity,
	pay *revenue_iface.OnOrderRequest,
	tran *accounting_core.Transaction,
) error {
	var err error

	if pay.OwnStockAmount == 0 {
		return nil
	}

	// sisi selling
	err = bookmng.
		NewCreateEntry(uint(pay.TeamId), agent.GetUserID()).
		From(&accounting_core.EntryAccountPayload{
			Key:    accounting_core.StockReadyAccount,
			TeamID: uint(pay.WarehouseId),
		}, pay.OwnStockAmount).
		To(&accounting_core.EntryAccountPayload{
			Key:    accounting_core.StockCostAccount,
			TeamID: uint(pay.WarehouseId),
		}, pay.OwnStockAmount).
		Transaction(tran).
		Commit().
		Err()

	if err != nil {
		return err
	}

	// sisi gudang
	err = bookmng.
		NewCreateEntry(uint(pay.WarehouseId), agent.GetUserID()).
		From(&accounting_core.EntryAccountPayload{
			Key:    accounting_core.StockReadyAccount,
			TeamID: uint(pay.TeamId),
		}, pay.OwnStockAmount).
		To(&accounting_core.EntryAccountPayload{
			Key:    accounting_core.StockCostAccount,
			TeamID: uint(pay.TeamId),
		}, pay.OwnStockAmount).
		Transaction(tran).
		Err()

	if err != nil {
		return err
	}

	return nil
}

func (r *revenueServiceImpl) createWarehouseFee(
	bookmng accounting_core.BookManage,
	agent authorization_iface.Identity,
	pay *revenue_iface.OnOrderRequest,
	tran *accounting_core.Transaction,
) error {

	var err error

	// sisi gudang
	err = bookmng.
		NewCreateEntry(uint(pay.WarehouseId), agent.GetUserID()).
		To(&accounting_core.EntryAccountPayload{
			Key:    accounting_core.ServiceRevenueAccount, // revenue gudang [pendapatan jasa]
			TeamID: uint(pay.TeamId),
		}, pay.WarehouseFee).
		To(&accounting_core.EntryAccountPayload{
			Key:    accounting_core.ReceivableAccount,
			TeamID: uint(pay.TeamId),
		}, pay.WarehouseFee).
		Transaction(tran).
		Commit().
		Err()

	if err != nil {
		return err
	}

	// sisi selling
	err = bookmng.
		NewCreateEntry(uint(pay.TeamId), agent.GetUserID()).
		To(&accounting_core.EntryAccountPayload{
			Key:    accounting_core.WarehouseCostAccount,
			TeamID: uint(pay.WarehouseId),
		},
			pay.WarehouseFee).
		To(&accounting_core.EntryAccountPayload{
			Key:    accounting_core.PayableAccount,
			TeamID: uint(pay.WarehouseId),
		}, pay.WarehouseFee).
		Transaction(tran).
		Commit().
		Err()

	if err != nil {
		return err
	}

	return nil

}

func (r *revenueServiceImpl) crossProductStock(
	bookmng accounting_core.BookManage,
	agent authorization_iface.Identity,
	pay *revenue_iface.OnOrderRequest,
	tran *accounting_core.Transaction,
) error {
	var err error
	if len(pay.BorrowStock) == 0 {
		return nil
	}

	for _, bor := range pay.BorrowStock {
		// sisi gudang
		err = bookmng.
			NewCreateEntry(uint(pay.WarehouseId), agent.GetUserID()).
			From(&accounting_core.EntryAccountPayload{
				Key:    accounting_core.StockReadyAccount,
				TeamID: uint(bor.TeamId),
			}, bor.Amount).
			To(&accounting_core.EntryAccountPayload{
				Key:    accounting_core.StockToBorrowCostAccount,
				TeamID: uint(bor.TeamId),
			}, bor.Amount).
			Transaction(tran).
			Commit().
			Err()

		if err != nil {
			return err
		}

		// sisi dipinjami
		err = bookmng.
			NewCreateEntry(uint(bor.TeamId), agent.GetUserID()).
			From(&accounting_core.EntryAccountPayload{
				Key:    accounting_core.StockReadyAccount,
				TeamID: uint(pay.WarehouseId),
			}, bor.Amount).
			To(&accounting_core.EntryAccountPayload{
				Key:    accounting_core.StockToBorrowCostAccount,
				TeamID: uint(pay.WarehouseId),
			}, bor.Amount).
			To(&accounting_core.EntryAccountPayload{
				Key:    accounting_core.BorrowStockRevenueAccount,
				TeamID: uint(pay.TeamId),
			}, bor.SellAmount).
			To(&accounting_core.EntryAccountPayload{
				Key:    accounting_core.ReceivableAccount,
				TeamID: uint(pay.TeamId),
			}, bor.SellAmount).
			Transaction(tran).
			Commit().
			Err()

		if err != nil {
			return err
		}

		// sisi peminjam
		err = bookmng.
			NewCreateEntry(uint(pay.TeamId), agent.GetUserID()).
			To(&accounting_core.EntryAccountPayload{
				Key:    accounting_core.StockBorrowCostAccount,
				TeamID: uint(bor.TeamId),
			}, bor.SellAmount).
			To(&accounting_core.EntryAccountPayload{
				Key:    accounting_core.PayableAccount,
				TeamID: uint(bor.TeamId),
			}, bor.SellAmount).
			Transaction(tran).
			Commit().
			Err()

		if err != nil {
			return err
		}

	}

	return nil
}

func (r *revenueServiceImpl) revenueOrder(
	bookmng accounting_core.BookManage,
	agent authorization_iface.Identity,
	pay *revenue_iface.OnOrderRequest,
	tran *accounting_core.Transaction,
) error {
	// var err error

	err := bookmng.
		NewCreateEntry(uint(pay.TeamId), agent.GetUserID()).
		To(&accounting_core.EntryAccountPayload{
			Key:    accounting_core.SalesRevenueAccount,
			TeamID: uint(pay.TeamId),
		}, pay.OrderAmount).
		To(&accounting_core.EntryAccountPayload{
			Key:    accounting_core.SellingEstReceivableAccount,
			TeamID: uint(pay.TeamId),
		}, pay.OrderAmount).
		Transaction(tran).
		Commit().
		Err()

	if err != nil {
		return err
	}

	return nil
}
