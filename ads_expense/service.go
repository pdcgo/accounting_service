package ads_expense

import (
	"context"
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

type AdsExpense struct {
	ID uint
}

func (a *AdsExpense) GetEntityID() string {
	return "accounting/ads_expense"
}

type adsExpenseImpl struct {
	db   *gorm.DB
	auth authorization_iface.Authorization
}

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

	db := a.db.WithContext(ctx)
	err = accounting_core.OpenTransaction(ctx, db, func(tx *gorm.DB, bookmng accounting_core.BookManage) error {
		ref := accounting_core.NewStringRefID(&accounting_core.StringRefData{
			RefType: accounting_core.AdsPaymentRef,
			ID:      fmt.Sprintf("%d_%d", pay.ShopId, time.Now().Unix()),
		})
		tran := accounting_core.Transaction{
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

		err = bookmng.NewTransaction().
			Create(&tran).
			AddShopID(uint(pay.ShopId)).
			AddTags(fixtags).
			Err()

		if err != nil {
			return err
		}

		// bookeeping sellernya
		err = bookmng.
			NewCreateEntry(uint(pay.TeamId), agent.IdentityID()).
			From(&accounting_core.EntryAccountPayload{
				Key:    accounting_core.CashAccount,
				TeamID: uint(pay.TeamId),
			}, pay.Amount).
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

		return nil
	})

	return &connect.Response[accounting_iface.AdsExCreateResponse]{}, err

}

// AdsExEdit implements accounting_ifaceconnect.AdsExpenseServiceHandler.
func (a *adsExpenseImpl) AdsExEdit(context.Context, *connect.Request[accounting_iface.AdsExEditRequest]) (*connect.Response[accounting_iface.AdsExEditResponse], error) {
	panic("unimplemented")
}

// AdsExList implements accounting_ifaceconnect.AdsExpenseServiceHandler.
func (a *adsExpenseImpl) AdsExList(context.Context, *connect.Request[accounting_iface.AdsExListRequest]) (*connect.Response[accounting_iface.AdsExListResponse], error) {
	panic("unimplemented")
}

// AdsExOverviewMetric implements accounting_ifaceconnect.AdsExpenseServiceHandler.
func (a *adsExpenseImpl) AdsExOverviewMetric(context.Context, *connect.Request[accounting_iface.AdsExOverviewMetricRequest]) (*connect.Response[accounting_iface.AdsExOverviewMetricResponse], error) {
	panic("unimplemented")
}

// AdsExShopMetric implements accounting_ifaceconnect.AdsExpenseServiceHandler.
func (a *adsExpenseImpl) AdsExShopMetric(context.Context, *connect.Request[accounting_iface.AdsExShopMetricRequest]) (*connect.Response[accounting_iface.AdsExShopMetricResponse], error) {
	panic("unimplemented")
}

// AdsExTimeMetric implements accounting_ifaceconnect.AdsExpenseServiceHandler.
func (a *adsExpenseImpl) AdsExTimeMetric(context.Context, *connect.Request[accounting_iface.AdsExTimeMetricRequest]) (*connect.Response[accounting_iface.AdsExTimeMetricResponse], error) {
	panic("unimplemented")
}

func NewAdsExpenseService(db *gorm.DB, auth authorization_iface.Authorization) *adsExpenseImpl {
	return &adsExpenseImpl{
		db:   db,
		auth: auth,
	}
}
