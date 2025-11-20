package stock

import (
	"context"

	"connectrpc.com/connect"
	"github.com/pdcgo/schema/services/stock_iface/v1"
	"github.com/pdcgo/shared/db_models"
	"github.com/pdcgo/shared/interfaces/authorization_iface"
	"gorm.io/gorm"
)

type stockServiceImpl struct {
	db   *gorm.DB
	auth authorization_iface.Authorization
}

// InboundAccept implements stock_ifaceconnect.StockServiceHandler.
func (s *stockServiceImpl) InboundAccept(
	ctx context.Context,
	req *connect.Request[stock_iface.InboundAcceptRequest],
) (*connect.Response[stock_iface.InboundAcceptResponse], error) {
	var err error

	pay := req.Msg
	result := &stock_iface.InboundAcceptResponse{}

	identity := s.auth.AuthIdentityFromHeader(req.Header())
	agent := identity.Identity()
	err = identity.
		HasPermission(authorization_iface.CheckPermissionGroup{
			&db_models.InvTransaction{}: &authorization_iface.CheckPermission{
				DomainID: uint(pay.WarehouseId),
				Actions:  []authorization_iface.Action{authorization_iface.Update},
			},
		}).
		Err()

	if err != nil {
		return connect.NewResponse(result), err
	}

	handle := inboundAccept{
		s:     s,
		ctx:   ctx,
		req:   req,
		agent: agent,
	}

	result, err = handle.accept()
	return connect.NewResponse(result), err
}

// StockAdjustment implements stock_ifaceconnect.StockServiceHandler.
func (s *stockServiceImpl) StockAdjustment(
	ctx context.Context,
	req *connect.Request[stock_iface.StockAdjustmentRequest],
) (*connect.Response[stock_iface.StockAdjustmentResponse], error) {
	var err error

	pay := req.Msg
	result := &stock_iface.StockAdjustmentResponse{}

	identity := s.auth.AuthIdentityFromHeader(req.Header())
	agent := identity.Identity()
	err = identity.
		HasPermission(authorization_iface.CheckPermissionGroup{
			&db_models.InvTransaction{}: &authorization_iface.CheckPermission{DomainID: uint(pay.TeamId), Actions: []authorization_iface.Action{authorization_iface.Update}},
		}).
		Err()

	if err != nil {
		return connect.NewResponse(result), err
	}

	handle := stockAdjustment{
		s:     s,
		ctx:   ctx,
		req:   req,
		agent: agent,
	}

	result, err = handle.adjustment()
	return connect.NewResponse(result), err

}

func NewStockService(db *gorm.DB, auth authorization_iface.Authorization) *stockServiceImpl {
	return &stockServiceImpl{
		db:   db,
		auth: auth,
	}
}
