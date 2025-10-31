package accounting_model

import "time"

type BankTransfer struct{}

// GetEntityID implements authorization_iface.Entity.
func (b *BankTransfer) GetEntityID() string {
	return "accounting/bank_transfer"
}

type Transfer struct {
	ID            uint `json:"id" gorm:"primarykey"`
	FromAccountID uint `json:"from_account_id"`
	ToAccountID   uint `json:"to_account_id"`
	Amount        float64
	FeeAmount     float64
	Description   string
	CreatedAt     time.Time
}
