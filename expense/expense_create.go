package expense

import (
	"context"
	"time"

	"connectrpc.com/connect"
	"github.com/pdcgo/accounting_service/accounting_core"
	"github.com/pdcgo/accounting_service/accounting_model"
	"github.com/pdcgo/accounting_service/accounting_transaction/expense_transaction"
	"github.com/pdcgo/schema/services/accounting_iface/v1"
	"github.com/pdcgo/schema/services/common/v1"
	"github.com/pdcgo/shared/authorization"
	"github.com/pdcgo/shared/interfaces/authorization_iface"
	"gorm.io/gorm"
)

// ExpenseCreate implements accounting_ifaceconnect.ExpenseServiceHandler.
func (e *expenseServiceImpl) ExpenseCreate(
	ctx context.Context,
	req *connect.Request[accounting_iface.ExpenseCreateRequest],
) (*connect.Response[accounting_iface.ExpenseCreateResponse], error) {
	var err error
	result := connect.Response[accounting_iface.ExpenseCreateResponse]{
		Msg: &accounting_iface.ExpenseCreateResponse{},
	}

	pay := req.Msg
	identity := e.auth.AuthIdentityFromHeader(req.Header())
	agent := identity.Identity()

	var domainID uint
	switch pay.RequestFrom {
	case common.RequestFrom_REQUEST_FROM_ADMIN:
		domainID = authorization.RootDomain
	case common.RequestFrom_REQUEST_FROM_SELLING, common.RequestFrom_REQUEST_FROM_WAREHOUSE:
		domainID = uint(pay.TeamId)

	}

	err = identity.
		HasPermission(authorization_iface.CheckPermissionGroup{
			accounting_model.ExpenseEntity{}: &authorization_iface.CheckPermission{
				DomainID: domainID,
				Actions:  []authorization_iface.Action{authorization_iface.Create},
			},
		}).
		Err()

	if err != nil {
		return &result, err
	}

	err = e.
		db.
		WithContext(ctx).
		Transaction(func(tx *gorm.DB) error {
			exp := accounting_model.Expense{
				TeamID:      uint(pay.TeamId),
				CreatedByID: agent.GetUserID(),
				ExpenseType: pay.ExpenseType,
				ExpenseKey:  pay.ExpenseKey,
				Desc:        pay.Desc,
				Amount:      pay.Amount,
				CreatedAt:   time.Now(),
			}

			err = tx.Save(&exp).Error
			if err != nil {
				return err
			}

			err = expense_transaction.
				NewExpenseTransaction(ctx, tx, identity.Identity()).
				ExpenseCreate(&expense_transaction.CreatePayload{
					TeamID:      uint(pay.TeamId),
					ExpenseKey:  accounting_core.AccountKey(pay.ExpenseKey),
					ExpenseType: pay.ExpenseType,
					Amount:      pay.Amount,
					Desc:        pay.Desc,
				})

			return err
		})

	return &result, err
}
