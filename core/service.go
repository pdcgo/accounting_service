package core

import (
	"context"

	"connectrpc.com/connect"
	"github.com/pdcgo/accounting_service/accounting_core"
	"github.com/pdcgo/schema/services/accounting_iface/v1"
	"github.com/pdcgo/shared/interfaces/authorization_iface"
	"gorm.io/gorm"
)

type coreServiceImpl struct {
	db   *gorm.DB
	auth authorization_iface.Authorization
}

// AccountKeyList implements accounting_ifaceconnect.LedgerServiceHandler.
func (l *coreServiceImpl) AccountKeyList(
	ctx context.Context,
	req *connect.Request[accounting_iface.AccountKeyListRequest],
) (*connect.Response[accounting_iface.AccountKeyListResponse], error) {
	var err error
	result := accounting_iface.AccountKeyListResponse{
		Keys: []*accounting_iface.AccountKeyItem{},
	}

	pay := req.Msg

	db := l.db.WithContext(ctx)
	err = db.
		Table("accounts a").
		Select([]string{
			"a.account_key as key",
			"a.coa",
			`case a.balance_type
				when 'd' then 1
				when 'c' then 2
				else 0
			end as balance_type`,
		}).
		Where("a.team_id = ?", pay.TeamId).
		Find(&result.Keys).
		Error

	if err != nil {
		return connect.NewResponse(&result), err
	}

	for _, item := range result.Keys {
		item.FilterExtra = filterExtra[accounting_core.AccountKey(item.Key)]
		if item.FilterExtra == nil {
			item.FilterExtra = &accounting_iface.AccountFilterExtra{
				CustomTag: true,
			}
		}
	}

	return connect.NewResponse(&result), err
}

func NewCoreService(db *gorm.DB, auth authorization_iface.Authorization) *coreServiceImpl {
	return &coreServiceImpl{
		db:   db,
		auth: auth,
	}
}

var filterExtra = map[accounting_core.AccountKey]*accounting_iface.AccountFilterExtra{
	accounting_core.PayableAccount: {
		Team: true,
	},
	accounting_core.SellingEstReceivableAccount: {
		Cs:        true,
		Shop:      true,
		CustomTag: true,
	},
	accounting_core.ReceivableAccount: {
		Team: true,
	},
	accounting_core.BorrowStockRevenueAccount: {
		Team:      true,
		Shop:      true,
		CustomTag: true,
	},
	accounting_core.StockLostAccount: {
		Team: true,
	},
	accounting_core.StockCostAccount: {
		Team:      true,
		Shop:      true,
		Supplier:  true,
		Cs:        true,
		CustomTag: true,
	},
	accounting_core.StockLostCostAccount: {
		Team: true,
	},
	accounting_core.StockBrokenAccount: {
		Team: true,
	},
	accounting_core.StockBorrowCostAmount: {
		Team:      true,
		Shop:      true,
		Supplier:  true,
		Cs:        true,
		CustomTag: true,
	},
	accounting_core.StockPendingAccount: {
		Team: true,
	},
	accounting_core.StockReadyAccount: {
		Team: true,
	},
	accounting_core.WarehouseCostAccount: {
		Team: true,
		Shop: true,
	},
	accounting_core.PaymentInTransitAccount: {
		Team: true,
	},
	accounting_core.SalesReturnRevenueAccount: {
		Cs:        true,
		Shop:      true,
		CustomTag: true,
	},
	accounting_core.SalesRevenueAccount: {
		Cs:        true,
		Shop:      true,
		CustomTag: true,
	},
}
