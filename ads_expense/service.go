package ads_expense

import (
	"context"

	"connectrpc.com/connect"
	"github.com/pdcgo/schema/services/accounting_iface/v1"
	"github.com/pdcgo/shared/interfaces/authorization_iface"
	"gorm.io/gorm"
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
