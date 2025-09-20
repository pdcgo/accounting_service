package accounting_service

import (
	"context"
	"errors"

	"connectrpc.com/connect"
	"github.com/pdcgo/schema/services/common/v1"
	"gorm.io/gorm"
)

type userServiceImpl struct {
	db *gorm.DB
}

// PublicUserIDs implements commonconnect.UserServiceHandler.
func (u *userServiceImpl) PublicUserIDs(
	ctx context.Context,
	req *connect.Request[common.PublicUserIDsRequest],
) (*connect.Response[common.PublicUserIDsResponse], error) {
	var err error
	result := &common.PublicUserIDsResponse{}
	res := &connect.Response[common.PublicUserIDsResponse]{
		Msg: result,
	}
	pay := req.Msg
	if len(pay.Ids) == 0 {
		return res, errors.New("ids empty")
	}

	query := u.db.WithContext(ctx)

	items := []*common.User{}
	err = query.
		Table("users u").
		Where("u.id = ?", pay.Ids).
		Find(&items).
		Error

	if err != nil {
		return res, err
	}

	result.Data = map[uint64]*common.User{}
	for _, d := range items {
		usr := d
		result.Data[usr.Id] = usr
	}

	return res, nil
}

func NewUserService(db *gorm.DB) *userServiceImpl {
	return &userServiceImpl{
		db: db,
	}
}
