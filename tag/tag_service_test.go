package tag_test

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	"github.com/pdcgo/accounting_service/accounting_core"
	"github.com/pdcgo/accounting_service/tag"
	"github.com/pdcgo/schema/services/accounting_iface/v1"
	"github.com/pdcgo/shared/authorization/authorization_mock"
	"github.com/pdcgo/shared/pkg/debugtool"
	"github.com/pdcgo/shared/pkg/moretest"
	"github.com/pdcgo/shared/pkg/moretest/moretest_mock"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

func TestTagCreate(t *testing.T) {
	var db gorm.DB

	var migrate moretest.SetupFunc = func(t *testing.T) func() error {
		err := db.AutoMigrate(
			&accounting_core.AccountingTag{},
		)
		assert.Nil(t, err)

		return nil
	}

	moretest.Suite(t, "TestTagCreate",
		moretest.SetupListFunc{
			moretest_mock.MockSqliteDatabase(&db),
			migrate,
		},
		func(t *testing.T) {

			authMock := authorization_mock.EmptyAuthorizationMock{}
			service := tag.NewTagService(&db, &authMock)

			ctx := context.Background()

			t.Run("success creating new tags", func(t *testing.T) {
				// Reset auth mock for this test case

				req := connect.NewRequest(&accounting_iface.TagCreateRequest{
					Tags: []string{"tag1", "tag2"},
				})

				_, err := service.TagCreate(ctx, req)
				assert.NoError(t, err)

				var tags []accounting_core.AccountingTag
				db.Find(&tags)

				assert.Len(t, tags, 2)
				assert.ElementsMatch(t, []string{"tag1", "tag2"}, []string{tags[0].Name, tags[1].Name})
			})

			t.Run("success with some existing tags", func(t *testing.T) {
				// Pre-insert a tag
				existingTag := accounting_core.AccountingTag{Name: "tag2"}
				db.Create(&existingTag)

				req := connect.NewRequest(&accounting_iface.TagCreateRequest{
					Tags: []string{"tag2", "tag3"},
				})

				_, err := service.TagCreate(ctx, req)
				assert.NoError(t, err)

				var tags []accounting_core.AccountingTag
				db.Order("name asc").Find(&tags)

				assert.Len(t, tags, 3)
				assert.Equal(t, "tag1", tags[0].Name)
				assert.Equal(t, "tag2", tags[1].Name)
				assert.Equal(t, "tag3", tags[2].Name)
			})

		},
	)

}

func TestTagList(t *testing.T) {
	var db gorm.DB

	var migrate moretest.SetupFunc = func(t *testing.T) func() error {
		err := db.AutoMigrate(
			&accounting_core.AccountingTag{},
		)
		assert.Nil(t, err)

		// Seed data
		tagsToCreate := []accounting_core.AccountingTag{
			{Name: "apple"},
			{Name: "banana"},
			{Name: "apricot"},
			{Name: "blueberry"},
			{Name: "cherry"},
		}
		err = db.Create(&tagsToCreate).Error
		assert.NoError(t, err)

		return nil
	}

	moretest.Suite(t, "TestTagList",
		moretest.SetupListFunc{
			moretest_mock.MockSqliteDatabase(&db),
			migrate,
		},
		func(t *testing.T) {
			authMock := authorization_mock.EmptyAuthorizationMock{}
			service := tag.NewTagService(&db, &authMock)
			ctx := context.Background()

			t.Run("success listing all tags", func(t *testing.T) {
				req := connect.NewRequest(&accounting_iface.TagListRequest{
					Limit: 100,
				})

				res, err := service.TagList(ctx, req)
				assert.NoError(t, err)
				assert.NotNil(t, res)

				debugtool.LogJson(res.Msg)

				expectedTags := []string{"apple", "apricot", "banana", "blueberry", "cherry"}
				assert.Equal(t, expectedTags, res.Msg.Tags)
			})

			t.Run("success listing with query", func(t *testing.T) {
				req := connect.NewRequest(&accounting_iface.TagListRequest{
					Q:     "ap",
					Limit: 100,
				})

				res, err := service.TagList(ctx, req)
				assert.NoError(t, err)
				assert.NotNil(t, res)

				expectedTags := []string{"apple", "apricot"}
				assert.Equal(t, expectedTags, res.Msg.Tags)
			})

			t.Run("success listing with pagination", func(t *testing.T) {
				req := connect.NewRequest(&accounting_iface.TagListRequest{
					Limit:  2,
					Offset: 1,
				})

				res, err := service.TagList(ctx, req)
				assert.NoError(t, err)
				assert.NotNil(t, res)

				expectedTags := []string{"apricot", "banana"}
				assert.Equal(t, expectedTags, res.Msg.Tags)
			})

		},
	)
}
