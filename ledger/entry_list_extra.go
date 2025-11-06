package ledger

import (
	"context"

	"connectrpc.com/connect"
	"github.com/pdcgo/accounting_service/accounting_core"
	"github.com/pdcgo/schema/services/accounting_iface/v1"
	"github.com/pdcgo/schema/services/common/v1"
	"gorm.io/gorm"
)

// EntryListExtra implements accounting_ifaceconnect.LedgerServiceHandler.
func (l *ledgerServiceImpl) EntryListExtra(
	ctx context.Context,
	req *connect.Request[accounting_iface.EntryListExtraRequest]) (*connect.Response[accounting_iface.EntryListExtraResponse], error) {
	var err error
	result := accounting_iface.EntryListExtraResponse{
		MapTag: map[uint64]*accounting_iface.TagList{},
	}

	db := l.db.WithContext(ctx)
	pay := req.Msg

	result.MapShop, err = GetShopList(db, pay.TxIds)
	if err != nil {
		return connect.NewResponse(&result), err
	}

	result.MapTypeLabel, err = GetTypeLabel(db, pay.TxIds)
	if err != nil {
		return connect.NewResponse(&result), err

	}

	return connect.NewResponse(&result), nil

}

type TypeLabelRow struct {
	accounting_core.TransactionTypeLabel
	accounting_iface.TypeLabel
}

func GetTypeLabel(db *gorm.DB, txIDs []uint64) (map[uint64]*accounting_iface.TypeLabelList, error) {
	var err error
	result := map[uint64]*accounting_iface.TypeLabelList{}

	query := db.
		Table("transaction_type_labels ttl").
		Joins("JOIN type_labels tl ON tl.id = ttl.type_label_id").
		Where("ttl.transaction_id IN ?", txIDs)

	rows, err := query.Rows()
	if err != nil {
		return result, err

	}
	defer rows.Close()

	for rows.Next() {
		d := TypeLabelRow{}
		err = db.ScanRows(rows, &d)
		if err != nil {
			return result, err
		}

		key := uint64(d.TransactionID)
		if result[key] == nil {
			result[key] = &accounting_iface.TypeLabelList{
				List: []*accounting_iface.TypeLabel{},
			}
		}

		result[key].List = append(result[key].List, &d.TypeLabel)
	}

	return result, nil

}

type ShopRow struct {
	accounting_core.TransactionShop
	common.Shop
}

func GetShopList(db *gorm.DB, txIDs []uint64) (map[uint64]*common.ShopList, error) {
	var err error
	result := map[uint64]*common.ShopList{}

	query := db.
		Table("transaction_shops ts").
		Joins("JOIN marketplaces m ON m.id = ts.shop_id").
		Where("ts.transaction_id IN ?", txIDs).
		Select([]string{
			"id",
			"ts.shop_id",
			"ts.transaction_id",
			"m.team_id",
			"m.mp_name as shop_name",
			"m.mp_username as shop_username",
			`
			case 
				when m.mp_type = 'tokopedia' then 2
				when m.mp_type = 'shopee' then 3
				when m.mp_type = 'lazada' then 5
				when m.mp_type = 'mengantar' then 6
				when m.mp_type = 'tiktok' then 4
				when m.mp_type = 'custom' then 1
				else 0

			end as marketplace_type
			`,
			"uri",
			//   string shop_name = 3;
			//   string shop_username = 4;
			//   MarketplaceType marketplace_type = 5;
			//   string uri = 6;
		})

	rows, err := query.Rows()
	if err != nil {
		return result, err
	}
	defer rows.Close()

	for rows.Next() {
		d := ShopRow{}
		err = db.ScanRows(rows, &d)
		if err != nil {
			return result, err
		}

		key := uint64(d.TransactionID)
		if result[key] == nil {
			result[key] = &common.ShopList{
				List: []*common.Shop{},
			}
		}

		result[key].List = append(result[key].List, &d.Shop)
	}

	return result, nil

}
