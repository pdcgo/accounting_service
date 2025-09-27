package payment

import (
	"context"
	"time"

	"connectrpc.com/connect"
	"github.com/pdcgo/accounting_service/accounting_model"
	"github.com/pdcgo/schema/services/common/v1"
	"github.com/pdcgo/schema/services/payment_iface/v1"
	"github.com/pdcgo/shared/interfaces/authorization_iface"
	"gorm.io/gorm"
)

type paymentServiceImpl struct {
	db   *gorm.DB
	auth authorization_iface.Authorization
}

// PaymentGet implements payment_ifaceconnect.PaymentServiceHandler.
func (p *paymentServiceImpl) PaymentGet(
	ctx context.Context,
	req *connect.Request[payment_iface.PaymentGetRequest],
) (*connect.Response[payment_iface.PaymentGetResponse], error) {
	panic("unimplemented")
}

// PaymentList implements payment_ifaceconnect.PaymentServiceHandler.
func (p *paymentServiceImpl) PaymentList(
	ctx context.Context,
	req *connect.Request[payment_iface.PaymentListRequest],
) (*connect.Response[payment_iface.PaymentListResponse], error) {
	var err error
	pay := req.Msg
	result := payment_iface.PaymentListResponse{
		Payments: []*payment_iface.Payment{},
		PageInfo: &common.PageInfo{},
	}

	db := p.db.WithContext(ctx)

	identity := p.
		auth.
		AuthIdentityFromHeader(req.Header())

	err = identity.
		HasPermission(authorization_iface.CheckPermissionGroup{
			&accounting_model.Payment{}: &authorization_iface.CheckPermission{
				DomainID: uint(pay.TeamId),
				Actions:  []authorization_iface.Action{authorization_iface.Read},
			},
		}).
		Err()

	if err != nil {
		return connect.NewResponse(&result), err
	}

	createQuery := func() *gorm.DB {
		query := db.
			Table("payments p")

		switch pay.PaymentType {
		case payment_iface.PaymentType_PAYMENT_TYPE_UNSPECIFIED:
		case payment_iface.PaymentType_PAYMENT_TYPE_OTHER, payment_iface.PaymentType_PAYMENT_TYPE_PRODUCT_CROSS:
			query = query.Where("p.payment_type = ?", pay.PaymentType)
		}

		switch pay.Source {
		case payment_iface.PaymentSource_PAYMENT_SOURCE_FROM:
			query = query.
				Where("p.from_team_id = ?", pay.TeamId)
		case payment_iface.PaymentSource_PAYMENT_SOURCE_TO:
			query = query.
				Where("p.to_team_id = ?", pay.TeamId)
		default:
			query = query.
				Where("p.from_team_id = ? or p.to_team_id = ?", pay.TeamId, pay.TeamId)
		}

		var startFieldTime, endFieldTime string
		switch pay.TimeFilterType {
		case payment_iface.PaymentTimeType_PAYMENT_TIME_TYPE_ACCEPTED:
			startFieldTime = "p.accepted_at > ?"
			endFieldTime = "p.accepted_at <= ?"
		case payment_iface.PaymentTimeType_PAYMENT_TIME_TYPE_CREATED:
			startFieldTime = "p.created_at > ?"
			endFieldTime = "p.created_at <= ?"
		default:
			startFieldTime = "p.created_at > ?"
			endFieldTime = "p.created_at <= ?"
		}

		if pay.TimeRange != nil {
			trange := pay.TimeRange
			if trange.EndDate != 0 {
				query = query.Where(endFieldTime,
					time.UnixMicro(trange.EndDate).Local(),
				)
			}

			if trange.StartDate != 0 {
				query = query.Where(startFieldTime,
					time.UnixMicro(trange.StartDate).Local(),
				)
			}
		}

		return query
	}

	// pagination
	var total int64
	page := pay.Page
	offset := (page.Page * page.Limit) - page.Limit

	err = createQuery().
		Select([]string{
			"count(1)",
		}).
		Find(&total).
		Error

	if err != nil {
		return connect.NewResponse(&result), err
	}

	result.PageInfo.CurrentPage = page.Page
	result.PageInfo.TotalPage = total / page.Limit
	if result.PageInfo.TotalPage == 0 {
		result.PageInfo.TotalPage = 1
	}
	result.PageInfo.TotalItems = total

	// get data
	err = createQuery().
		Select([]string{
			"p.id",
			"p.from_team_id",
			"p.to_team_id",

			"p.amount",
			"p.payment_type",
			"p.status",
			"(EXTRACT(EPOCH FROM p.created_at) * 1000000)::BIGINT AS created_at",
			"(EXTRACT(EPOCH FROM p.accepted_at) * 1000000)::BIGINT AS accepted_at",
		}).
		Offset(int(offset)).
		Limit(int(page.Limit)).
		Find(&result.Payments).
		Error

	if err != nil {
		return connect.NewResponse(&result), err
	}

	return connect.NewResponse(&result), nil
}

func NewPaymentService(db *gorm.DB, auth authorization_iface.Authorization) *paymentServiceImpl {
	return &paymentServiceImpl{
		db:   db,
		auth: auth,
	}
}
