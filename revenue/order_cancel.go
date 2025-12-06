package revenue

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	"github.com/pdcgo/accounting_service/accounting_core"
	"github.com/pdcgo/schema/services/revenue_iface/v1"
	"github.com/pdcgo/shared/db_models"
	"github.com/pdcgo/shared/interfaces/authorization_iface"
	"gorm.io/gorm"
)

// OrderCancel implements revenue_ifaceconnect.RevenueServiceHandler.
func (r *revenueServiceImpl) OrderCancel(
	ctx context.Context,
	req *connect.Request[revenue_iface.OrderCancelRequest],
) (*connect.Response[revenue_iface.OrderCancelResponse], error) {
	var err error
	res := connect.NewResponse(&revenue_iface.OrderCancelResponse{})
	db := r.db.WithContext(ctx)
	pay := req.Msg

	identity := r.
		auth.
		AuthIdentityFromHeader(req.Header()).
		HasPermission(authorization_iface.CheckPermissionGroup{
			&db_models.Order{}: &authorization_iface.CheckPermission{
				DomainID: uint(pay.TeamId),
				Actions:  []authorization_iface.Action{authorization_iface.Update},
			},
		})
	agent := identity.Identity()

	err = identity.Err()
	if err != nil {
		return res, err
	}

	err = db.Transaction(func(tx *gorm.DB) error {
		ref := accounting_core.NewRefID(&accounting_core.RefData{
			RefType: accounting_core.OrderRef,
			ID:      uint(pay.OrderId),
		})
		return accounting_core.
			NewTransactionMutation(ctx, tx).
			ByRefID(accounting_core.RefID(ref), true).
			RollbackEntry(agent.GetUserID(), fmt.Sprintf("cancelling order %s", ref)).
			Err()
	})

	return res, err
}
