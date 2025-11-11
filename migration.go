package accounting_service

import (
	"log"
	"log/slog"

	"github.com/pdcgo/accounting_service/accounting_core"
	"github.com/pdcgo/accounting_service/accounting_model"
	"github.com/pdcgo/schema/services/accounting_iface/v1"
	"github.com/pdcgo/schema/services/common/v1"
	"github.com/pdcgo/schema/services/stock_iface/v1"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type SeedHandler func() error

func NewSeedHandler(
	db *gorm.DB,
) SeedHandler {
	return func() error {
		log.Println("seeding account service")
		// domain := authorization.NewDomainV2(db, authorization.RootDomain)
		// domain.RoleAddPermission()
		return nil
	}
}

type MigrationHandler func() error

func NewMigrationHandler(
	db *gorm.DB,
) MigrationHandler {
	return func() error {
		log.Println("migrating account service")
		err := db.AutoMigrate(
			&accounting_core.Account{},
			&accounting_core.JournalEntry{},
			&accounting_core.Transaction{},
			&accounting_core.AccountDailyBalance{},
			&accounting_core.AccountKeyDailyBalance{},
			&accounting_core.TransactionShop{},
			&accounting_core.TransactionSupplier{},
			&accounting_core.TransactionCustomerService{},
			&accounting_core.AccountingTag{},
			&accounting_core.TransactionTag{},
			&accounting_core.CsDailyBalance{},
			&accounting_core.ShopDailyBalance{},
			&accounting_core.SupplierDailyBalance{},
			&accounting_core.CustomLabelDailyBalance{},
			&accounting_core.TypeLabel{},
			&accounting_core.TransactionTypeLabel{},
			&accounting_core.TypeLabelDailyBalance{},

			&accounting_model.BankAccountV2{},
			&accounting_model.BankAccountLabel{},
			&accounting_model.BankAccountLabelRelation{},
			&accounting_model.BankTransferHistory{},
			&accounting_model.Expense{},
			&accounting_model.Payment{},
		)
		if err != nil {
			return err

		}

		// adding type label
		mplabels := []common.MarketplaceType{
			common.MarketplaceType_MARKETPLACE_TYPE_CUSTOM,
			common.MarketplaceType_MARKETPLACE_TYPE_LAZADA,
			common.MarketplaceType_MARKETPLACE_TYPE_MENGANTAR,
			common.MarketplaceType_MARKETPLACE_TYPE_TOKOPEDIA,
			common.MarketplaceType_MARKETPLACE_TYPE_TIKTOK,
			common.MarketplaceType_MARKETPLACE_TYPE_SHOPEE,
		}

		for _, mp := range mplabels {
			mplabel := &accounting_core.TypeLabel{
				Key:   accounting_iface.LabelKey_LABEL_KEY_MARKETPLACE,
				Label: common.MarketplaceType_name[int32(mp)],
			}

			err = db.
				Clauses(clause.OnConflict{DoNothing: true}).
				Create(&mplabel).Error

			if err != nil {
				slog.Error(err.Error())
			}

			// 	err = db.
			//  Model().
		}

		inboundsources := []stock_iface.InboundSource{
			stock_iface.InboundSource_INBOUND_SOURCE_RESTOCK,
			stock_iface.InboundSource_INBOUND_SOURCE_RETURN,
			stock_iface.InboundSource_INBOUND_SOURCE_TRANSFER,
		}

		for _, src := range inboundsources {
			srco := &accounting_core.TypeLabel{
				Key:   accounting_iface.LabelKey_LABEL_KEY_WAREHOUSE_TRANSACTION_TYPE,
				Label: stock_iface.InboundSource_name[int32(src)],
			}

			err = db.
				Clauses(clause.OnConflict{DoNothing: true}).
				Create(&srco).Error

			if err != nil {
				slog.Error(err.Error())
			}
		}

		return nil
	}
}
