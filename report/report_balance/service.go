package report_balance

import (
	"context"
	"errors"
	"fmt"

	"connectrpc.com/connect"
	"github.com/pdcgo/schema/services/report_iface/v1"
	"github.com/pdcgo/shared/interfaces/authorization_iface"
	"gorm.io/gorm"
)

type balanceImpl struct {
	db   *gorm.DB
	auth authorization_iface.Authorization
}

// BalanceResync implements report_ifaceconnect.BalanceServiceHandler.
func (b *balanceImpl) BalanceResync(
	ctx context.Context,
	req *connect.Request[report_iface.BalanceResyncRequest],
	stream *connect.ServerStream[report_iface.BalanceResyncResponse]) error {
	var err error
	db := b.db.WithContext(ctx)

	identity := b.auth.AuthIdentityFromHeader(req.Header())
	agent := identity.Identity()

	err = identity.Err()
	if err != nil {
		return err
	}

	if !agent.IsSuperUser() {
		return errors.New("bukan superuser")
	}

	var streamlog = func(format string, a ...any) {
		msg := fmt.Sprintf(format, a...)
		stream.Send(&report_iface.BalanceResyncResponse{
			Msg: msg,
		})
	}

	streamlog("syncing daily account key")
	err = db.Transaction(func(tx *gorm.DB) error {
		statements := accountKeyDailySync()

		for msg, stmt := range statements {
			streamlog(msg)
			err = tx.Exec(stmt).Error
			if err != nil {
				return err
			}
		}

		return nil
	})
	if err != nil {
		return err
	}

	err = db.Transaction(func(tx *gorm.DB) error {
		statements := accountDailySync()
		for msg, stmt := range statements {
			streamlog(msg)
			err = tx.Exec(stmt).Error
			if err != nil {
				return err
			}
		}

		return nil
	})

	if err != nil {
		return err
	}

	streamlog("complete sync daily account key")
	return nil

}

func NewBalanceService(
	db *gorm.DB,
	auth authorization_iface.Authorization,
) *balanceImpl {
	return &balanceImpl{
		db:   db,
		auth: auth,
	}
}
