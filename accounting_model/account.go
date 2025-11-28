package accounting_model

import (
	"time"

	"github.com/pdcgo/shared/db_models"
)

type BankAccountV2 struct {
	ID            uint      `json:"id" gorm:"primaryKey"`
	TeamID        uint      `json:"team_id"`
	AccountTypeID uint      `json:"account_type_id"`
	Name          string    `json:"name"`
	NumberID      string    `json:"number_id" gorm:"unique"`
	Balance       float64   `json:"balance"`
	Disabled      bool      `json:"disabled"`
	Deleted       bool      `json:"deleted"`
	CreatedAt     time.Time `json:"created_at"`
	DeletedAt     time.Time `json:"deleted_at"`

	AccountType db_models.AccountType `json:"account_type"`
}

// GetEntityID implements authorization_iface.Entity.
func (b *BankAccountV2) GetEntityID() string {
	return "accounting/bank_account/v2"
}

type BankAccountLabelRelation struct {
	AccountID uint `json:"business_account_id" gorm:"primaryKey"`
	LabelID   uint `json:"business_account_label_id" gorm:"primaryKey"`

	Account *BankAccountV2    `json:"-"`
	Label   *BankAccountLabel `json:"-"`
}

type BankAccountLabel struct {
	ID    uint   `json:"id" gorm:"primaryKey"`
	Key   string `json:"key" gorm:"index:key_unique,unique"`
	Value string `json:"value"`
}

type BankTransferHistory struct {
	ID            uint `json:"id" gorm:"primarykey"`
	TxID          uint `json:"tx_id"`
	TeamID        uint `json:"team_id"`
	FromAccountID uint `json:"from_account_id"`
	ToAccountID   uint `json:"to_account_id"`
	// TransferAt    time.Time `json:"transfer_at"`
	Amount    float64   `json:"amount"`
	FeeAmount float64   `json:"fee_amount"`
	Desc      string    `json:"desc"`
	Created   time.Time `json:"created"`

	FromAccount *BankAccountV2
	ToAccount   *BankAccountV2
	Team        *db_models.Team
}
