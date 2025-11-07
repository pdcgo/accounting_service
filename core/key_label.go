package core

import (
	"context"

	"connectrpc.com/connect"
	"github.com/pdcgo/accounting_service/accounting_core"
	"github.com/pdcgo/schema/services/accounting_iface/v1"
)

// TypeLabelList implements accounting_ifaceconnect.CoreServiceHandler.
func (l *coreServiceImpl) TypeLabelList(
	ctx context.Context,
	req *connect.Request[accounting_iface.TypeLabelListRequest]) (*connect.Response[accounting_iface.TypeLabelListResponse], error) {

	var err error
	db := l.db.WithContext(ctx)
	result := &accounting_iface.TypeLabelListResponse{
		List: []*accounting_iface.TypeLabel{},
	}

	err = db.
		Model(&accounting_core.TypeLabel{}).
		Find(&result.List).
		Error

	return connect.NewResponse(result), err
}
