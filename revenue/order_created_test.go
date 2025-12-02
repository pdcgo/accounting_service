package revenue_test

import (
	"testing"

	"connectrpc.com/connect"
	"github.com/pdcgo/accounting_service/accounting_core"
	"github.com/pdcgo/accounting_service/accounting_mock"
	"github.com/pdcgo/accounting_service/revenue"
	"github.com/pdcgo/schema/services/common/v1"
	"github.com/pdcgo/schema/services/revenue_iface/v1"
	"github.com/pdcgo/shared/authorization/authorization_mock"
	"github.com/pdcgo/shared/db_models"
	"github.com/pdcgo/shared/interfaces/authorization_iface"
	"github.com/pdcgo/shared/pkg/moretest"
	"github.com/pdcgo/shared/pkg/moretest/moretest_mock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gorm.io/gorm"
)

type MockAuth struct {
	mock.Mock
}

func (m *MockAuth) AuthIdentityFromToken(token string) authorization_iface.Identity {
	args := m.Called(token)
	return args.Get(0).(authorization_iface.Identity)
}

type MockIdentity struct {
	mock.Mock
}

func (m *MockIdentity) Identity() authorization_iface.Identity {
	args := m.Called()
	return args.Get(0).(authorization_iface.Identity)
}

func (m *MockIdentity) Err() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockIdentity) GetUserID() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockIdentity) IdentityID() string {
	args := m.Called()
	return args.String(0)
}

func TestOnOrder(t *testing.T) {
	var db gorm.DB

	var migrate moretest.SetupFunc = func(t *testing.T) func() error {
		err := db.AutoMigrate(
			accounting_core.Account{},
			accounting_core.JournalEntry{},
			accounting_core.Transaction{},
			accounting_core.AccountingTag{},
			accounting_core.TransactionTag{},
			accounting_core.TransactionShop{},
			accounting_core.TransactionCustomerService{},
			accounting_core.TypeLabel{},
			accounting_core.TransactionTypeLabel{},
			db_models.Marketplace{},
		)
		assert.Nil(t, err)
		return nil
	}

	var seed moretest.SetupFunc = func(t *testing.T) func() error {

		mps := []*db_models.Marketplace{
			{
				ID: 3,
			},
		}
		err := db.Save(&mps).Error
		assert.Nil(t, err)

		return nil
	}

	moretest.Suite(t, "testing onorder",
		moretest.SetupListFunc{
			moretest_mock.MockSqliteDatabase(&db),
			migrate,
			accounting_mock.PopulateAccountKey(&db, 101),
			accounting_mock.PopulateAccountKey(&db, 789),
			accounting_mock.PopulateAccountKey(&db, 102),
			seed,
		},
		func(t *testing.T) {
			service := revenue.NewRevenueService(&db, &authorization_mock.EmptyAuthorizationMock{
				AuthIdentityMock: &authorization_mock.AuthIdentityMock{
					IdentityMock: &authorization_mock.IdentityMock{
						ID: 20,
					},
				},
			})

			t.Run("successful on order", func(t *testing.T) {
				req := &connect.Request[revenue_iface.OnOrderRequest]{
					Msg: &revenue_iface.OnOrderRequest{
						Token:   "test_token",
						Event:   revenue_iface.OrderEvent_ORDER_EVENT_CREATED,
						OrderId: 123,
						OrderInfo: &revenue_iface.OrderInfo{
							Receipt:         "GERSASDASD",
							ExternalOrderId: "ASDASD",
						},
						LabelInfo: &revenue_iface.ExtraLabelInfo{
							CsId:       1,
							SupplierId: 2,
							Tags:       []string{"shopee"},
							ShopId:     3,
						},
						WarehouseId:    789,
						TeamId:         101,
						OwnStockAmount: 10,
						WarehouseFee:   5,
						BorrowStock: []*revenue_iface.BorrowStock{
							{
								TeamId:     102,
								Amount:     2000,
								SellAmount: 3000,
							},
						},
						OrderAmount: 100,
					},
				}

				_, err := service.OnOrder(t.Context(), req)
				assert.NoError(t, err)

				t.Run("checking entries", func(t *testing.T) {
					var entries accounting_core.JournalEntriesList
					err := db.Model(&accounting_core.JournalEntry{}).Find(&entries).Error
					assert.NoError(t, err)

					entries.PrintJournalEntries(&db)
				})
			})

		},
	)

}

func TestOnOrderCustom(t *testing.T) {
	var db gorm.DB

	var migrate moretest.SetupFunc = func(t *testing.T) func() error {
		err := db.AutoMigrate(
			accounting_core.Account{},
			accounting_core.JournalEntry{},
			accounting_core.Transaction{},
			accounting_core.AccountingTag{},
			accounting_core.TransactionTag{},
			accounting_core.TransactionShop{},
			accounting_core.TransactionCustomerService{},
			accounting_core.TypeLabel{},
			accounting_core.TransactionTypeLabel{},
			db_models.Marketplace{},
		)
		assert.Nil(t, err)
		return nil
	}

	var seed moretest.SetupFunc = func(t *testing.T) func() error {

		mps := []*db_models.Marketplace{
			{
				ID: 3,
			},
		}
		err := db.Save(&mps).Error
		assert.Nil(t, err)

		return nil
	}

	moretest.Suite(t, "testing onorder",
		moretest.SetupListFunc{
			moretest_mock.MockSqliteDatabase(&db),
			migrate,
			accounting_mock.PopulateAccountKey(&db, 101),
			accounting_mock.PopulateAccountKey(&db, 789),
			accounting_mock.PopulateAccountKey(&db, 102),
			seed,
		},
		func(t *testing.T) {
			service := revenue.NewRevenueService(&db, &authorization_mock.EmptyAuthorizationMock{
				AuthIdentityMock: &authorization_mock.AuthIdentityMock{
					IdentityMock: &authorization_mock.IdentityMock{
						ID: 20,
					},
				},
			})

			t.Run("successful on order", func(t *testing.T) {
				req := &connect.Request[revenue_iface.OnOrderRequest]{
					Msg: &revenue_iface.OnOrderRequest{
						Token:   "test_token",
						Event:   revenue_iface.OrderEvent_ORDER_EVENT_CREATED,
						OrderId: 123,
						OrderInfo: &revenue_iface.OrderInfo{
							Receipt:         "GERSASDASD",
							ExternalOrderId: "ASDASD",
						},
						LabelInfo: &revenue_iface.ExtraLabelInfo{
							CsId:       1,
							SupplierId: 2,
							Tags:       []string{"shopee"},
							ShopId:     3,
						},
						WarehouseId:    789,
						TeamId:         101,
						OwnStockAmount: 10,
						WarehouseFee:   5,
						BorrowStock: []*revenue_iface.BorrowStock{
							{
								TeamId:     102,
								Amount:     2000,
								SellAmount: 3000,
							},
						},
						OrderAmount: 100,
						AdditionalPayment: &revenue_iface.OnOrderRequest_FakeOrderPayment{
							FakeOrderPayment: &revenue_iface.FakeOrderPayment{
								PaymentMethod: common.PaymentMethod_PAYMENT_METHOD_CASH,
								Amount:        12000,
							},
						},
					},
				}

				_, err := service.OnOrder(t.Context(), req)
				assert.NoError(t, err)

				t.Run("checking entries", func(t *testing.T) {
					var entries accounting_core.JournalEntriesList
					err := db.Model(&accounting_core.JournalEntry{}).Find(&entries).Error
					assert.NoError(t, err)

					entries.PrintJournalEntries(&db)
				})
			})

		},
	)

}
