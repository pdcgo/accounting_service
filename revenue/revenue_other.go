package revenue

import (
	"context"
	"fmt"
	"time"

	"connectrpc.com/connect"
	"github.com/pdcgo/accounting_service/accounting_core"
	"github.com/pdcgo/schema/services/access_iface/v1"
	"github.com/pdcgo/schema/services/revenue_iface/v1"
	"github.com/pdcgo/shared/authorization"
	"github.com/pdcgo/shared/custom_connect"
	"github.com/pdcgo/shared/db_models"
	"github.com/pdcgo/shared/interfaces/authorization_iface"
	"gorm.io/gorm"
)

// RevenueOther implements revenue_ifaceconnect.RevenueServiceHandler.
func (r *revenueServiceImpl) RevenueOther(ctx context.Context, req *connect.Request[revenue_iface.RevenueOtherRequest]) (*connect.Response[revenue_iface.RevenueOtherResponse], error) {
	var err error

	result := revenue_iface.RevenueOtherResponse{}
	pay := req.Msg

	source, err := custom_connect.GetRequestSource(ctx)
	if err != nil {
		return connect.NewResponse(&result), err
	}

	identity := r.auth.AuthIdentityFromHeader(req.Header())
	agent := identity.Identity()

	var domainCheck uint
	switch source.RequestFrom {
	case access_iface.RequestFrom_REQUEST_FROM_ADMIN:
		domainCheck = authorization.RootDomain
	default:
		domainCheck = uint(source.TeamId)

	}

	identity.
		HasPermission(authorization_iface.CheckPermissionGroup{
			&db_models.Order{}: &authorization_iface.CheckPermission{
				DomainID: domainCheck,
				Actions:  []authorization_iface.Action{authorization_iface.Update},
			},
		})

	err = identity.Err()
	if err != nil {
		return connect.NewResponse(&result), err
	}

	db := r.db.WithContext(ctx)
	err = accounting_core.OpenTransaction(ctx, db, func(tx *gorm.DB, bookmng accounting_core.BookManage) error {
		ref := accounting_core.NewStringRefID(&accounting_core.StringRefData{
			RefType: accounting_core.RevenueAdjustmentRef,
			ID:      pay.ExternalRevenueId,
		})

		refmut := accounting_core.NewTransactionMutation(ctx, tx).
			ByRefID(ref, true)

		err = refmut.Err()
		if err != nil {
			return err
		}

		texist := refmut.IsExist()

		if texist {
			return accounting_core.ErrSkipTransaction
		}

		tran := accounting_core.Transaction{
			RefID:       ref,
			TeamID:      uint(pay.TeamId),
			CreatedByID: agent.IdentityID(),
			Desc:        fmt.Sprintf("returning order %s %s", ref, pay.Desc),
			Created:     time.Now(),
		}

		err = bookmng.
			NewTransaction().
			Create(&tran).
			AddCustomerServiceID(uint(pay.LabelInfo.CsId)).
			AddShopID(uint(pay.LabelInfo.ShopId)).
			AddTypeLabel(pay.LabelInfo.TypeLabels).
			Err()

		if err != nil {
			return err
		}

		entry := bookmng.
			NewCreateEntry(uint(pay.TeamId), agent.IdentityID()).
			To(&accounting_core.EntryAccountPayload{
				Key:    accounting_core.OtherRevenueAccount,
				TeamID: uint(pay.TeamId),
			}, pay.Amount).
			To(&accounting_core.EntryAccountPayload{
				Key:    accounting_core.CashAccount,
				TeamID: uint(pay.TeamId),
			}, pay.Amount)

		err = entry.
			Transaction(&tran).
			Commit().
			Err()

		if err != nil {
			return err
		}

		result.TransactionId = uint64(tran.ID)
		return nil
	})

	if err != nil {
		return connect.NewResponse(&result), err
	}

	return connect.NewResponse(&result), nil
}
