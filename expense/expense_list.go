package expense

import (
	"context"
	"errors"

	"connectrpc.com/connect"
	"github.com/pdcgo/accounting_service/accounting_model"
	"github.com/pdcgo/schema/services/accounting_iface/v1"
	"github.com/pdcgo/schema/services/common/v1"
	"github.com/pdcgo/shared/authorization"
	"github.com/pdcgo/shared/interfaces/authorization_iface"
	"gorm.io/gorm"
)

// ExpenseList implements accounting_ifaceconnect.ExpenseServiceHandler.
func (e *expenseServiceImpl) ExpenseList(
	ctx context.Context,
	req *connect.Request[accounting_iface.ExpenseListRequest],
) (*connect.Response[accounting_iface.ExpenseListResponse], error) {
	var err error
	result := &accounting_iface.ExpenseListResponse{
		Data:     []*accounting_iface.ExpenseItem{},
		PageInfo: &common.PageInfo{},
	}
	res := &connect.Response[accounting_iface.ExpenseListResponse]{
		Msg: result,
	}

	// checking payload
	pay := req.Msg

	if pay.TimeRange == nil {
		return res, errors.New("time range must set")
	}

	if pay.Page == nil {
		return res, errors.New("page must set")
	}

	if pay.Page.Limit == 0 {
		pay.Page.Limit = 100
	}
	if pay.Page.Page == 0 {
		pay.Page.Page = 1
	}

	identity := e.auth.AuthIdentityFromHeader(req.Header())
	var domainID uint
	switch pay.RequestFrom {
	case common.RequestFrom_REQUEST_FROM_ADMIN:
		domainID = authorization.RootDomain
	case common.RequestFrom_REQUEST_FROM_SELLING, common.RequestFrom_REQUEST_FROM_WAREHOUSE:
		domainID = uint(pay.TeamId)

	}
	err = identity.
		HasPermission(authorization_iface.CheckPermissionGroup{
			&accounting_model.ExpenseEntity{}: &authorization_iface.CheckPermission{
				DomainID: domainID,
				Actions:  []authorization_iface.Action{authorization_iface.Read},
			},
		}).
		Err()

	if err != nil {
		return res, err
	}

	db := e.db.WithContext(ctx)
	createQuery := func() *gorm.DB {
		query := db.
			Table("expenses e").
			Select([]string{
				"e.id",
				"e.team_id",
				"e.created_by_id",
				"e.desc",
				"e.expense_type",
				"e.amount",
				"(EXTRACT(EPOCH FROM e.expense_at) * 1000000)::BIGINT AS expense_at",
				"(EXTRACT(EPOCH FROM e.created_at) * 1000000)::BIGINT AS created_at",
			})

		if pay.TeamId != 0 {
			query = query.Where("e.team_id = ?", pay.TeamId)
		}

		if pay.ByUserId != 0 {
			query = query.Where("e.created_by_id = ?", pay.ByUserId)
		}

		if pay.TimeRange.StartDate.IsValid() {
			query = query.Where("e.expense_at > ?",
				pay.TimeRange.StartDate.AsTime(),
			)
		}

		if pay.TimeRange.EndDate.IsValid() {
			query = query.Where("e.expense_at <= ?",
				pay.TimeRange.EndDate.AsTime(),
			)
		}

		if pay.ExpenseType != accounting_iface.ExpenseType_EXPENSE_TYPE_UNSPECIFIED {
			query = query.Where("e.expense_type = ?", pay.ExpenseType)
		}

		return query
	}

	query := createQuery()

	page := pay.Page.Page
	offset := (page - 1) * pay.Page.Limit
	err = query.
		Offset(int(offset)).
		Limit(int(pay.Page.Limit)).
		Find(&result.Data).Error
	if err != nil {
		return res, err
	}

	// paginasi belum implement
	var itemcount int64

	query = createQuery()
	err = query.Count(&itemcount).Error
	if err != nil {
		return res, err
	}
	var total int64 = int64(itemcount / pay.Page.Limit)
	if total == 0 {
		total = 1
	}

	result.PageInfo = &common.PageInfo{
		CurrentPage: pay.Page.Page,
		TotalPage:   total,
		TotalItems:  itemcount,
	}

	return res, nil
}
