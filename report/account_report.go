package report

import (
	"github.com/pdcgo/shared/interfaces/authorization_iface"
	"gorm.io/gorm"
)

type accountReportImpl struct {
	db   *gorm.DB
	auth authorization_iface.Authorization
}

func NewAccountReportService(db *gorm.DB, auth authorization_iface.Authorization) *accountReportImpl {
	return &accountReportImpl{
		db:   db,
		auth: auth,
	}
}
