package revenue

import (
	"fmt"
	"math"
	"time"

	"github.com/pdcgo/accounting_service/accounting_core"
	"gorm.io/gorm"
)

type FundPayload struct {
	EstAmount float64
	Amount    float64
	At        time.Time
	Desc      string
	RefID     string
}

type WithdrawPayload struct {
	Amount float64
	At     time.Time
	Desc   string
	RefID  string
}

type Revenue interface {
	AddRevenue() error
	Fund(fund *FundPayload) error
	Adjustment() error
	Withdraw(wd *WithdrawPayload) error
}

type revenueImpl struct {
	db     *gorm.DB
	teamID uint
	userID uint
	shopID uint
}

// AddRevenue implements Revenue.
func (r *revenueImpl) AddRevenue() error {
	panic("unimplemented")
}

// Adjustment implements Revenue.
func (r *revenueImpl) Adjustment() error {
	panic("unimplemented")
}

// Fund implements Revenue.
func (r *revenueImpl) Fund(fund *FundPayload) error {
	var err error
	err = accounting_core.OpenTransaction(r.db, func(tx *gorm.DB, bookmng accounting_core.BookManage) error {
		refID := accounting_core.NewStringRefID(&accounting_core.StringRefData{
			RefType: accounting_core.OrderFundRef,
			ID:      fund.RefID,
		})

		tran := accounting_core.Transaction{
			RefID:       refID,
			Desc:        fund.Desc,
			TeamID:      r.teamID,
			CreatedByID: r.userID,
			Created:     fund.At,
		}

		err = bookmng.
			NewTransaction().
			Create(&tran).
			Err()

		if err != nil {
			return err
		}

		entry := bookmng.
			NewCreateEntry(r.teamID, r.userID)

		entry.
			From(&accounting_core.EntryAccountPayload{
				Key:    accounting_core.SellingEstReceivableAccount,
				TeamID: r.teamID,
			}, fund.EstAmount).
			To(&accounting_core.EntryAccountPayload{
				Key:    accounting_core.SellingReceivableAccount,
				TeamID: r.teamID,
			}, fund.Amount)

		if fund.EstAmount != fund.Amount {
			diffAmount := fund.EstAmount - fund.Amount
			if diffAmount > 0 {
				entry.
					To(&accounting_core.EntryAccountPayload{
						Key:    accounting_core.SellingAdjReceivableAccount,
						TeamID: r.teamID,
					}, diffAmount)
			}

			if diffAmount < 0 {
				entry.
					From(&accounting_core.EntryAccountPayload{
						Key:    accounting_core.SellingAdjReceivableAccount,
						TeamID: r.teamID,
					}, math.Abs(diffAmount))
			}
		}
		err = entry.
			To(&accounting_core.EntryAccountPayload{
				Key:    accounting_core.SellingAdjReceivableAccount,
				TeamID: r.teamID,
			}, fund.EstAmount-fund.Amount).
			Transaction(&tran).
			Commit(accounting_core.CustomTimeOption(fund.At)).
			Err()

		if err != nil {
			return err
		}

		return nil
	})

	return err
}

// Withdraw implements Revenue.
func (r *revenueImpl) Withdraw(wd *WithdrawPayload) error {
	var err error
	err = accounting_core.OpenTransaction(r.db, func(tx *gorm.DB, bookmng accounting_core.BookManage) error {
		refID := NewWithdrawRefID(&WithdrawRefData{
			RefType: accounting_core.WithdrawalRef,
			ShopID:  r.shopID,
			At:      wd.At,
		})

		tran := accounting_core.Transaction{
			RefID:       refID,
			TeamID:      r.teamID,
			CreatedByID: r.userID,
			Desc:        wd.Desc,
			Created:     wd.At,
		}
		err = bookmng.
			NewTransaction().
			Create(&tran).
			Err()

		if err != nil {
			return err
		}

		entry := bookmng.
			NewCreateEntry(r.teamID, r.userID)

		err = entry.
			From(&accounting_core.EntryAccountPayload{
				Key:    accounting_core.SellingReceivableAccount,
				TeamID: r.teamID,
			}, math.Abs(wd.Amount)).
			To(&accounting_core.EntryAccountPayload{
				Key:    accounting_core.CashAccount,
				TeamID: r.teamID,
			}, math.Abs(wd.Amount)).
			Transaction(&tran).
			Commit(accounting_core.CustomTimeOption(wd.At)).
			Err()

		return err
	})
	return err
}

// type RevenueConfig struct {
// 	TeamID uint
// 	ShopID uint
// 	Tags   []string
// }

func NewRevenue(db *gorm.DB, teamID uint, userID uint, shopID uint) Revenue {
	return &revenueImpl{
		db:     db,
		teamID: teamID,
		userID: userID,
		shopID: shopID,
	}
}

type WithdrawRefData struct {
	RefType accounting_core.RefType
	ShopID  uint
	At      time.Time
}

func NewWithdrawRefID(data *WithdrawRefData) accounting_core.RefID {
	raw := fmt.Sprintf("%s#%d#%s", data.RefType, data.ShopID, data.At.Format("2006-01-02#1500"))
	return accounting_core.RefID(raw)
}
