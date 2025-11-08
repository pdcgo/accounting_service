package ledger

import (
	"context"
	"encoding/csv"
	"fmt"
	"time"

	"connectrpc.com/connect"
	"github.com/pdcgo/schema/services/accounting_iface/v1"
	"github.com/pdcgo/shared/interfaces/authorization_iface"
	"gorm.io/gorm"
)

type ledgerServiceImpl struct {
	db   *gorm.DB
	auth authorization_iface.Authorization
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
		Search(pay.Keyword).
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
