package common

import (
	"context"
	"math"

	"connectrpc.com/connect"
	"github.com/pdcgo/schema/services/common/v1"
	"github.com/pdcgo/shared/db_models"
	"gorm.io/gorm"
)

type teamServiceImpl struct {
	db *gorm.DB
}

// PublicTeamList implements commonconnect.TeamServiceHandler.
func (t *teamServiceImpl) PublicTeamList(
	ctx context.Context,
	req *connect.Request[common.PublicTeamListRequest],
) (*connect.Response[common.PublicTeamListResponse], error) {
	var err error

	pay := req.Msg
	db := t.db.WithContext(ctx)
	result := common.PublicTeamListResponse{
		Datas:    []*common.Team{},
		PageInfo: &common.PageInfo{},
	}

	page := pay.Page
	offset := page.Page*page.Limit - page.Limit

	tx := db.Model(&db_models.Team{}).Where("deleted = ?", false)
	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return &connect.Response[common.PublicTeamListResponse]{}, err
	}

	err = tx.
		Select([]string{
			"id",
			"name",
			"team_code",
			"type",
		}).
		Limit(int(page.Limit)).
		Offset(int(offset)).
		Find(&result.Datas).Error

	if err != nil {
		return nil, err
	}

	pageCount := math.Ceil(float64(total) / float64(page.Limit))
	result.PageInfo.CurrentPage = page.Page
	result.PageInfo.TotalPage = int64(pageCount)

	return connect.NewResponse(&result), nil
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
