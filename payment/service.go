package payment

import (
	"context"

	"connectrpc.com/connect"
	"github.com/pdcgo/schema/services/payment_iface/v1"
	"github.com/pdcgo/shared/interfaces/authorization_iface"
	"gorm.io/gorm"
)

type paymentServiceImpl struct {
	db   *gorm.DB
	auth authorization_iface.Authorization
}

// PaymentGet implements payment_ifaceconnect.PaymentServiceHandler.
func (p *paymentServiceImpl) PaymentGet(context.Context, *connect.Request[payment_iface.PaymentGetRequest]) (*connect.Response[payment_iface.PaymentGetResponse], error) {
	panic("unimplemented")
}

// PaymentList implements payment_ifaceconnect.PaymentServiceHandler.
func (p *paymentServiceImpl) PaymentList(context.Context, *connect.Request[payment_iface.PaymentListRequest]) (*connect.Response[payment_iface.PaymentListResponse], error) {
	panic("unimplemented")
}

func NewPaymentService(db *gorm.DB, auth authorization_iface.Authorization) *paymentServiceImpl {
	return &paymentServiceImpl{
		db:   db,
		auth: auth,
	}
}
