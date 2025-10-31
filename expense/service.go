package expense

import (
	"context"
	"errors"
	"time"

	"connectrpc.com/connect"
	"github.com/pdcgo/accounting_service/accounting_core"
	"github.com/pdcgo/accounting_service/accounting_model"
	"github.com/pdcgo/schema/services/accounting_iface/v1"
	"github.com/pdcgo/schema/services/common/v1"
	"github.com/pdcgo/shared/interfaces/authorization_iface"
	"gorm.io/gorm"
)

type KeyValue struct {
	T     time.Time
	Key   string
	Value float64
}

type expenseServiceImpl struct {
	db   *gorm.DB
	auth authorization_iface.Authorization
}

// ExpenseTimeMetric implements accounting_ifaceconnect.ExpenseServiceHandler.
func (e *expenseServiceImpl) ExpenseTimeMetric(
	ctx context.Context,
	req *connect.Request[accounting_iface.ExpenseTimeMetricRequest],
) (*connect.Response[accounting_iface.ExpenseTimeMetricResponse], error) {
	var err error
	result := &accounting_iface.ExpenseTimeMetricResponse{
		ExpenseTotal: 0,
		Data:         map[int64]*accounting_iface.KeyValueMetricList{},
	}
	res := &connect.Response[accounting_iface.ExpenseTimeMetricResponse]{
		Msg: result,
	}
	pay := req.Msg
	identity := e.auth.AuthIdentityFromHeader(req.Header())
	err = identity.
		HasPermission(authorization_iface.CheckPermissionGroup{
			accounting_model.ExpenseEntity{}: &authorization_iface.CheckPermission{DomainID: uint(pay.TeamId), Actions: []authorization_iface.Action{authorization_iface.Read}},
		}).
		Err()
	if err != nil {
		return res, err
	}

	if pay.TimeRange == nil {
		return res, errors.New("specify time range")
	}

	db := e.
		db.
		WithContext(ctx)

	var tfield string
	switch pay.TimeType {
	case common.TimeType_TIME_TYPE_DAILY:
		tfield = "DATE_TRUNC('day', je.entry_time AT TIME ZONE 'Asia/Jakarta') as t"
	case common.TimeType_TIME_TYPE_MONTHLY:
		tfield = "DATE_TRUNC('month', je.entry_time AT TIME ZONE 'Asia/Jakarta') as t"
	case common.TimeType_TIME_TYPE_YEARLY:
		tfield = "DATE_TRUNC('year', je.entry_time AT TIME ZONE 'Asia/Jakarta') as t"
	default:
		tfield = "DATE_TRUNC('day', je.entry_time AT TIME ZONE 'Asia/Jakarta') as t"
	}

	query := db.
		Table("journal_entries je").
		Select([]string{
			tfield, // field mode
			"a.account_key as key",
			"sum(je.debit - je.credit) as value",
		}).
		Joins("JOIN accounts a on a.id = je.account_id").
		Where("a.coa = ?", accounting_core.EXPENSE).
		Group("a.account_key, t")

	if pay.TeamId != 0 {
		query = query.Where("je.team_id = ?", pay.TeamId)
	}

	if pay.TimeRange.EndDate.IsValid() {
		query = query.Where("je.entry_time <= ?",
			pay.TimeRange.EndDate.AsTime(),
		)
	}

	if pay.TimeRange.StartDate.IsValid() {
		query = query.Where("je.entry_time > ?",
			pay.TimeRange.StartDate.AsTime(),
		)
	}
	items := []*KeyValue{}
	err = query.Find(&items).Error
	if err != nil {
		return res, err
	}

	for _, item := range items {
		result.ExpenseTotal += item.Value
		key := item.T.UnixMicro()
		if result.Data[key] == nil {
			result.Data[key] = &accounting_iface.KeyValueMetricList{
				Data: []*accounting_iface.KeyValueMetric{},
			}
		}

		result.Data[key].Data = append(result.Data[key].Data, &accounting_iface.KeyValueMetric{
			Key:   item.Key,
			Value: item.Value,
		})
	}

	return res, err
}

// ExpenseOverviewMetric implements accounting_ifaceconnect.ExpenseServiceHandler.
func (e *expenseServiceImpl) ExpenseOverviewMetric(
	ctx context.Context,
	req *connect.Request[accounting_iface.ExpenseOverviewMetricRequest],
) (*connect.Response[accounting_iface.ExpenseOverviewMetricResponse], error) {
	var err error
	result := &accounting_iface.ExpenseOverviewMetricResponse{
		ExpenseDetails: map[string]float64{},
	}
	res := &connect.Response[accounting_iface.ExpenseOverviewMetricResponse]{
		Msg: result,
	}

	pay := req.Msg
	identity := e.auth.AuthIdentityFromHeader(req.Header())
	err = identity.
		HasPermission(authorization_iface.CheckPermissionGroup{
			accounting_model.ExpenseEntity{}: &authorization_iface.CheckPermission{DomainID: uint(pay.TeamId), Actions: []authorization_iface.Action{authorization_iface.Read}},
		}).
		Err()

	if err != nil {
		return res, err
	}

	db := e.
		db.
		WithContext(ctx)

	items := []*KeyValue{}

	query := db.
		Table("journal_entries je").
		Select([]string{
			"a.account_key as key",
			"sum(je.debit - je.credit) as value",
		}).
		Joins("JOIN accounts a on a.id = je.account_id").
		Where("a.coa = ?", accounting_core.EXPENSE).
		Group("a.account_key")

	if pay.TeamId != 0 {
		query = query.Where("je.team_id = ?", pay.TeamId)
	}

	if pay.TimeRange.EndDate.IsValid() {
		query = query.Where("je.entry_time <= ?",
			pay.TimeRange.EndDate.AsTime(),
		)
	}

	if pay.TimeRange.StartDate.IsValid() {
		query = query.Where("je.entry_time > ?",
			pay.TimeRange.StartDate.AsTime(),
		)
	}

	err = query.Find(&items).Error
	if err != nil {
		return res, err
	}

	for _, item := range items {
		result.ExpenseTotal += item.Value
		result.ExpenseDetails[item.Key] = item.Value
	}

	return res, err
}

// ExpenseTypeList implements accounting_ifaceconnect.ExpenseServiceHandler.
func (e *expenseServiceImpl) ExpenseTypeList(
	ctx context.Context,
	req *connect.Request[accounting_iface.ExpenseTypeListRequest],
) (*connect.Response[accounting_iface.ExpenseTypeListResponse], error) {
	var err error
	var result connect.Response[accounting_iface.ExpenseTypeListResponse] = connect.Response[accounting_iface.ExpenseTypeListResponse]{
		Msg: &accounting_iface.ExpenseTypeListResponse{},
	}

	data := req.Msg

	switch data.Type {
	case accounting_iface.ExpenseType_EXPENSE_TYPE_INTERNAL,
		accounting_iface.ExpenseType_EXPENSE_TYPE_SELLING:
		return &result, nil
	case accounting_iface.ExpenseType_EXPENSE_TYPE_WAREHOUSE:
		result.Msg.Data = []*common.KeyName{
			{
				Key:  string(accounting_core.OwnerAccommodationAccount),
				Name: "Akomodasi Owner",
			},
			{
				Key:  string(accounting_core.KitchenExpenseAccount),
				Name: "Dapur",
			},
			{
				Key:  string(accounting_core.SalaryAccount),
				Name: "Gaji",
			},
			{
				Key:  string(accounting_core.InternetConnectionAccount),
				Name: "Internet",
			},
			{
				Key:  string(accounting_core.PackingExpenseAccount),
				Name: "Packing",
			},
			{
				Key:  string(accounting_core.ElectricityExpenseAccount),
				Name: "Listrik",
			},
			{
				Key:  string(accounting_core.EquipmentExpenseAccount),
				Name: "Peralatan",
			},
			{
				Key:  string(accounting_core.ToolExpenseAccount),
				Name: "Tools",
			},
			{
				Key:  string(accounting_core.TransportExpenseAccount),
				Name: "Transportasi",
			},
			{
				Key:  string(accounting_core.ServerExpenseAccount),
				Name: "Server",
			},
			{
				Key:  string(accounting_core.OtherExpenseAccount),
				Name: "Lain-Lain",
			},
		}
	default:
		return &result, errors.New("expense type unspecified or supported")
	}

	return &result, err
}

func NewExpenseService(db *gorm.DB, auth authorization_iface.Authorization) *expenseServiceImpl {
	return &expenseServiceImpl{
		db:   db,
		auth: auth,
	}
}
