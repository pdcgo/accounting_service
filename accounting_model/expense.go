package accounting_model

import (
	"time"

	"github.com/pdcgo/schema/services/accounting_iface/v1"
)

type Expense struct {
	ID          uint `json:"id" gorm:"primarykey"`
	TeamID      uint
	CreatedByID uint
	Desc        string
	ExpenseType accounting_iface.ExpenseType
	ExpenseKey  string
	Amount      float64
	CreatedAt   time.Time
}

type ExpenseEntity struct{} // hanya untuk memberikan akses

// GetEntityID implements authorization_iface.Entity.
func (e ExpenseEntity) GetEntityID() string {
	return "accounting/expense_entity"
}
