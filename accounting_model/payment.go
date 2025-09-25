package accounting_model

import (
	"time"

	"github.com/pdcgo/schema/services/payment_iface/v1"
)

type Payment struct {
	ID          uint                        `json:"id" gorm:"primarykey"`
	FromTeamID  uint                        `json:"from_team_id"`
	ToTeamID    uint                        `json:"to_team_id"`
	Status      payment_iface.PaymentStatus `json:"status"`
	PaymentType payment_iface.PaymentType   `json:"payment_type"`
	Amount      float64                     `json:"amount"`
	CreatedByID uint                        `json:"created_by_id"`
	CreatedAt   time.Time                   `json:"created_at"`
	AcceptedAt  time.Time                   `json:"accepted_at"`
}

// GetEntityID implements authorization_iface.Entity.
func (p *Payment) GetEntityID() string {
	return "accounting/payment"
}
