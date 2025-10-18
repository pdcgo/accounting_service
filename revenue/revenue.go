package revenue

import (
	"context"
	"fmt"
	"math"
	"time"

	"connectrpc.com/connect"
	"github.com/pdcgo/accounting_service/accounting_core"
	"github.com/pdcgo/schema/services/revenue_iface/v1"
	"gorm.io/gorm"
)

// RevenueStream implements revenue_ifaceconnect.RevenueServiceHandler.
func (r *revenueServiceImpl) RevenueStream(
	ctx context.Context,
	stream *connect.ClientStream[revenue_iface.RevenueStreamRequest],
) (*connect.Response[revenue_iface.RevenueStreamResponse], error) {
	var err error
	var isAuthenticated bool

	res := connect.NewResponse(&revenue_iface.RevenueStreamResponse{})
	processor := revenueProcessor{
		db:   r.db,
		init: &revenue_iface.RevenueStreamEventInit{},
	}

	for stream.Receive() {
		msg := stream.Msg()

		switch event := msg.Event.Kind.(type) {
		case *revenue_iface.RevenueStreamEvent_Init:
			err = r.
				auth.
				AuthIdentityFromToken(event.Init.Token).
				Err()

			if err != nil {
				return res, err
			}

			isAuthenticated = true
			processor.init = event.Init

		case *revenue_iface.RevenueStreamEvent_Fund:
			if !isAuthenticated {
				return res, fmt.Errorf("not authenticated")
			}

			err = processor.fund(event.Fund)
			if err != nil {
				return res, err
			}

		case *revenue_iface.RevenueStreamEvent_Adjustment:
			if !isAuthenticated {
				return res, fmt.Errorf("not authenticated")
			}

			err = processor.adjustment(event.Adjustment)
			if err != nil {
				return res, err
			}

		case *revenue_iface.RevenueStreamEvent_Withdrawal:
			if !isAuthenticated {
				return res, fmt.Errorf("not authenticated")
			}

			err = processor.withdrawal(event.Withdrawal)
			if err != nil {
				return res, err
			}
		}
	}

	return res, stream.Err()
}

type revenueProcessor struct {
	db   *gorm.DB
	init *revenue_iface.RevenueStreamEventInit
}

func (r *revenueProcessor) fund(fund *revenue_iface.RevenueStreamEventFund) error {
	var err error
	init := r.init
	var shopID uint = uint(init.ShopId)
	var teamID uint = uint(init.TeamId)
	var userID uint = uint(init.UserId)

	err = accounting_core.OpenTransaction(r.db, func(tx *gorm.DB, bookmng accounting_core.BookManage) error {
		refID := accounting_core.NewStringRefID(&accounting_core.StringRefData{
			RefType: accounting_core.OrderFundRef,
			ID:      fund.OrderId,
		})

		tran := accounting_core.Transaction{
			RefID:       refID,
			Desc:        fund.Desc,
			TeamID:      teamID,
			CreatedByID: userID,
			Created:     fund.At.AsTime(),
		}

		err = bookmng.
			NewTransaction().
			Create(&tran).
			AddShopID(shopID).
			Err()

		if err != nil {
			return err
		}

		entry := bookmng.
			NewCreateEntry(teamID, userID)

		entry.
			From(&accounting_core.EntryAccountPayload{
				Key:    accounting_core.SellingEstReceivableAccount,
				TeamID: teamID,
			}, fund.EstAmount).
			To(&accounting_core.EntryAccountPayload{
				Key:    accounting_core.SellingReceivableAccount,
				TeamID: teamID,
			}, fund.Amount)

		if fund.EstAmount != fund.Amount {
			diffAmount := fund.EstAmount - fund.Amount
			if diffAmount > 0 {
				entry.
					To(&accounting_core.EntryAccountPayload{
						Key:    accounting_core.SellingAdjReceivableAccount,
						TeamID: teamID,
					}, diffAmount)
			}

			if diffAmount < 0 {
				entry.
					From(&accounting_core.EntryAccountPayload{
						Key:    accounting_core.SellingAdjReceivableAccount,
						TeamID: teamID,
					}, math.Abs(diffAmount))
			}
		}
		err = entry.
			To(&accounting_core.EntryAccountPayload{
				Key:    accounting_core.SellingAdjReceivableAccount,
				TeamID: teamID,
			}, fund.EstAmount-fund.Amount).
			Transaction(&tran).
			Commit(accounting_core.CustomTimeOption(fund.At.AsTime())).
			Err()

		if err != nil {
			return err
		}

		return nil
	})

	return err
}

func (r *revenueProcessor) withdrawal(wd *revenue_iface.RevenueStreamEventWithdrawal) error {
	var err error
	init := r.init

	err = accounting_core.OpenTransaction(r.db, func(tx *gorm.DB, bookmng accounting_core.BookManage) error {
		refID := NewShopDateRefID(&WithdrawRefData{
			RefType: accounting_core.WithdrawalRef,
			ShopID:  uint(init.ShopId),
			At:      wd.At.AsTime(),
		})

		tran := accounting_core.Transaction{
			RefID:       refID,
			TeamID:      uint(init.TeamId),
			CreatedByID: uint(init.UserId),
			Desc:        wd.Desc,
			Created:     wd.At.AsTime(),
		}
		err = bookmng.
			NewTransaction().
			Create(&tran).
			Err()

		if err != nil {
			return err
		}

		entry := bookmng.
			NewCreateEntry(uint(init.TeamId), uint(init.UserId))

		teamID := uint(init.TeamId)

		err = entry.
			From(&accounting_core.EntryAccountPayload{
				Key:    accounting_core.SellingReceivableAccount,
				TeamID: teamID,
			}, math.Abs(wd.Amount)).
			To(&accounting_core.EntryAccountPayload{
				Key:    accounting_core.CashAccount,
				TeamID: teamID,
			}, math.Abs(wd.Amount)).
			Transaction(&tran).
			Commit(accounting_core.CustomTimeOption(wd.At.AsTime())).
			Err()

		return err
	})
	return err
}

func (r *revenueProcessor) adjustment(adj *revenue_iface.RevenueStreamEventAdjustment) error {
	var err error
	init := r.init
	var shopID uint = uint(init.ShopId)
	var teamID uint = uint(init.TeamId)
	var userID uint = uint(init.UserId)

	err = accounting_core.OpenTransaction(r.db, func(tx *gorm.DB, bookmng accounting_core.BookManage) error {
		refID := NewShopDateRefID(&WithdrawRefData{
			RefType: accounting_core.AdminAdjustmentRef,
			ShopID:  shopID,
			At:      adj.At.AsTime(),
		})

		tran := accounting_core.Transaction{
			RefID:       refID,
			Desc:        adj.Desc,
			TeamID:      teamID,
			CreatedByID: userID,
			Created:     adj.At.AsTime(),
		}

		err = bookmng.
			NewTransaction().
			Create(&tran).
			AddShopID(shopID).
			AddTags(adj.Tags).
			Err()

		if err != nil {
			return err
		}

		entry := bookmng.
			NewCreateEntry(teamID, userID)

		if adj.Amount > 0 {
			entry.
				From(&accounting_core.EntryAccountPayload{
					Key:    accounting_core.SellingEstReceivableAccount,
					TeamID: teamID,
				}, adj.Amount).
				To(&accounting_core.EntryAccountPayload{
					Key:    accounting_core.SellingReceivableAccount,
					TeamID: teamID,
				}, adj.Amount)

		} else if adj.Amount < 0 {
			entry.
				From(&accounting_core.EntryAccountPayload{
					Key:    accounting_core.SellingReceivableAccount,
					TeamID: teamID,
				}, math.Abs(adj.Amount)).
				To(&accounting_core.EntryAccountPayload{
					Key:    accounting_core.SellingAdjReceivableAccount,
					TeamID: teamID,
				}, math.Abs(adj.Amount))
		}

		err = entry.
			Transaction(&tran).
			Commit(accounting_core.CustomTimeOption(adj.At.AsTime())).
			Err()

		return err
	})

	return err
}

type WithdrawRefData struct {
	RefType accounting_core.RefType
	ShopID  uint
	At      time.Time
}

func NewShopDateRefID(data *WithdrawRefData) accounting_core.RefID {
	raw := fmt.Sprintf("%s#%d#%s", data.RefType, data.ShopID, data.At.Format("2006-01-02#1500"))
	return accounting_core.RefID(raw)
}
