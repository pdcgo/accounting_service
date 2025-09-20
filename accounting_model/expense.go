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
	ExpenseAt   time.Time
	CreatedAt   time.Time
}
