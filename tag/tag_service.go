package tag

import (
	"context"
	"strings"

	"connectrpc.com/connect"
	"github.com/pdcgo/accounting_service/accounting_core"
	"github.com/pdcgo/schema/services/accounting_iface/v1"
	"github.com/pdcgo/shared/interfaces/authorization_iface"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type tagServiceImpl struct {
	db   *gorm.DB
	auth authorization_iface.Authorization
}

// TagCreate implements accounting_ifaceconnect.TagServiceHandler.
func (t *tagServiceImpl) TagCreate(
	ctx context.Context,
	req *connect.Request[accounting_iface.TagCreateRequest],
) (*connect.Response[accounting_iface.TagCreateResponse], error) {
	var err error

	pay := req.Msg
	err = t.
		auth.
		AuthIdentityFromHeader(req.Header()).
		Err()

	if err != nil {
		return &connect.Response[accounting_iface.TagCreateResponse]{}, err
	}

	db := t.db.WithContext(ctx)
	err = db.Transaction(func(tx *gorm.DB) error {
		for _, tname := range pay.Tags {
			tag := &accounting_core.AccountingTag{
				Name: accounting_core.SanityTag(tname),
			}
			err = tx.
				Clauses(
					clause.OnConflict{
						Columns:   []clause.Column{{Name: "name"}},
						DoNothing: true,
					},
				).
				Save(tag).
				Error
			if err != nil {
				return err
			}
		}

		return nil
	})

	if err != nil {
		return &connect.Response[accounting_iface.TagCreateResponse]{}, err
	}

	return &connect.Response[accounting_iface.TagCreateResponse]{}, nil
}

// TagList implements accounting_ifaceconnect.TagServiceHandler.
func (t *tagServiceImpl) TagList(
	ctx context.Context,
	req *connect.Request[accounting_iface.TagListRequest],
) (*connect.Response[accounting_iface.TagListResponse], error) {
	var err error
	result := &accounting_iface.TagListResponse{
		Tags: []string{},
	}
	pay := req.Msg
	err = t.
		auth.
		AuthIdentityFromHeader(req.Header()).
		Err()

	if err != nil {
		return &connect.Response[accounting_iface.TagListResponse]{}, err
	}

	db := t.db.WithContext(ctx)
	query := db.
		Model(&accounting_core.AccountingTag{}).
		Select("name")

	if pay.Q != "" {
		query = query.Where("LOWER(name) LIKE ?", "%"+strings.ToLower(pay.Q)+"%")
	}

	query = query.Order("name asc")

	err = query.
		Limit(int(pay.Limit)).
		Offset(int(pay.Offset)).
		Find(&result.Tags).
		Error

	return connect.NewResponse(result), err

}

func NewTagService(
	db *gorm.DB,
	auth authorization_iface.Authorization,
) *tagServiceImpl {
	return &tagServiceImpl{
		db:   db,
		auth: auth,
	}
}
