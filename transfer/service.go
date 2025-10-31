package transfer

import (
	"context"

	"connectrpc.com/connect"
	"github.com/pdcgo/schema/services/accounting_iface/v1"
	"github.com/pdcgo/shared/interfaces/authorization_iface"
	"gorm.io/gorm"
)

type transferImpl struct {
	db   *gorm.DB
	auth authorization_iface.Authorization
}

// TransferTeam implements accounting_ifaceconnect.TransferServiceHandler.
func (t *transferImpl) TransferTeam(context.Context, *connect.Request[accounting_iface.TransferTeamRequest]) (*connect.Response[accounting_iface.TransferTeamResponse], error) {
	panic("unimplemented")
}

// TransferList implements accounting_ifaceconnect.TransferServiceHandler.
func (t *transferImpl) TransferList(
	ctx context.Context,
	req *connect.Request[accounting_iface.TransferListRequest],
) (*connect.Response[accounting_iface.TransferListResponse], error) {
	panic("unimplemented")
}

func NewTransferService(db *gorm.DB, auth authorization_iface.Authorization) *transferImpl {
	return &transferImpl{
		db:   db,
		auth: auth,
	}
}
