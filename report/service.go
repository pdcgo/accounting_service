package report

import (
	"context"

	"cloud.google.com/go/cloudtasks/apiv2/cloudtaskspb"
	"github.com/googleapis/gax-go/v2"
	"github.com/pdcgo/shared/configs"
	"github.com/pdcgo/shared/interfaces/authorization_iface"
	"github.com/pdcgo/shared/pkg/ware_cache"
	"gorm.io/gorm"
)

type accountReportImpl struct {
	cfg       *configs.DispatcherConfig
	accConfig *configs.AccountingService
	db        *gorm.DB
	auth      authorization_iface.Authorization
	cache     ware_cache.Cache
	dispather ReportDispatcher
}

func NewAccountReportService(

	cfg *configs.DispatcherConfig,
	accConfig *configs.AccountingService,
	db *gorm.DB,
	auth authorization_iface.Authorization,
	cache ware_cache.Cache,
	dispather ReportDispatcher,
) *accountReportImpl {
	return &accountReportImpl{
		cfg:       cfg,
		accConfig: accConfig,
		db:        db,
		auth:      auth,
		cache:     cache,
		dispather: dispather,
	}
}

type ReportDispatcher func(ctx context.Context, req *cloudtaskspb.CreateTaskRequest, opts ...gax.CallOption) error
