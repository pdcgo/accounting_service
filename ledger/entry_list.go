package ledger

import (
	"context"
	"fmt"
	"math"
	"strings"

	"connectrpc.com/connect"
	"github.com/pdcgo/accounting_service/accounting_core"
	"github.com/pdcgo/schema/services/accounting_iface/v1"
	"github.com/pdcgo/schema/services/common/v1"
	"github.com/pdcgo/shared/db_connect"
	"github.com/pdcgo/shared/pkg/ware_cache"
	"gorm.io/gorm"
)

func CachePaginated(ctx context.Context, cache ware_cache.Cache, pagination *common.PageFilter, info *common.PageInfo) db_connect.NextHandler {
	return func(db *gorm.DB, next db_connect.NextFunc) db_connect.NextFunc {
		return func(query *gorm.DB) (*gorm.DB, error) {
			var err error
			// // getting sql string
			// stmt := query.
			// 	Session(&gorm.Session{DryRun: true}).
			// 	Find(&map[string]any{}).
			// 	Statement
			// sql := db.Dialector.Explain(stmt.SQL.String(), stmt.Vars...)

			// sum := md5.Sum([]byte(sql))
			// sqlhash := hex.EncodeToString(sum[:])
			// key := fmt.Sprintf("sql/%s", sqlhash)
			// err := cache.Get(ctx, key, info)

			// if err == nil {
			// 	log.Println("pagination using cache")
			// 	offset := (pagination.Page - 1) * pagination.Limit
			// 	return next(
			// 		query.
			// 			Offset(int(offset)).
			// 			Limit(int(pagination.Limit)),
			// 	)
			// }

			// log.Println(err)

			var queryPaginated *gorm.DB
			pageResult := &common.PageInfo{}

			queryPaginated, pageResult, err = db_connect.SetPaginationQuery(db, func() (*gorm.DB, error) {
				return query.Session(&gorm.Session{}), nil

			}, pagination)

			if err != nil {
				return query, err
			}
			info.TotalItems = pageResult.TotalItems
			info.CurrentPage = pageResult.CurrentPage
			info.TotalPage = pageResult.TotalPage

			// err = cache.Add(ctx, &ware_cache.CacheItem{
			// 	Key:        key,
			// 	Expiration: time.Second,
			// 	Data:       pageResult,
			// })

			// if err != nil {
			// 	return query, err
			// }

			return next(queryPaginated)
		}
	}
}

// EntryList implements accounting_ifaceconnect.LedgerServiceHandler.
func (l *ledgerServiceImpl) EntryList(
	ctx context.Context,
	req *connect.Request[accounting_iface.EntryListRequest],
) (*connect.Response[accounting_iface.EntryListResponse], error) {
	var err error
	result := accounting_iface.EntryListResponse{
		Data:     []*accounting_iface.EntryItem{},
		PageInfo: &common.PageInfo{},
	}

	pay := req.Msg

	db := l.db.WithContext(ctx)
	err = l.
		auth.
		AuthIdentityFromHeader(req.Header()).
		Err()

	if err != nil {
		return connect.NewResponse(&result), err
	}

	if pay.Marketplace != common.MarketplaceType_MARKETPLACE_TYPE_UNSPECIFIED {
		pay.Label = append(pay.Label, &accounting_iface.TypeLabelFilter{
			Label: &accounting_iface.TypeLabelFilter_Marketplace{
				Marketplace: pay.Marketplace,
			},
		})
	}

	query, err := db_connect.NewQueryChain(db,
		func(db *gorm.DB, next db_connect.NextFunc) db_connect.NextFunc { // creating base
			return func(query *gorm.DB) (*gorm.DB, error) {
				return next(
					query.
						Table("journal_entries je").
						Joins("join accounts a on a.id = je.account_id"),
				)

			}
		},
		func(db *gorm.DB, next db_connect.NextFunc) db_connect.NextFunc {
			return func(query *gorm.DB) (*gorm.DB, error) { // filter team id
				if pay.TeamId == 0 {
					return next(query)
				}

				tid := uint(pay.TeamId)
				return next(
					query.
						Where("je.team_id = ?", tid),
				)
			}
		},

		func(db *gorm.DB, next db_connect.NextFunc) db_connect.NextFunc {
			return func(query *gorm.DB) (*gorm.DB, error) { // filter account key
				return next(
					query.
						Where("a.account_key = ?", pay.AccountKey),
				)

			}
		},
		func(db *gorm.DB, next db_connect.NextFunc) db_connect.NextFunc {
			return func(query *gorm.DB) (*gorm.DB, error) { // filter account team id
				if pay.AccountTeamId == 0 {
					return next(query)
				}

				return next(
					query.
						Where("a.team_id = ?", pay.AccountTeamId),
				)
			}
		},
		func(db *gorm.DB, next db_connect.NextFunc) db_connect.NextFunc {
			return func(query *gorm.DB) (*gorm.DB, error) { // filter search keyword
				if pay.Keyword == "" {
					return next(query)
				}

				keyword := strings.ToLower(pay.Keyword)
				return next(
					query.
						Where("je.desc ilike ?", "%"+keyword+"%"),
				)
			}
		},
		func(db *gorm.DB, next db_connect.NextFunc) db_connect.NextFunc {
			return func(query *gorm.DB) (*gorm.DB, error) { // filter shop
				if pay.ShopId == 0 {
					return next(query)
				}

				sid := uint(pay.ShopId)
				return next(
					query.
						Joins("JOIN transaction_shops ts ON ts.transaction_id = je.transaction_id").
						Where("ts.shop_id = ?", sid),
				)
			}
		},
		func(db *gorm.DB, next db_connect.NextFunc) db_connect.NextFunc {
			return func(query *gorm.DB) (*gorm.DB, error) { // filter time range
				trange := pay.TimeRange
				if trange.StartDate.IsValid() {
					query = query.
						Where("je.entry_time > ?",
							trange.StartDate.AsTime().Local(),
						)
				}

				if trange.EndDate.IsValid() {
					query = query.
						Where("je.entry_time <= ?",
							trange.EndDate.AsTime().Local(),
						)
				}

				return next(query)
			}
		},
		func(db *gorm.DB, next db_connect.NextFunc) db_connect.NextFunc {
			return func(query *gorm.DB) (*gorm.DB, error) { // filter type label
				if len(pay.Label) == 0 {
					return next(query)
				}

				var keys []accounting_iface.LabelKey
				var labelVals []string
				for _, label := range pay.Label {
					switch val := label.Label.(type) {
					case *accounting_iface.TypeLabelFilter_Marketplace:
						keys = append(keys, accounting_iface.LabelKey_LABEL_KEY_MARKETPLACE)
						labelVals = append(labelVals, common.MarketplaceType_name[int32(val.Marketplace)])
					case *accounting_iface.TypeLabelFilter_OrderType:
						keys = append(keys, accounting_iface.LabelKey_LABEL_KEY_ORDER_TYPE)
						labelVals = append(labelVals, accounting_iface.OrderType_name[int32(val.OrderType)])
					case *accounting_iface.TypeLabelFilter_RevenueSource:
						keys = append(keys, accounting_iface.LabelKey_LABEL_KEY_REVENUE_SOURCE)
						labelVals = append(labelVals, accounting_iface.RevenueSource_name[int32(val.RevenueSource)])
					case *accounting_iface.TypeLabelFilter_TransferPurpose:
						keys = append(keys, accounting_iface.LabelKey_LABEL_KEY_TRANSFER_PURPOSE)
						labelVals = append(labelVals, accounting_iface.MutationPurpose_name[int32(val.TransferPurpose)])
					case *accounting_iface.TypeLabelFilter_WarehouseTransactionType:
						keys = append(keys, accounting_iface.LabelKey_LABEL_KEY_WAREHOUSE_TRANSACTION_TYPE)
						labelVals = append(labelVals, common.InboundSource_name[int32(val.WarehouseTransactionType)])
					}
				}

				tlabelIDs := []uint64{}

				err := db.
					Model(&accounting_core.TypeLabel{}).
					Select("id").
					Where("key in ? and label in ?", keys, labelVals).
					Find(&tlabelIDs).
					Error

				if err != nil {
					return query, err
				}

				return next(
					query.
						Joins("JOIN transaction_type_labels ttl ON ttl.transaction_id = je.transaction_id").
						Where("ttl.type_label_id in ?", tlabelIDs),
				)
			}
		},
		CachePaginated(ctx, l.cache, pay.Page, result.PageInfo),
		// func(db *gorm.DB, next db_connect.NextFunc) db_connect.NextFunc {
		// 	return func(query *gorm.DB) (*gorm.DB, error) { // paginasi
		// 		var queryPaginated *gorm.DB

		// 		queryPaginated, result.PageInfo, err = db_connect.SetPaginationQuery(
		// 			db,
		// 			func() (*gorm.DB, error) {
		// 				return query.Session(&gorm.Session{}), nil
		// 			},
		// 			pay.Page,
		// 		)

		// 		if err != nil {
		// 			return query, err
		// 		}

		// 		return next(queryPaginated)
		// 	}
		// },
		func(db *gorm.DB, next db_connect.NextFunc) db_connect.NextFunc {
			return func(query *gorm.DB) (*gorm.DB, error) { // sorting
				if pay.Sort == nil {
					return next(query)
				}

				var field string
				var orderq string
				switch pay.Sort.Field {
				case accounting_iface.EntryFieldSort_ENTRY_FIELD_SORT_ENTRYTIME:
					field = "je.entry_time"
				default:
					field = "je.entry_time"
				}

				switch pay.Sort.Type {
				case common.SortType_SORT_TYPE_DESC:
					orderq = fmt.Sprintf("%s desc", field)
				case common.SortType_SORT_TYPE_ASC:
					orderq = fmt.Sprintf("%s asc", field)
				default:
					orderq = fmt.Sprintf("%s desc", field)
				}

				return next(
					query.
						Order(orderq),
				)
			}
		},
		func(db *gorm.DB, next db_connect.NextFunc) db_connect.NextFunc {
			return func(query *gorm.DB) (*gorm.DB, error) {
				return next(
					query.
						Select([]string{
							"je.id",
							"je.account_id",
							"je.transaction_id",
							"(EXTRACT(EPOCH FROM je.entry_time) * 1000000)::BIGINT AS entry_time",
							"je.desc",
							"je.debit",
							"je.credit",
						}),
				)
			}
		},
	)

	if err != nil {
		return connect.NewResponse(&result), err
	}

	err = query.
		Find(&result.Data).
		Error

	if err != nil {
		return connect.NewResponse(&result), err
	}

	accountMap := map[uint64]*accounting_iface.EntryAccount{}
	var ok bool
	for _, d := range result.Data {
		var account *accounting_iface.EntryAccount
		account, ok = accountMap[d.AccountId]
		if !ok {
			account = &accounting_iface.EntryAccount{}
			accountMap[d.AccountId] = account

			err = l.
				db.
				Table("accounts a").
				Select([]string{
					"a.id",
					"a.team_id",
					"a.account_key",
					"a.name",
				}).
				Where("id = ?", d.AccountId).
				Find(account).
				Error

			if err != nil {
				return &connect.Response[accounting_iface.EntryListResponse]{}, err
			}

			d.Account = account
		}

	}

	return connect.NewResponse(&result), err
}

type LedgerView interface {
	createQuery() LedgerView
	TeamID(tid uint) LedgerView
	AccountTeamID(tid uint64) LedgerView
	ShopID(sid uint64) LedgerView
	// Marketplace(mpType common.MarketplaceType) LedgerView
	AccountKey(acc_key string) LedgerView
	TimeRange(trange *common.TimeFilterRange) LedgerView
	Page(page *common.PageFilter, pageinfo *common.PageInfo) LedgerView
	Search(keyword string) LedgerView
	TypeLabel(labels []*accounting_iface.TypeLabelFilter) LedgerView
	Count(c *int64) LedgerView
	Sort(sortpay *accounting_iface.EntryListSort) LedgerView
	Iterate(handle func(d *accounting_iface.EntryItem) error) error
	Find(dest interface{}) LedgerView
	Err() error
}

type ledgerViewImpl struct {
	db    *gorm.DB
	query *gorm.DB
	err   error

	mpquery        func(query *gorm.DB) *gorm.DB
	typelabelquery func(query *gorm.DB) *gorm.DB
}

// TypeLabel implements LedgerView.
func (l *ledgerViewImpl) TypeLabel(labels []*accounting_iface.TypeLabelFilter) LedgerView {
	if labels == nil {
		return l
	}

	if len(labels) == 0 {
		return l
	}

	var keys []accounting_iface.LabelKey
	var labelVals []string
	for _, label := range labels {
		switch val := label.Label.(type) {
		case *accounting_iface.TypeLabelFilter_Marketplace:
			keys = append(keys, accounting_iface.LabelKey_LABEL_KEY_MARKETPLACE)
			labelVals = append(labelVals, common.MarketplaceType_name[int32(val.Marketplace)])
		case *accounting_iface.TypeLabelFilter_OrderType:
			keys = append(keys, accounting_iface.LabelKey_LABEL_KEY_ORDER_TYPE)
			labelVals = append(labelVals, accounting_iface.OrderType_name[int32(val.OrderType)])
		case *accounting_iface.TypeLabelFilter_RevenueSource:
			keys = append(keys, accounting_iface.LabelKey_LABEL_KEY_REVENUE_SOURCE)
			labelVals = append(labelVals, accounting_iface.RevenueSource_name[int32(val.RevenueSource)])
		case *accounting_iface.TypeLabelFilter_TransferPurpose:
			keys = append(keys, accounting_iface.LabelKey_LABEL_KEY_TRANSFER_PURPOSE)
			labelVals = append(labelVals, accounting_iface.MutationPurpose_name[int32(val.TransferPurpose)])
		case *accounting_iface.TypeLabelFilter_WarehouseTransactionType:
			keys = append(keys, accounting_iface.LabelKey_LABEL_KEY_WAREHOUSE_TRANSACTION_TYPE)
			labelVals = append(labelVals, common.InboundSource_name[int32(val.WarehouseTransactionType)])
		}
	}

	tlabelIDs := []uint64{}

	err := l.
		db.
		Model(&accounting_core.TypeLabel{}).
		Select("id").
		Where("key in ? and label in ?", keys, labelVals).
		Find(&tlabelIDs).
		Error

	if err != nil {
		return l.setErr(err)
	}

	l.typelabelquery = func(query *gorm.DB) *gorm.DB {
		return query.
			Joins("JOIN transaction_type_labels ttl ON ttl.transaction_id = je.transaction_id").
			Where("ttl.type_label_id in ?", tlabelIDs)
	}

	return l
}

// SearchDescription implements LedgerView.
func (l *ledgerViewImpl) SearchDescription(keyword string) LedgerView {
	l.query = l.
		query.
		Where("je.desc ilike ?", "%"+keyword+"%")
	return l
}

// ShopID implements LedgerView.
func (l *ledgerViewImpl) ShopID(sid uint64) LedgerView {
	if sid == 0 {
		return l
	}

	l.mpquery = func(query *gorm.DB) *gorm.DB {
		return query.
			Joins("JOIN transaction_shops ts ON ts.transaction_id = je.transaction_id").
			Where("ts.shop_id = ?", sid)
	}
	return l

}

// AccountTeamID implements LedgerView.
func (l *ledgerViewImpl) AccountTeamID(tid uint64) LedgerView {
	if tid == 0 {
		return l
	}

	l.query = l.
		query.
		Where("a.team_id = ?", tid)
	return l

}

// Search implements LedgerView.
func (l *ledgerViewImpl) Search(keyword string) LedgerView {
	if keyword == "" {
		return l
	}

	keyword = strings.ToLower(keyword)
	l.query = l.
		query.
		Where("je.desc ilike ?", "%"+keyword+"%")
	return l
}

// Sort implements LedgerView.
func (l *ledgerViewImpl) Sort(sortpay *accounting_iface.EntryListSort) LedgerView {
	if sortpay == nil {
		return l
	}

	var field string
	var orderq string
	switch sortpay.Field {
	case accounting_iface.EntryFieldSort_ENTRY_FIELD_SORT_ENTRYTIME:
		field = "je.entry_time"
	default:
		field = "je.entry_time"
	}

	switch sortpay.Type {
	case common.SortType_SORT_TYPE_DESC:
		orderq = fmt.Sprintf("%s desc", field)
	case common.SortType_SORT_TYPE_ASC:
		orderq = fmt.Sprintf("%s asc", field)
	default:
		orderq = fmt.Sprintf("%s desc", field)
	}

	l.query = l.
		query.
		Order(orderq)

	return l
}

// Page implements LedgerView.
func (l *ledgerViewImpl) Page(page *common.PageFilter, pageinfo *common.PageInfo) LedgerView {
	var err error
	var count int64

	err = l.Count(&count).Err()
	if err != nil {
		return l.setErr(err)
	}
	var total int64 = int64(math.Ceil(float64(count) / float64(page.Limit)))
	pageinfo.TotalItems = count
	pageinfo.CurrentPage = page.Page
	pageinfo.TotalPage = total

	offset := (page.Page - 1) * page.Limit
	l.query = l.
		query.
		Offset(int(offset)).
		Limit(int(page.Limit))

	return l
}

// Iterate implements LedgerView.
func (l *ledgerViewImpl) Iterate(handle func(d *accounting_iface.EntryItem) error) error {
	query := l.query
	if l.typelabelquery != nil {
		query = l.typelabelquery(query)
	}
	if l.mpquery != nil {
		query = l.mpquery(query)
	}

	rows, err := query.
		Select(l.selectFields()).
		Rows()
	if err != nil {
		return err
	}

	accountMap := map[uint64]*accounting_iface.EntryAccount{}
	var ok bool
	for rows.Next() {
		d := accounting_iface.EntryItem{}
		err = l.db.ScanRows(rows, &d)
		if err != nil {
			return err
		}

		var account *accounting_iface.EntryAccount
		account, ok = accountMap[d.AccountId]
		if !ok {
			account = &accounting_iface.EntryAccount{}
			accountMap[d.AccountId] = account

			err = l.
				db.
				Table("accounts a").
				Select([]string{
					"a.id",
					"a.team_id",
					"a.account_key",
					"a.name",
				}).
				Where("id = ?", d.AccountId).
				Find(account).
				Error

			if err != nil {
				return err
			}
		}

		d.Account = account

		err = handle(&d)
		if err != nil {
			return err
		}

	}

	return nil
}

// AccountKey implements LedgerView.
func (l *ledgerViewImpl) AccountKey(acc_key string) LedgerView {
	l.query = l.
		query.
		Where("a.account_key = ?", acc_key)

	return l
}

// TeamID implements LedgerView.
func (l *ledgerViewImpl) TeamID(tid uint) LedgerView {
	l.query = l.
		query.
		Where("je.team_id = ?", tid)

	return l
}

// TimeRange implements LedgerView.
func (l *ledgerViewImpl) TimeRange(trange *common.TimeFilterRange) LedgerView {
	// filter time range
	if trange.StartDate.IsValid() {
		l.query = l.
			query.
			Where("je.entry_time > ?",
				trange.StartDate.AsTime().Local(),
			)
	}

	if trange.EndDate.IsValid() {
		l.query = l.
			query.
			Where("je.entry_time <= ?",
				trange.EndDate.AsTime().Local(),
			)
	}

	return l
}

// Err implements LedgerView.
func (l *ledgerViewImpl) Err() error {
	return l.err
}

func (l *ledgerViewImpl) selectFields() []string {
	return []string{
		"je.id",
		"je.account_id",
		"je.transaction_id",
		"(EXTRACT(EPOCH FROM je.entry_time) * 1000000)::BIGINT AS entry_time",
		"je.desc",
		"je.debit",
		"je.credit",
	}
}

// Find implements LedgerView.
func (l *ledgerViewImpl) Find(dest interface{}) LedgerView {
	err := l.
		query.
		Select(l.selectFields()).
		// Order("je.entry_time desc").
		Find(dest).
		Error

	if err != nil {
		return l.setErr(err)
	}
	return l
}

// Count implements LedgerView.
func (l *ledgerViewImpl) Count(c *int64) LedgerView {

	err := l.
		query.
		Select("count(1)").
		Find(c).
		Error
	return l.setErr(err)
}

func (l *ledgerViewImpl) createQuery() LedgerView {
	l.query = l.
		query.
		Table("journal_entries je").
		Joins("join accounts a on a.id = je.account_id")

	return l
}

func (l *ledgerViewImpl) setErr(err error) *ledgerViewImpl {
	if l.err != nil {
		return l
	}

	if err != nil {
		l.err = err
	}
	return l
}

func NewLedgerView(db *gorm.DB) LedgerView {
	return &ledgerViewImpl{
		db:    db,
		query: db,
	}
}
