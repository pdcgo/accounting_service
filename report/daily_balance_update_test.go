package report

import (
	"encoding/json"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/pdcgo/accounting_service/accounting_core"
	"github.com/pdcgo/accounting_service/accounting_mock"
	"github.com/pdcgo/schema/services/accounting_iface/v1"
	"github.com/pdcgo/schema/services/report_iface/v1"
	"github.com/pdcgo/shared/authorization/authorization_mock"
	"github.com/pdcgo/shared/pkg/moretest"
	"github.com/pdcgo/shared/pkg/moretest/moretest_mock"
	"github.com/pdcgo/shared/pkg/ware_cache"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/gorm"
)

func loadAccount(db *gorm.DB, teamID uint, key accounting_core.AccountKey, account *accounting_core.Account) moretest.SetupFunc {

	return func(t *testing.T) func() error {
		err := db.
			Model(&accounting_core.Account{}).
			Where("account_key = ?", key).
			Where("team_id = ?", teamID).
			First(&account).
			Error

		assert.Nil(t, err)

		return nil
	}
}

func TestAccountKeyDailyBalance(t *testing.T) {
	var db gorm.DB
	var migrate moretest.SetupFunc = func(t *testing.T) func() error {
		err := db.AutoMigrate(
			&accounting_core.Account{},
			&accounting_core.AccountKeyDailyBalance{},
			&accounting_core.AccountDailyBalance{},
			&accounting_core.CsDailyBalance{},
			&accounting_core.ShopDailyBalance{},
			&accounting_core.SupplierDailyBalance{},
			&accounting_core.CustomLabelDailyBalance{},
		)

		assert.Nil(t, err)

		return nil
	}

	var hutangAcc2 accounting_core.Account

	moretest.Suite(t, "testing account key",
		moretest.SetupListFunc{
			moretest_mock.MockSqliteDatabase(&db),
			migrate,
			accounting_mock.PopulateAccountKey(&db, 1),
			accounting_mock.PopulateAccountKey(&db, 2),
			loadAccount(&db, 2, accounting_core.PayableAccount, &hutangAcc2),
		},
		func(t *testing.T) {

			reportService := NewAccountReportService(
				&db,
				&authorization_mock.EmptyAuthorizationMock{},
				ware_cache.NewLocalCache(),
			)

			t.Run("testing daily accountkey", func(t *testing.T) {
				_, err := reportService.DailyUpdateBalance(t.Context(), &connect.Request[report_iface.DailyUpdateBalanceRequest]{
					Msg: &report_iface.DailyUpdateBalanceRequest{
						LabelExtra: &report_iface.TxLabelExtra{
							TypeLabels: []*accounting_iface.TypeLabel{
								{
									Key:   accounting_iface.LabelKey_LABEL_KEY_MARKETPLACE,
									Label: "shopee",
								},
							},
						},
						Entries: []*report_iface.EntryPayload{
							{
								Id:            1,
								AccountId:     uint64(hutangAcc2.ID),
								TeamId:        1,
								EntryTime:     timestamppb.Now(),
								Debit:         12000,
								Desc:          "today",
								TransactionId: 1,
								Credit:        0,
							},
							{
								Id:            1,
								AccountId:     uint64(hutangAcc2.ID),
								TeamId:        1,
								EntryTime:     timestamppb.Now(),
								Debit:         0,
								Desc:          "today",
								TransactionId: 1,
								Credit:        12000,
							},
							{
								Id:            1,
								AccountId:     uint64(hutangAcc2.ID),
								TeamId:        1,
								EntryTime:     timestamppb.New(time.Now().AddDate(0, 0, -1)),
								Debit:         0,
								Desc:          "today",
								TransactionId: 1,
								Credit:        12000,
							},
						},
					},
				})

				assert.Nil(t, err)

				t.Run("testing accountkey", func(t *testing.T) {
					dAccounts := []*accounting_core.AccountKeyDailyBalance{}
					err = db.
						Model(&accounting_core.AccountKeyDailyBalance{}).
						Find(&dAccounts).
						Order("day desc").
						Error

					assert.Nil(t, err)
					assert.Len(t, dAccounts, 2)
					assert.NotEqual(t, 0, dAccounts[0].Balance)

				})
			})

		},
	)

}

func TestDailyUpdateBalance(t *testing.T) {
	var db gorm.DB

	var migrate moretest.SetupFunc = func(t *testing.T) func() error {
		err := db.AutoMigrate(
			&accounting_core.Account{},
			&accounting_core.AccountKeyDailyBalance{},
			&accounting_core.AccountDailyBalance{},
			&accounting_core.CsDailyBalance{},
			&accounting_core.ShopDailyBalance{},
			&accounting_core.SupplierDailyBalance{},
			&accounting_core.CustomLabelDailyBalance{},
		)

		assert.Nil(t, err)

		return nil
	}

	var seed moretest.SetupFunc = func(t *testing.T) func() error {
		accounts := []*accounting_core.Account{
			{
				ID:          3,
				AccountKey:  accounting_core.StockReadyAccount,
				TeamID:      1,
				BalanceType: accounting_core.DebitBalance,
				Coa:         accounting_core.ASSET,
				Name:        "Stock Ready",
				Created:     time.Now(),
			},
			{
				ID:          4,
				AccountKey:  accounting_core.StockBrokenAccount,
				TeamID:      1,
				BalanceType: accounting_core.DebitBalance,
				Coa:         accounting_core.ASSET,
				Name:        "Stock Broken",
				Created:     time.Now(),
			},
		}

		err := db.Save(&accounts).Error
		assert.Nil(t, err)

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

func TestDailyBalance(t *testing.T) {
	var db gorm.DB
	var entries accounting_core.JournalEntriesList
	entriesMap := map[uint]accounting_core.JournalEntriesList{}
	var accounts []*accounting_core.Account

	var migrate moretest.SetupFunc = func(t *testing.T) func() error {
		err := db.AutoMigrate(
			&accounting_core.JournalEntry{},
			&accounting_core.Account{},
			&accounting_core.AccountKeyDailyBalance{},
			&accounting_core.AccountDailyBalance{},
			&accounting_core.ShopDailyBalance{},
		)

		assert.Nil(t, err)

		return nil
	}

	var seed moretest.SetupFunc = func(t *testing.T) func() error {
		err := db.Save(&entries).Error
		assert.Nil(t, err)

		for _, dd := range entries {
			entry := dd
			entriesMap[entry.TransactionID] = append(entriesMap[entry.TransactionID], entry)
		}

		err = db.Save(&accounts).Error
		assert.Nil(t, err)

		return nil
	}

	// var migrate moretest.SetupFunc = func(t *testing.T) func() error {
	// 	err := db
	// }

	moretest.Suite(t, "test balance",
		moretest.SetupListFunc{
			moretest_mock.MockSqliteDatabase(&db),
			migrate,
			EntriesDump(&entries),
			AccountDump(&accounts),
			seed,
		},
		func(t *testing.T) {
			reportService := NewAccountReportService(
				&db,
				&authorization_mock.EmptyAuthorizationMock{},
				ware_cache.NewLocalCache(),
			)

			t.Run("testing updater daily", func(t *testing.T) {
				for txID, entrys := range entriesMap {

					entrypay := []*report_iface.EntryPayload{}
					for _, item := range entrys {
						entrypay = append(entrypay, &report_iface.EntryPayload{
							AccountId:     uint64(item.AccountID),
							TeamId:        uint64(item.TeamID),
							TransactionId: uint64(txID),
							EntryTime:     timestamppb.New(item.EntryTime),
							Debit:         item.Debit,
							Credit:        item.Credit,
							Desc:          item.Desc,
						})
					}

					_, err := reportService.DailyUpdateBalance(t.Context(), &connect.Request[report_iface.DailyUpdateBalanceRequest]{
						Msg: &report_iface.DailyUpdateBalanceRequest{
							LabelExtra: &report_iface.TxLabelExtra{
								ShopId: 1,
							},
							Entries: entrypay,
						},
					})

					assert.Nil(t, err)

				}
			})

			t.Run("check daily balance", func(t *testing.T) {
				dailys := []*accounting_core.AccountDailyBalance{}
				err := db.
					Model(&accounting_core.AccountDailyBalance{}).
					Where("journal_team_id = ?", 56).
					Where("account_id = ?", 2073).
					Order("day desc").
					Find(&dailys).
					Error
				assert.Nil(t, err)

				assert.Len(t, dailys, 2)

				// debugtool.LogJson(dailys)

				assert.Equal(t, 28715.5, dailys[0].Balance)

			})

		},
	)

}

func EntriesDump(entries *accounting_core.JournalEntriesList) moretest.SetupFunc {
	// select * from journal_entries je where je.transaction_id in (8537, 38574)
	entryRaw := `
[
	{
		"id" : 68207,
		"account_id" : 1585,
		"team_id" : 67,
		"transaction_id" : 8537,
		"created_by_id" : 411,
		"entry_time" : "2025-10-02T09:06:01.599Z",
		"debit" : 0,
		"credit" : 122820.5,
		"desc" : "stock diterima stock_accept#599947",
		"rollback" : null
	},
	{
		"id" : 68208,
		"account_id" : 1589,
		"team_id" : 67,
		"transaction_id" : 8537,
		"created_by_id" : 411,
		"entry_time" : "2025-10-02T09:06:01.599Z",
		"debit" : 95438,
		"credit" : 0,
		"desc" : "stock diterima stock_accept#599947",
		"rollback" : null
	},
	{
		"id" : 68209,
		"account_id" : 1583,
		"team_id" : 67,
		"transaction_id" : 8537,
		"created_by_id" : 411,
		"entry_time" : "2025-10-02T09:06:01.599Z",
		"debit" : 27382.5,
		"credit" : 0,
		"desc" : "stock diterima stock_accept#599947",
		"rollback" : null
	},
	{
		"id" : 68210,
		"account_id" : 2075,
		"team_id" : 56,
		"transaction_id" : 8537,
		"created_by_id" : 411,
		"entry_time" : "2025-10-02T09:06:01.616Z",
		"debit" : 0,
		"credit" : 122820.5,
		"desc" : "stock diterima stock_accept#599947",
		"rollback" : null
	},
	{
		"id" : 68211,
		"account_id" : 2079,
		"team_id" : 56,
		"transaction_id" : 8537,
		"created_by_id" : 411,
		"entry_time" : "2025-10-02T09:06:01.616Z",
		"debit" : 95438,
		"credit" : 0,
		"desc" : "stock diterima stock_accept#599947",
		"rollback" : null
	},
	{
		"id" : 68212,
		"account_id" : 2073,
		"team_id" : 56,
		"transaction_id" : 8537,
		"created_by_id" : 411,
		"entry_time" : "2025-10-02T09:06:01.616Z",
		"debit" : 27382.5,
		"credit" : 0,
		"desc" : "stock diterima stock_accept#599947",
		"rollback" : null
	},
	{
		"id" : 424920,
		"account_id" : 1583,
		"team_id" : 67,
		"transaction_id" : 38574,
		"created_by_id" : 411,
		"entry_time" : "2025-10-10T07:42:49.981Z",
		"debit" : 1333,
		"credit" : 0,
		"desc" : "stock diterima stock_accept#623458",
		"rollback" : null
	},
	{
		"id" : 424921,
		"account_id" : 1585,
		"team_id" : 67,
		"transaction_id" : 38574,
		"created_by_id" : 411,
		"entry_time" : "2025-10-10T07:42:49.981Z",
		"debit" : 0,
		"credit" : 42038,
		"desc" : "stock diterima stock_accept#623458",
		"rollback" : null
	},
	{
		"id" : 424922,
		"account_id" : 1589,
		"team_id" : 67,
		"transaction_id" : 38574,
		"created_by_id" : 411,
		"entry_time" : "2025-10-10T07:42:49.981Z",
		"debit" : 40705,
		"credit" : 0,
		"desc" : "stock diterima stock_accept#623458",
		"rollback" : null
	},
	{
		"id" : 424923,
		"account_id" : 2075,
		"team_id" : 56,
		"transaction_id" : 38574,
		"created_by_id" : 411,
		"entry_time" : "2025-10-10T07:42:50.006Z",
		"debit" : 0,
		"credit" : 42038,
		"desc" : "stock diterima stock_accept#623458",
		"rollback" : null
	},
	{
		"id" : 424924,
		"account_id" : 2079,
		"team_id" : 56,
		"transaction_id" : 38574,
		"created_by_id" : 411,
		"entry_time" : "2025-10-10T07:42:50.006Z",
		"debit" : 40705,
		"credit" : 0,
		"desc" : "stock diterima stock_accept#623458",
		"rollback" : null
	},
	{
		"id" : 424925,
		"account_id" : 2073,
		"team_id" : 56,
		"transaction_id" : 38574,
		"created_by_id" : 411,
		"entry_time" : "2025-10-10T07:42:50.006Z",
		"debit" : 1333,
		"credit" : 0,
		"desc" : "stock diterima stock_accept#623458",
		"rollback" : null
	}
]	
	`

	return func(t *testing.T) func() error {
		err := json.Unmarshal([]byte(entryRaw), entries)
		assert.Nil(t, err)
		return nil
	}

}

func AccountDump(accounts *[]*accounting_core.Account) moretest.SetupFunc {
	accountRaw := `
[
	{
		"id" : 1583,
		"account_key" : "stock_lost",
		"team_id" : 56,
		"coa" : 10,
		"balance_type" : "d",
		"name" : "stock_lost (Team 4)",
		"created" : "2025-09-17T07:26:49.512Z"
	},
	{
		"id" : 1585,
		"account_key" : "stock_pending",
		"team_id" : 56,
		"coa" : 10,
		"balance_type" : "d",
		"name" : "stock_pending (Team 4)",
		"created" : "2025-09-17T07:26:49.538Z"
	},
	{
		"id" : 1589,
		"account_key" : "stock_ready",
		"team_id" : 56,
		"coa" : 10,
		"balance_type" : "d",
		"name" : "stock_ready (Team 4)",
		"created" : "2025-09-17T07:26:49.587Z"
	},
	{
		"id" : 2075,
		"account_key" : "stock_pending",
		"team_id" : 67,
		"coa" : 10,
		"balance_type" : "d",
		"name" : "stock_pending (Febri Warehouse)",
		"created" : "2025-09-17T07:26:56.405Z"
	},
	{
		"id" : 2079,
		"account_key" : "stock_ready",
		"team_id" : 67,
		"coa" : 10,
		"balance_type" : "d",
		"name" : "stock_ready (Febri Warehouse)",
		"created" : "2025-09-17T07:26:56.447Z"
	},
	{
		"id" : 2073,
		"account_key" : "stock_lost",
		"team_id" : 67,
		"coa" : 10,
		"balance_type" : "d",
		"name" : "stock_lost (Febri Warehouse)",
		"created" : "2025-09-17T07:26:56.383Z"
	}
]
	
	`

	return func(t *testing.T) func() error {
		err := json.Unmarshal([]byte(accountRaw), accounts)
		assert.Nil(t, err)
		return nil
	}
}
