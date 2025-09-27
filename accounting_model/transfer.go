package accounting_model

type BankTransfer struct{}

// GetEntityID implements authorization_iface.Entity.
func (b *BankTransfer) GetEntityID() string {
	return "accounting/bank_transfer"
}
