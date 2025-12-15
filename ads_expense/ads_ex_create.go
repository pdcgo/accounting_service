package ads_expense

import (
	"context"
	"errors"
	"fmt"
	"time"

	"connectrpc.com/connect"
	"github.com/pdcgo/accounting_service/accounting_core"
	"github.com/pdcgo/schema/services/access_iface/v1"
	"github.com/pdcgo/schema/services/accounting_iface/v1"
	"github.com/pdcgo/schema/services/common/v1"
	"github.com/pdcgo/shared/authorization"
	"github.com/pdcgo/shared/custom_connect"
	"github.com/pdcgo/shared/interfaces/authorization_iface"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// AdsExCreate implements accounting_ifaceconnect.AdsExpenseServiceHandler.
func (a *adsExpenseImpl) AdsExCreate(
	ctx context.Context,
	req *connect.Request[accounting_iface.AdsExCreateRequest],
) (*connect.Response[accounting_iface.AdsExCreateResponse], error) {
	var err error

	pay := req.Msg
	identity := a.
		auth.
		AuthIdentityFromHeader(req.Header())
	agent := identity.Identity()

	source, err := custom_connect.GetRequestSource(ctx)
	if err != nil {
		return nil, err
	}

	var domainID uint
	switch source.RequestFrom {
	case access_iface.RequestFrom_REQUEST_FROM_ADMIN:
		domainID = authorization.RootDomain
	default:
		domainID = uint(pay.TeamId)
	}

	err = identity.
		HasPermission(authorization_iface.CheckPermissionGroup{
			&AdsExpense{}: &authorization_iface.CheckPermission{
				DomainID: domainID,
				Actions:  []authorization_iface.Action{authorization_iface.Create},
			},
		}).
		Err()

	if err != nil {
		return &connect.Response[accounting_iface.AdsExCreateResponse]{}, err
	}
	result := accounting_iface.AdsExCreateResponse{}

	db := a.db.WithContext(ctx)
	err = accounting_core.OpenTransaction(ctx, db, func(tx *gorm.DB, bookmng accounting_core.BookManage) error {
		var extRef string
		var tran accounting_core.Transaction

		switch pay.Source {
		case accounting_iface.AccountSource_ACCOUNT_SOURCE_SHOP:
			if pay.ExternalRefId == "" {
				return errors.New("source shop must have ext ref")
			}

			extRef = pay.ExternalRefId

		default:
			extRef = time.Now().String()
		}

		ref := accounting_core.NewStringRefID(&accounting_core.StringRefData{
			RefType: accounting_core.AdsPaymentRef,
			ID:      fmt.Sprintf("%d_%s", pay.ShopId, extRef),
		})

		txmut := accounting_core.
			NewTransactionMutation(ctx, tx).
			ByRefID(ref, true)

		err := txmut.Err()
		if err == nil {
			return accounting_core.ErrSkipTransaction
		} else {
			if !errors.Is(err, accounting_core.ErrTransactionNotFound) {
				return err
			}
		}

		tran = accounting_core.Transaction{
			TeamID:      uint(pay.TeamId),
			RefID:       ref,
			CreatedByID: agent.IdentityID(),
			Desc:        pay.Desc,
			Created:     time.Now(),
		}

		tags := []string{
			common.MarketplaceType_name[int32(pay.MpType)],
		}

		if len(pay.CustomTag) != 0 {
			tags = append(tags, pay.CustomTag...)
		}

		fixtags := []string{}
		for _, tname := range tags {
			tag := &accounting_core.AccountingTag{
				Name: accounting_core.SanityTag(tname),
			}
			err = tx.
				Clauses(
					clause.OnConflict{
						Columns:   []clause.Column{{Name: "name"}},
						DoNothing: true,
					},
				).
				Save(tag).
				Error
			if err != nil {
				return err
			}

			fixtags = append(fixtags, tag.Name)
		}

		err = bookmng.
			NewTransaction().
			Create(&tran).
			AddShopID(uint(pay.ShopId)).
			AddTags(fixtags).
			Err()

		if err != nil {
			return err
		}

		entry := bookmng.
			NewCreateEntry(uint(pay.TeamId), agent.IdentityID())

		switch pay.Source {
		case accounting_iface.AccountSource_ACCOUNT_SOURCE_SHOP:
			entry.From(&accounting_core.EntryAccountPayload{
				Key:    accounting_core.SellingReceivableAccount,
				TeamID: uint(pay.TeamId),
			}, pay.Amount)
		default:
			entry.
				From(&accounting_core.EntryAccountPayload{
					Key:    accounting_core.CashAccount,
					TeamID: uint(pay.TeamId),
				}, pay.Amount)
		}

		// bookeeping sellernya
		err = entry.
			To(&accounting_core.EntryAccountPayload{
				Key:    accounting_core.AdsExpenseAccount,
				TeamID: uint(pay.TeamId),
			}, pay.Amount).
			Transaction(&tran).
			Commit().
			Err()
		if err != nil {
			return err
		}

		result.TransactionId = uint64(tran.ID)

		return nil
	})

	return connect.NewResponse(&result), err

}
