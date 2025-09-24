package accounting_mock

import (
	"fmt"
	"testing"

	"github.com/pdcgo/accounting_service/accounting_core"
	"github.com/pdcgo/shared/pkg/moretest"
	"github.com/zeebo/assert"
	"gorm.io/gorm"
)

func PopulateAccountKey(db *gorm.DB, teamID uint) moretest.SetupFunc {
	return func(t *testing.T) func() error {
		var err error
		accs := accounting_core.DefaultSeedAccount()
		for _, acc := range accs {

			var old accounting_core.Account
			err =
				db.
					Model(&accounting_core.Account{}).
					Where("team_id = ?", teamID).
					Where("account_key = ?", acc.AccountKey).
					Find(&old).
					Error

			if err != nil {
				assert.Nil(t, err)
			}

			if old.ID != 0 {
				continue
			}

			err = accounting_core.
				NewCreateAccount(db).
				Create(
					acc.BalanceType,
					acc.Coa,
					teamID,
					acc.AccountKey,
					fmt.Sprintf("%s (%d)", acc.AccountKey, teamID),
				)

			if err != nil {
				assert.Nil(t, err)
			}

		}

		return nil
	}
}
