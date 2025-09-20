package ledger

import (
	"context"
	"encoding/csv"
	"fmt"
	"math"
	"time"

	"connectrpc.com/connect"
	"github.com/pdcgo/schema/services/accounting_iface/v1"
	"github.com/pdcgo/schema/services/common/v1"
	"github.com/pdcgo/shared/interfaces/authorization_iface"
	"gorm.io/gorm"
)

type ledgerServiceImpl struct {
	db   *gorm.DB
	auth authorization_iface.Authorization
}

// AccountKeyList implements accounting_ifaceconnect.LedgerServiceHandler.
func (l *ledgerServiceImpl) AccountKeyList(
	ctx context.Context,
	req *connect.Request[accounting_iface.AccountKeyListRequest],
) (*connect.Response[accounting_iface.AccountKeyListResponse], error) {
	var err error
	result := accounting_iface.AccountKeyListResponse{
		Keys: []string{},
	}

	pay := req.Msg

	db := l.db.WithContext(ctx)
	err = db.
		Table("accounts a").
		Select("a.account_key").
		Where("a.team_id = ?", pay.TeamId).
		Find(&result.Keys).
		Error

	return connect.NewResponse(&result), err
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

	view := NewLedgerView(db)
	view.
		createQuery().
		TeamID(uint(pay.TeamId)).
		AccountKey(pay.AccountKey).
		TimeRange(pay.TimeRange).
		Page(pay.Page, result.PageInfo)

	err = view.
		Iterate(func(d *accounting_iface.EntryItem) error {
			result.Data = append(result.Data, d)
			return err
		})

	return connect.NewResponse(&result), err
}

// EntryListExport implements accounting_ifaceconnect.LedgerServiceHandler.
func (l *ledgerServiceImpl) EntryListExport(
	ctx context.Context,
	req *connect.Request[accounting_iface.EntryListExportRequest],
	stream *connect.ServerStream[accounting_iface.EntryListExportResponse]) error {

	var err error
	streamlog := func(msg string) {
		stream.Send(&accounting_iface.EntryListExportResponse{
			Message: msg,
		})
	}

	db := l.db.WithContext(ctx)
	pay := req.Msg

	streamlog("counting data..")

	view := NewLedgerView(db)

	view.
		createQuery().
		TeamID(uint(pay.TeamId)).
		AccountKey(pay.AccountKey).
		TimeRange(pay.TimeRange)

	writer := &ConnectStreamWriter{
		stream: stream,
		offset: 0,
		total:  0,
	}

	err = view.
		Count(&writer.total).
		Err()
	if err != nil {
		return err
	}

	streamlog(fmt.Sprintf("%d data ditemukan..", writer.total))

	csvWriter := csv.NewWriter(writer)

	headers := []string{
		"entry_at",
		"desc",
		"debit",
		"credit",
		"balance",
		"account",
	}

	csvWriter.Write(headers)

	err = view.Iterate(func(d *accounting_iface.EntryItem) error {
		err = csvWriter.Write([]string{
			time.UnixMicro(d.EntryTime).String(),
			d.Desc,
			fmt.Sprintf("%4.f", d.Debit),
			fmt.Sprintf("%4.f", d.Credit),
			fmt.Sprintf("%4.f", d.Balance),
			d.Account.Name,
		})
		csvWriter.Flush()

		return err
	})

	return err
}

func NewLedgerService(db *gorm.DB, auth authorization_iface.Authorization) *ledgerServiceImpl {
	return &ledgerServiceImpl{
		db:   db,
		auth: auth,
	}
}

type ConnectStreamWriter struct {
	stream *connect.ServerStream[accounting_iface.EntryListExportResponse]
	c      int
	offset int64
	total  int64
}

// Write implements io.Writer.
func (c *ConnectStreamWriter) Write(p []byte) (n int, err error) {
	c.c += len(p)
	c.offset += 1
	err = c.stream.Send(&accounting_iface.EntryListExportResponse{
		Offset: c.offset,
		Total:  c.total,
		Data:   p,
	})

	return c.c, err
}

type LedgerView interface {
	createQuery() LedgerView
	TeamID(tid uint) LedgerView
	AccountKey(acc_key string) LedgerView
	TimeRange(trange *common.TimeFilter) LedgerView
	Page(page *common.PageFilter, pageinfo *common.PageInfo) LedgerView
	Count(c *int64) LedgerView
	Iterate(handle func(d *accounting_iface.EntryItem) error) error
	Find(dest interface{}) LedgerView
	Err() error
}

type ledgerViewImpl struct {
	db    *gorm.DB
	query *gorm.DB
	err   error
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
	rows, err := l.
		query.
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
func (l *ledgerViewImpl) TimeRange(trange *common.TimeFilter) LedgerView {
	// filter time range
	if trange.StartDate != 0 {
		l.query = l.
			query.
			Where("je.entry_time > ?",
				time.UnixMicro(trange.StartDate).Local(),
			)
	}

	if trange.EndDate != 0 {
		l.query = l.
			query.
			Where("je.entry_time <= ?",
				time.UnixMicro(trange.EndDate).Local(),
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
		"(EXTRACT(EPOCH FROM je.entry_time) * 1000000)::BIGINT AS entry_time",
		"je.desc",
		"je.debit",
		"je.credit",
		`SUM(
					case
						when a.balance_type = 'd' then je.credit - je.debit
						when a.balance_type = 'c' then je.debit - je.credit
					end
					
				) OVER (
					PARTITION BY je.account_id 
					ORDER BY je.id
					ROWS BETWEEN UNBOUNDED PRECEDING AND CURRENT ROW
				) AS balance`,
	}
}

// Find implements LedgerView.
func (l *ledgerViewImpl) Find(dest interface{}) LedgerView {
	err := l.
		query.
		Select(l.selectFields()).
		Order("je.entry_time desc").
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
