package accounting_service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"connectrpc.com/connect"
	"github.com/pdcgo/accounting_service/accounting_core"
	"github.com/pdcgo/accounting_service/accounting_model"
	"github.com/pdcgo/accounting_service/accounting_transaction/expense_transaction"
	"github.com/pdcgo/schema/services/accounting_iface/v1"
	"github.com/pdcgo/schema/services/common/v1"
	"github.com/pdcgo/shared/db_models"
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

// ExpenseSetup implements accounting_ifaceconnect.ExpenseServiceHandler.
func (e *expenseServiceImpl) ExpenseSetup(
	ctx context.Context,
	req *connect.Request[accounting_iface.ExpenseSetupRequest],
) (*connect.Response[accounting_iface.ExpenseSetupResponse], error) {
	var err error
	result := connect.Response[accounting_iface.ExpenseSetupResponse]{
		Msg: &accounting_iface.ExpenseSetupResponse{},
	}
	pay := req.Msg
	if pay.TeamId == 0 {
		return &result, errors.New("team_id not set")
	}

	// getting team
	var team db_models.Team
	err = e.db.Model(&db_models.Team{}).First(&team, pay.TeamId).Error
	if err != nil {
		return &result, err
	}

	err = e.
		db.
		WithContext(ctx).
		Transaction(func(tx *gorm.DB) error {

			accounts := accounting_core.DefaultSeedAccount()
			for _, acc := range accounts {
				err = accounting_core.
					NewCreateAccount(tx).
					Create(
						acc.BalanceType,
						acc.Coa,
						uint(pay.TeamId),
						acc.AccountKey,
						fmt.Sprintf("%s (%s)", acc.AccountKey, team.Name),
					)

				if err != nil {
					return err
				}
			}

			return nil
		})

	return &result, err
}

// ExpenseCreate implements accounting_ifaceconnect.ExpenseServiceHandler.
func (e *expenseServiceImpl) ExpenseCreate(
	ctx context.Context,
	req *connect.Request[accounting_iface.ExpenseCreateRequest],
) (*connect.Response[accounting_iface.ExpenseCreateResponse], error) {
	var err error
	result := connect.Response[accounting_iface.ExpenseCreateResponse]{
		Msg: &accounting_iface.ExpenseCreateResponse{},
	}

	pay := req.Msg
	identity := e.auth.AuthIdentityFromHeader(req.Header())
	agent := identity.Identity()
	err = identity.
		HasPermission(authorization_iface.CheckPermissionGroup{
			accounting_model.ExpenseEntity{}: &authorization_iface.CheckPermission{
				DomainID: uint(pay.TeamId),
				Actions:  []authorization_iface.Action{authorization_iface.Create},
			},
		}).
		Err()

	if err != nil {
		return &result, err
	}

	err = e.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		exp := accounting_model.Expense{
			TeamID:      uint(pay.TeamId),
			CreatedByID: agent.GetUserID(),
			ExpenseType: pay.ExpenseType,
			ExpenseKey:  pay.ExpenseKey,
			Desc:        pay.Desc,
			Amount:      pay.Amount,
			ExpenseAt:   time.Now(),
			CreatedAt:   time.Now(),
		}

		err = tx.Save(&exp).Error
		if err != nil {
			return err
		}

		err = expense_transaction.
			NewExpenseTransaction(tx, identity.Identity()).
			ExpenseCreate(&expense_transaction.CreatePayload{
				TeamID:      uint(pay.TeamId),
				ExpenseKey:  accounting_core.AccountKey(pay.ExpenseKey),
				ExpenseType: pay.ExpenseType,
				Amount:      pay.Amount,
				Desc:        pay.Desc,
				RefID:       pay.RefId,
			})

		return err
	})

	return &result, err
}

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
	err = identity.
		HasPermission(authorization_iface.CheckPermissionGroup{
			accounting_model.ExpenseEntity{}: &authorization_iface.CheckPermission{
				DomainID: uint(pay.TeamId),
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
				Key:  string(accounting_core.KitchenExpenseAccount),
				Name: "Dapur",
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
