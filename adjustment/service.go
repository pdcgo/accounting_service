package adjustment

import (
	"context"
	"time"

	"connectrpc.com/connect"
	"github.com/pdcgo/accounting_service/accounting_core"
	"github.com/pdcgo/schema/services/accounting_iface/v1"
	"github.com/pdcgo/shared/authorization"
	"github.com/pdcgo/shared/interfaces/authorization_iface"
	"gorm.io/gorm"
)

type Adjustment struct{}

// GetEntityID implements authorization_iface.Entity.
func (a *Adjustment) GetEntityID() string {
	return "accounting_adjustment"
}

type adjServiceImpl struct {
	db   *gorm.DB
	auth authorization_iface.Authorization
}

// AdjCreate implements accounting_ifaceconnect.AdjustmentServiceHandler.
func (a *adjServiceImpl) AdjCreate(
	ctx context.Context,
	req *connect.Request[accounting_iface.AdjCreateRequest],
) (*connect.Response[accounting_iface.AdjCreateResponse], error) {
	var err error
	result := accounting_iface.AdjCreateResponse{}

	pay := req.Msg

	identity := a.
		auth.
		AuthIdentityFromHeader(req.Header())
	agent := identity.Identity()

	if identity.Err() != nil {
		return &connect.Response[accounting_iface.AdjCreateResponse]{}, err
	}

	err = identity.
		HasPermission(authorization_iface.CheckPermissionGroup{
			&Adjustment{}: &authorization_iface.CheckPermission{
				DomainID: authorization.RootDomain,
				Actions:  []authorization_iface.Action{authorization_iface.Create},
			},
		}).
		Err()

	if err != nil {
		return &connect.Response[accounting_iface.AdjCreateResponse]{}, err
	}

	db := a.db.WithContext(ctx)
	err = accounting_core.OpenTransaction(db, func(tx *gorm.DB, bookmng accounting_core.BookManage) error {
		id := time.Now().Unix()
		ref := accounting_core.NewRefID(&accounting_core.RefData{
			RefType: accounting_core.AdminAdjustmentRef,
			ID:      uint(id),
		})

		tran := accounting_core.Transaction{
			RefID:       ref,
			TeamID:      authorization.RootDomain,
			CreatedByID: agent.IdentityID(),
			Desc:        pay.Description,
			Created:     time.Now(),
		}

		err = bookmng.
			NewTransaction().
			Create(&tran).
			Err()
		if err != nil {
			return err
		}

		for _, book := range pay.Books {
			entry := bookmng.
				NewCreateEntry(uint(book.TeamId), agent.IdentityID())

			for _, pentry := range book.Entries {
				entry.Set(uint(pentry.AccountId), pentry.Credit, pentry.Debit)
			}

			err = entry.
				Transaction(&tran).
				Commit().
				Err()

			if err != nil {
				return err
			}
		}

		return nil
	})

	return connect.NewResponse(&result), nil

}

func NewAdjustmentService(db *gorm.DB, auth authorization_iface.Authorization) *adjServiceImpl {
	return &adjServiceImpl{
		db:   db,
		auth: auth,
	}
}
