package accounting_service_test

import (
	"context"
	"net/http"
	"testing"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"github.com/pdcgo/accounting_service"
	"github.com/pdcgo/accounting_service/accounting_model"
	"github.com/pdcgo/schema/services/accounting_iface/v1"
	"github.com/pdcgo/schema/services/common/v1"
	"github.com/pdcgo/shared/interfaces/authorization_iface"
	"github.com/pdcgo/shared/pkg/moretest"
	"github.com/pdcgo/shared/pkg/moretest/moretest_mock"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

type authIdMock struct{}

// Err implements authorization_iface.AuthIdentity.
func (a *authIdMock) Err() error {
	return nil
}

// HasPermission implements authorization_iface.AuthIdentity.
func (a *authIdMock) HasPermission(perms authorization_iface.CheckPermissionGroup) authorization_iface.AuthIdentity {
	return a
}

// Identity implements authorization_iface.AuthIdentity.
func (a *authIdMock) Identity() authorization_iface.Identity {
	return nil
}

type authMock struct{}

// ApiQueryCheckPermission implements authorization_iface.Authorization.
func (a *authMock) ApiQueryCheckPermission(identity authorization_iface.Identity, query authorization_iface.PermissionQuery) (bool, error) {
	return true, nil
}

// AuthIdentityFromHeader implements authorization_iface.Authorization.
func (a *authMock) AuthIdentityFromHeader(header http.Header) authorization_iface.AuthIdentity {
	return &authIdMock{}
}

// AuthIdentityFromToken implements authorization_iface.Authorization.
func (a *authMock) AuthIdentityFromToken(token string) authorization_iface.AuthIdentity {
	return &authIdMock{}
}

// HasPermission implements authorization_iface.Authorization.
func (a *authMock) HasPermission(identity authorization_iface.Identity, perms authorization_iface.CheckPermissionGroup) error {
	return nil
}

func TestCreateAccount(t *testing.T) {
	var db gorm.DB

	moretest.Suite(
		t,
		"test create account",
		moretest.SetupListFunc{
			moretest_mock.MockSqliteDatabase(&db),
			func(t *testing.T) func() error {
				err := db.AutoMigrate(
					&accounting_model.BankAccountV2{},
					// &accounting_model.BusinessAccountLabel{},
					// &accounting_model.BusinessAccountLabelRelation{},
				)
				assert.Nil(t, err)

				return nil
			},
		},
		func(t *testing.T) {
			service := accounting_service.NewAccountService(&db, &authMock{})
			numberID := uuid.New().String()
			req := connect.NewRequest(&accounting_iface.AccountCreateRequest{
				TeamId:        1,
				Name:          "Gusti Irawan Baskoro",
				NumberId:      numberID,
				AccountTypeId: 1,
				// AccountType: "bank",
				Labels: []*common.KeyName{},
			})
			_, err := service.AccountCreate(context.Background(), req)
			assert.Nil(t, err)

			t.Run("test create duplicate number_id", func(t *testing.T) {
				_, err := service.AccountCreate(context.Background(), req)
				assert.NotNil(t, err)
			})
		},
	)
}
