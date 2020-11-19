package go_paginate

import (
	"context"
	"fmt"
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"testing"
	"time"
)

type User struct {
	Id        string `gorm:"primarykey"`
	Name      string
	CreatedAt time.Time `gorm:"index"`
}

func SetupDb(models ...interface{}) (*gorm.DB, context.CancelFunc) {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{NowFunc: func() time.Time { return time.Now().Local() }})
	if err != nil {
		panic(fmt.Sprintf("Failed to connect to db: %v", err))
	}

	db.AutoMigrate(models...)

	ctx, cancel := context.WithCancel(context.Background())

	return db.WithContext(ctx), cancel
}

func setup() (*gorm.DB, context.CancelFunc) {
	db, teardown := SetupDb(&User{})

	db.Unscoped().Where("1=1").Delete(&User{})

	base := time.Unix(0, 0).UTC()

	db.Create(&User{
		Name:      "u1",
		Id:        uuid.NewV4().String(),
		CreatedAt: base.Add(4 * time.Hour),
	})

	db.Create(&User{
		Name:      "u2",
		Id:        uuid.NewV4().String(),
		CreatedAt: base.Add(10 * time.Hour),
	})

	db.Create(&User{
		Name:      "u3",
		Id:        uuid.NewV4().String(),
		CreatedAt: base.Add(1 * time.Hour),
	})

	db.Create(&User{
		Name:      "u4",
		Id:        uuid.NewV4().String(),
		CreatedAt: base.Add(6 * time.Hour),
	})

	db = db.Debug()

	return db, teardown
}

func placeholderValue(*Column) string {
	return "datetime(?)"
}

func columnName(c *Column) string {
	return fmt.Sprintf("datetime(%v)", c.Name)
}

func printAll(tx *gorm.DB) {
	var values []string
	tx.Session(&gorm.Session{}).Order("`created_at` asc").Pluck("created_at", &values)
	tx.Logger.Info(tx.Statement.Context, "initial: %v", values)
}

type spec struct {
	hasPreviousPage bool
	hasNextPage     bool
	names           []string
}

func testPaginator(t *testing.T, columns []*Column, typ CursorType, limit int, specs []spec) {
	db, teardown := setup()
	defer teardown()

	tx := db.Model(&User{})

	printAll(tx)

	pg := New(&Paginator{
		Columns: columns,
	})

	nextCursor := ""
	for _, s := range specs {
		cursor, err := pg.Cursor(nextCursor, typ, limit)
		assert.NoError(t, err)

		res, err := pg.Paginate(cursor, tx)
		assert.NoError(t, err)

		assert.Equal(t, s.hasPreviousPage, res.PageInfo.HasPreviousPage)
		assert.Equal(t, s.hasNextPage, res.PageInfo.HasNextPage)

		assert.NotNil(t, res.Tx)

		var users []User
		err = res.Tx.Find(&users).Error
		assert.NoError(t, err)

		assert.Len(t, users, len(s.names))
		for i, n := range s.names {
			assert.Equal(t, n, users[i].Name)
		}

		nextCursor = res.PageInfo.EndCursor
	}
}

var simpleColumns = []*Column{
	{
		Name:        "created_at",
		Placeholder: placeholderValue,
		Reference:   columnName,
	},
}

func TestFactory_Empty(t *testing.T) {
	db, teardown := SetupDb(&User{})
	defer teardown()

	db.Where("1=1").Delete(&User{})

	tx := db.Model(&User{})

	pg := New(&Paginator{
		Columns: simpleColumns,
	})

	c, err := pg.Cursor("", CursorAfter, 2)
	assert.NoError(t, err)

	res, err := pg.Paginate(c, tx)
	assert.NoError(t, err)

	assert.False(t, res.PageInfo.HasPreviousPage)
	assert.False(t, res.PageInfo.HasNextPage)
	assert.Empty(t, res.PageInfo.StartCursor)
	assert.Empty(t, res.PageInfo.EndCursor)

	assert.Nil(t, res.Tx)
}

func TestFactory_After_Simple(t *testing.T) {
	testPaginator(t, simpleColumns, CursorAfter, 2, []spec{
		{
			hasPreviousPage: false,
			hasNextPage:     true,
			names:           []string{"u3", "u1"},
		},
		{
			hasPreviousPage: true,
			hasNextPage:     false,
			names:           []string{"u4", "u2"},
		},
	})
}

func TestFactory_Before_Simple(t *testing.T) {
	testPaginator(t, simpleColumns, CursorBefore, 2, []spec{
		{
			hasPreviousPage: false,
			hasNextPage:     true,
			names:           []string{"u2", "u4"},
		},
		{
			hasPreviousPage: true,
			hasNextPage:     false,
			names:           []string{"u1", "u3"},
		},
	})
}

var compositeColumns = []*Column{
	{
		Name:        "created_at",
		Desc:        true,
		Placeholder: placeholderValue,
		Reference:   columnName,
	},
	{
		Name: "id",
	},
}

func TestFactory_After_Composite(t *testing.T) {
	testPaginator(t, compositeColumns, CursorAfter, 2, []spec{
		{
			hasPreviousPage: false,
			hasNextPage:     true,
			names:           []string{"u2", "u4"},
		},
		{
			hasPreviousPage: true,
			hasNextPage:     false,
			names:           []string{"u1", "u3"},
		},
	})
}

func TestFactory_Before_Composite(t *testing.T) {
	testPaginator(t, compositeColumns, CursorBefore, 2, []spec{
		{
			hasPreviousPage: false,
			hasNextPage:     true,
			names:           []string{"u3", "u1"},
		},
		{
			hasPreviousPage: true,
			hasNextPage:     false,
			names:           []string{"u4", "u2"},
		},
	})
}
