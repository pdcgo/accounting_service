package revenue

import (
	"context"
	"fmt"
	"time"

	"connectrpc.com/connect"
	"github.com/pdcgo/accounting_service/accounting_core"
	"github.com/pdcgo/schema/services/revenue_iface/v1"
	"gorm.io/gorm"
)

// Withdrawal implements revenue_ifaceconnect.RevenueServiceHandler.
func (r *revenueServiceImpl) Withdrawal(
	ctx context.Context,
	req *connect.Request[revenue_iface.WithdrawalRequest],
) (*connect.Response[revenue_iface.WithdrawalResponse], error) {
	var err error
	res := connect.NewResponse(&revenue_iface.WithdrawalResponse{})
	db := r.db.WithContext(ctx)
	pay := req.Msg

	identity := r.
		auth.
		AuthIdentityFromHeader(req.Header())
	agent := identity.Identity()

	err = identity.Err()
	if err != nil {
		return res, err
	}

	refID := NewShopDateRefID(&WithdrawRefData{
		RefType: accounting_core.WithdrawalRef,
		ShopID:  uint(pay.ShopId),
		At:      pay.At.AsTime(),
	})

	var exist bool
	exist, err = r.checkTxExist(refID)
	if err != nil {
		return nil, err
	}

	if exist {
		return &connect.Response[revenue_iface.WithdrawalResponse]{}, nil
	}

	err = accounting_core.OpenTransaction(ctx, db, func(tx *gorm.DB, bookmng accounting_core.BookManage) error {

		tran := accounting_core.Transaction{
			RefID:       refID,
			TeamID:      uint(pay.TeamId),
			CreatedByID: agent.IdentityID(),
			Desc:        fmt.Sprintf("%s %s", refID, pay.Desc),
			Created:     time.Now(),
		}
		err = bookmng.
			NewTransaction().
			Create(&tran).
			AddShopID(uint(pay.ShopId)).
			Err()

		if err != nil {
			return err
		}

		entry := bookmng.
			NewCreateEntry(uint(pay.TeamId), agent.IdentityID())

		teamID := uint(pay.TeamId)

		err = entry.
			From(&accounting_core.EntryAccountPayload{
				Key:    accounting_core.SellingReceivableAccount,
				TeamID: teamID,
			}, pay.Amount).
			To(&accounting_core.EntryAccountPayload{
				Key:    accounting_core.CashAccount,
				TeamID: teamID,
			}, pay.Amount).
			Transaction(&tran).
			Commit().
			Err()

		return err
	})

	return &connect.Response[revenue_iface.WithdrawalResponse]{}, err
}

func (r *revenueServiceImpl) checkTxExist(refID accounting_core.RefID) (bool, error) {
	var err error
	var txID uint

	err = r.
		db.
		Model(&accounting_core.Transaction{}).
		Where("ref_id = ?", refID).
		Select("id").
		Find(&txID).
		Error

	if err != nil {
		return false, err
	}

	if txID != 0 {
		return true, nil
	}

	return false, err
}
