package report

import (
	"github.com/pdcgo/shared/interfaces/authorization_iface"
	"github.com/pdcgo/shared/pkg/ware_cache"
	"gorm.io/gorm"
)

type accountReportImpl struct {
	db    *gorm.DB
	auth  authorization_iface.Authorization
	cache ware_cache.Cache
}

func NewAccountReportService(
	db *gorm.DB,
	auth authorization_iface.Authorization,
	cache ware_cache.Cache,
) *accountReportImpl {
	return &accountReportImpl{
		db:    db,
		auth:  auth,
		cache: cache,
	}
}
