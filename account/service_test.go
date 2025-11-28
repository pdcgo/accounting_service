package account_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"github.com/pdcgo/accounting_service/account"
	"github.com/pdcgo/accounting_service/accounting_model"
	"github.com/pdcgo/schema/services/access_iface/v1"
	"github.com/pdcgo/schema/services/accounting_iface/v1"
	"github.com/pdcgo/schema/services/common/v1"
	"github.com/pdcgo/shared/custom_connect"
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
			service := account.NewAccountService(&db, &authMock{})
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

func TestDeleteAccount(t *testing.T) {
	var db gorm.DB
	ctx := context.WithValue(context.TODO(), "source", &access_iface.RequestSource{
		TeamId:      1,
		RequestFrom: access_iface.RequestFrom_REQUEST_FROM_SELLING,
	})

	moretest.Suite(
		t,
		"test delete account",
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
			service := account.NewAccountService(&db, &authMock{})
			numberID := uuid.New().String()
			req := connect.NewRequest(&accounting_iface.AccountCreateRequest{
				TeamId:        1,
				Name:          "Gusti Irawan Baskoro",
				NumberId:      numberID,
				AccountTypeId: 1,
				Labels:        []*common.KeyName{},
			})
			res, err := service.AccountCreate(ctx, req)
			assert.Nil(t, err)
			assert.NotZero(t, res.Msg.Id)

			t.Run(fmt.Sprintf("test delete account id %d", res.Msg.Id), func(t *testing.T) {
				req := connect.NewRequest(&accounting_iface.AccountDeleteRequest{
					TeamId:     1,
					AccountIds: []uint64{res.Msg.Id},
				})
				_, err = service.AccountDelete(ctx, req)
				assert.Nil(t, err)
			})
		},
	)
}

func TestUpdateAccount(t *testing.T) {
	var db gorm.DB
	ctx := context.WithValue(context.TODO(), custom_connect.SourceKey, &access_iface.RequestSource{
		TeamId:      1,
		RequestFrom: access_iface.RequestFrom_REQUEST_FROM_SELLING,
	})

	moretest.Suite(
		t,
		"test update account",
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
			service := account.NewAccountService(&db, &authMock{})
			numberID := uuid.New().String()
			req := connect.NewRequest(&accounting_iface.AccountCreateRequest{
				TeamId:        1,
				Name:          "Gusti Irawan Baskoro",
				NumberId:      numberID,
				AccountTypeId: 1,
				Labels:        []*common.KeyName{},
			})
			res, err := service.AccountCreate(ctx, req)
			assert.Nil(t, err)
			assert.NotZero(t, res.Msg.Id)

			newNumberID := uuid.New().String()
			t.Run(fmt.Sprintf("test update account id %d", res.Msg.Id), func(t *testing.T) {
				req := connect.NewRequest(&accounting_iface.AccountUpdateRequest{
					TeamId:   1,
					Id:       res.Msg.Id,
					Name:     "Caket Sirait",
					NumberId: newNumberID,
					Labels:   []*common.KeyName{},
				})
				_, err = service.AccountUpdate(ctx, req)
				assert.Nil(t, err)
			})

			t.Run(fmt.Sprintf("test account id %d updated", res.Msg.Id), func(t *testing.T) {
				var acc accounting_model.BankAccountV2
				err = db.First(&acc, res.Msg.Id).Error
				assert.Nil(t, err)
				assert.Equal(t, acc.NumberID, newNumberID)
			})
		},
	)
}
