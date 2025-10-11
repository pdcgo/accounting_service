package report

import (
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/pdcgo/accounting_service/accounting_core"
	"github.com/pdcgo/schema/services/report_iface/v1"
	"github.com/pdcgo/shared/authorization/authorization_mock"
	"github.com/pdcgo/shared/pkg/moretest"
	"github.com/pdcgo/shared/pkg/moretest/moretest_mock"
	"github.com/pdcgo/shared/pkg/ware_cache"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/gorm"
)

func TestDailyUpdateBalance(t *testing.T) {
	var db gorm.DB

	var migrate moretest.SetupFunc = func(t *testing.T) func() error {
		err := db.AutoMigrate(
			&accounting_core.Account{},
		)

		assert.Nil(t, err)

		return nil
	}

	var seed moretest.SetupFunc = func(t *testing.T) func() error {
		return nil
	}

	moretest.Suite(t, "test daily",
		moretest.SetupListFunc{
			moretest_mock.MockSqliteDatabase(&db),
			migrate,
			seed,
		},
		func(t *testing.T) {
			reportService := NewAccountReportService(
				&db,
				&authorization_mock.EmptyAuthorizationMock{},
				ware_cache.NewLocalCache(),
			)

			t.Run("test normal update", func(t *testing.T) {
				_, err := reportService.DailyUpdateBalance(t.Context(), &connect.Request[report_iface.DailyUpdateBalanceRequest]{
					Msg: &report_iface.DailyUpdateBalanceRequest{
						LabelExtra: &report_iface.TxLabelExtra{
							ShopId: 1,
							CsId:   2,
						},
						Entries: []*report_iface.EntryPayload{
							{
								AccountId:     3,
								TeamId:        1,
								TransactionId: 1,
								EntryTime:     timestamppb.New(time.Now()),
								Debit:         12000,
								Desc:          "test",
							},
							{
								AccountId:     4,
								TeamId:        1,
								TransactionId: 1,
								EntryTime:     timestamppb.New(time.Now()),
								Credit:        12000,
								Desc:          "test",
							},
						},
					},
				})
				assert.Nil(t, err)
			})

			t.Run("test rollback", func(t *testing.T) {

			})

		},
	)

}
