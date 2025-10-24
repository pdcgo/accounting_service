package expense_test

import (
	"testing"

	"connectrpc.com/connect"
	"github.com/pdcgo/accounting_service/accounting_core"
	"github.com/pdcgo/accounting_service/accounting_mock"
	"github.com/pdcgo/accounting_service/accounting_model"
	"github.com/pdcgo/accounting_service/expense"
	"github.com/pdcgo/schema/services/accounting_iface/v1"
	"github.com/pdcgo/schema/services/common/v1"
	"github.com/pdcgo/shared/authorization/authorization_mock"
	"github.com/pdcgo/shared/pkg/moretest"
	"github.com/pdcgo/shared/pkg/moretest/moretest_mock"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

func TestExpenseList(t *testing.T) {
	var db gorm.DB

	var migrate moretest.SetupFunc = func(t *testing.T) func() error {
		err := db.AutoMigrate(
			&accounting_model.Expense{},
			&accounting_core.Transaction{},
			&accounting_core.Label{},
			&accounting_core.TransactionLabel{},
			&accounting_core.Account{},
			&accounting_core.JournalEntry{},
		)
		assert.Nil(t, err)

		return nil
	}

	moretest.Suite(t, "testing expense",
		moretest.SetupListFunc{
			moretest_mock.MockSqliteDatabase(&db),
			migrate,
			accounting_mock.PopulateAccountKey(&db, 5),
		},
		func(t *testing.T) {
			srv := expense.NewExpenseService(&db, &authorization_mock.EmptyAuthorizationMock{
				AuthIdentityMock: &authorization_mock.AuthIdentityMock{
					IdentityMock: &authorization_mock.IdentityMock{
						ID: 1,
					},
				},
			})

			_, err := srv.ExpenseCreate(t.Context(), &connect.Request[accounting_iface.ExpenseCreateRequest]{
				Msg: &accounting_iface.ExpenseCreateRequest{
					TeamId:      5,
					Desc:        "asasdasdas ",
					ExpenseType: accounting_iface.ExpenseType_EXPENSE_TYPE_INTERNAL,
					ExpenseKey:  string(accounting_core.SalaryAccount),
					Amount:      12000,
					RequestFrom: common.RequestFrom_REQUEST_FROM_ADMIN,
				},
			})

			assert.Nil(t, err)

			// t.Run("testing expense list", func(t *testing.T) {

			// 	res, err := srv.ExpenseList(t.Context(), &connect.Request[accounting_iface.ExpenseListRequest]{
			// 		Msg: &accounting_iface.ExpenseListRequest{
			// 			TimeRange: &common.TimeFilterRange{
			// 				EndDate: timestamppb.Now(),
			// 			},
			// 			Page: &common.PageFilter{
			// 				Page:  1,
			// 				Limit: 10,
			// 			},
			// 		},
			// 	})

			// 	assert.Nil(t, err)
			// 	debugtool.LogJson(res)
			// })

		},
	)

}
