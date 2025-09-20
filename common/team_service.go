package common

import (
	"context"

	"connectrpc.com/connect"
	"github.com/pdcgo/schema/services/common/v1"
	"github.com/pdcgo/shared/db_models"
	"gorm.io/gorm"
)

type teamServiceImpl struct {
	db *gorm.DB
}

// PublicTeamIDs implements commonconnect.TeamServiceHandler.
func (t *teamServiceImpl) PublicTeamIDs(
	ctx context.Context,
	req *connect.Request[common.PublicTeamIDsRequest],
) (*connect.Response[common.PublicTeamIDsResponse], error) {
	var err error
	result := common.PublicTeamIDsResponse{
		Data: map[uint64]*common.Team{},
	}

	db := t.db.WithContext(ctx)
	pay := req.Msg

	teams := []*common.Team{}
	err = db.
		Model(db_models.Team{}).
		Select([]string{
			"id",
			"name",
			"team_code",
			"type",
		}).
		Where("id in ?", pay.Ids).
		Find(&teams).
		Error

	for _, d := range teams {
		result.Data[d.Id] = d
	}

	return connect.NewResponse(&result), err
}

func NewTeamService(db *gorm.DB) *teamServiceImpl {
	return &teamServiceImpl{
		db: db,
	}
}
