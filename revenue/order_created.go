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
		err = db.Transaction(func(tx *gorm.DB) error {
			// creating transaction
			ref := accounting_core.NewRefID(&accounting_core.RefData{
				RefType: accounting_core.OrderRef,
				ID:      uint(pay.OrderId),
			})
			tran := accounting_core.Transaction{
				RefID:   ref,
				Desc:    fmt.Sprintf("order from %s", ref),
				Created: time.Now(),
			}

			txcreate := accounting_core.
				NewTransaction(tx).
				Create(&tran).
				AddShopID(uint(labelInfo.ShopId)).
				AddCustomerServiceID(agent.IdentityID()).
				AddTags(labelInfo.Tags)

			err = txcreate.
				Err()

			if err != nil {
				return err
			}

			// creating fee ke gudang
			err = r.createWarehouseFee(tx, agent, pay, &tran)
			if err != nil {
				return err
			}

			// own product
			err = r.ownProductStock(tx, agent, pay, &tran)
			if err != nil {
				return err
			}

			// cross product
			err = r.crossProductStock(tx, agent, pay, &tran)
			if err != nil {
				return err
			}

			// revenue order
			err = r.revenueOrder(tx, agent, pay, &tran)
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
	tx *gorm.DB,
	agent authorization_iface.Identity,
	pay *revenue_iface.OnOrderRequest,
	tran *accounting_core.Transaction,
) error {
	var err error

	if pay.OwnStockAmount == 0 {
		return nil
	}

	// sisi selling
	err = accounting_core.
		NewCreateEntry(tx, uint(pay.TeamId), agent.GetUserID()).
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
	err = accounting_core.
		NewCreateEntry(tx, uint(pay.WarehouseId), agent.GetUserID()).
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
	tx *gorm.DB,
	agent authorization_iface.Identity,
	pay *revenue_iface.OnOrderRequest,
	tran *accounting_core.Transaction,
) error {

	var err error

	// sisi gudang
	err = accounting_core.
		NewCreateEntry(tx, uint(pay.WarehouseId), agent.GetUserID()).
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
	err = accounting_core.
		NewCreateEntry(tx, uint(pay.TeamId), agent.GetUserID()).
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

func (r *revenueServiceImpl) revenueOrder(
	tx *gorm.DB,
	agent authorization_iface.Identity,
	pay *revenue_iface.OnOrderRequest,
	tran *accounting_core.Transaction,
) error {
	// var err error

	err := accounting_core.
		NewCreateEntry(tx, uint(pay.TeamId), agent.GetUserID()).
		To(&accounting_core.EntryAccountPayload{
			Key:    accounting_core.SalesRevenueAccount,
			TeamID: uint(pay.TeamId),
		}, pay.OrderAmount).
		To(&accounting_core.EntryAccountPayload{
			Key:    accounting_core.SellingReceivableAccount,
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
