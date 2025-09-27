package setup

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	"github.com/pdcgo/accounting_service/accounting_core"
	"github.com/pdcgo/schema/services/accounting_iface/v1"
)

// RecalculateDaily implements accounting_ifaceconnect.AccountingSetupServiceHandler.
func (s *setupServiceImpl) RecalculateDaily(
	ctx context.Context,
	req *connect.Request[accounting_iface.RecalculateDailyRequest],
	stream *connect.ServerStream[accounting_iface.RecalculateDailyResponse]) error {

	balance := accounting_core.NewBalanceCalculate(
		s.db,
		&accounting_core.DBAccount{
			Tx: s.db,
		},
	)

	dayMap := map[string]bool{}
	getkey := func(data *accounting_core.AccountDailyBalance) string {
		return fmt.Sprintf("%s-%d", data.Day.String(), data.AccountID)
	}
	streamlog := func(format string, a ...any) {
		stream.Send(&accounting_iface.RecalculateDailyResponse{
			Message: fmt.Sprintf(format, a...),
		})
	}

	balance.BeforeUpdateDaily = func(daybalance *accounting_core.AccountDailyBalance) error {
		var err error

		key := getkey(daybalance)
		if !dayMap[key] {
			streamlog("removing %s", key)
			err = s.
				db.
				Model(&accounting_core.AccountDailyBalance{}).
				Where("day = ?", daybalance.Day).
				Where("account_id = ?", daybalance.AccountID).
				Delete(&accounting_core.AccountDailyBalance{}).
				Error

			if err != nil {
				return err
			}

			dayMap[key] = true

		}

		return err
	}

	streamlog("fixing daily")
	rows, err := s.db.Model(&accounting_core.JournalEntry{}).Rows()
	if err != nil {
		return err
	}

	for rows.Next() {
		var entry accounting_core.JournalEntry
		err = s.db.ScanRows(rows, &entry)
		if err != nil {
			return err
		}

		err = balance.AddEntry(&entry)
		if err != nil {
			return err
		}

		streamlog("[%d] adding entry %s", entry.ID, entry.Desc)
	}

	return nil
}
