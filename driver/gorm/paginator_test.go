package gorm

import (
	"context"
	"fmt"
	"github.com/raphaelvigee/go-paginate"
	"github.com/raphaelvigee/go-paginate/cursor"
	"github.com/raphaelvigee/go-paginate/driver/sqlbase"
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	gormdb "gorm.io/gorm"
	"os"
	"strconv"
	"testing"
	"time"
)

type User struct {
	Id        string `gorm:"primarykey"`
	Name      string
	CreatedAt time.Time `gorm:"index"`
}

func SetupDb(models ...interface{}) (*gormdb.DB, context.CancelFunc) {
	db, err := gormdb.Open(sqlite.Open("file::memory:?cache=shared"), &gormdb.Config{NowFunc: func() time.Time { return time.Now().Local() }})
	if err != nil {
		panic(fmt.Sprintf("Failed to connect to db: %v", err))
	}

	sqlDB, err := db.DB()
	if err != nil {
		panic(fmt.Sprintf("Failed to get db: %v", err))
	}
	sqlDB.SetMaxOpenConns(1)

	db.AutoMigrate(models...)

	ctx, cancel := context.WithCancel(context.Background())

	return db.WithContext(ctx), cancel
}

func setup() (*gormdb.DB, context.CancelFunc) {
	db, teardown := SetupDb(&User{})

	db.Unscoped().Where("1=1").Delete(&User{})

	baseTime := time.Unix(0, 0).UTC()

	db.Create(&User{
		Name:      "u1",
		Id:        uuid.NewV4().String(),
		CreatedAt: baseTime.Add(4 * time.Hour),
	})

	db.Create(&User{
		Name:      "u2",
		Id:        uuid.NewV4().String(),
		CreatedAt: baseTime.Add(10 * time.Hour),
	})

	db.Create(&User{
		Name:      "u3",
		Id:        uuid.NewV4().String(),
		CreatedAt: baseTime.Add(1 * time.Hour),
	})

	db.Create(&User{
		Name:      "u4",
		Id:        uuid.NewV4().String(),
		CreatedAt: baseTime.Add(6 * time.Hour),
	})

	if ok, _ := strconv.ParseBool(os.Getenv("DEBUG")); ok {
		db = db.Debug()
	}

	return db, teardown
}

func placeholderValue(column sqlbase.Column) string {
	return "datetime(?)"
}

func columnName(c sqlbase.Column) string {
	return fmt.Sprintf("datetime(%v)", c.Name)
}

func printAll(tx *gormdb.DB) {
	var values []string
	tx.Session(&gormdb.Session{}).Order("`created_at` asc").Pluck("created_at", &values)
	tx.Logger.Info(tx.Statement.Context, "initial: %v", values)
}

type spec struct {
	hasPreviousPage bool
	hasNextPage     bool
	names           []string
}

func testPaginator(t *testing.T, columns []sqlbase.Column, typ cursor.Type, limit int, specs []spec) {
	db, teardown := setup()
	defer teardown()

	tx := db.Model(&User{})

	printAll(tx)

	pg := go_paginate.New(go_paginate.Options{
		Driver: New(Options{
			Columns: columns,
		}),
	})

	nextCursor := ""
	for i, s := range specs {
		t.Logf("Spec %v\n", i)
		csr, err := pg.Cursor(nextCursor, typ, limit)
		assert.NoError(t, err)

		res, err := pg.Paginate(csr, tx)
		assert.NoError(t, err)

		assert.Equal(t, s.hasPreviousPage, res.PageInfo.HasPreviousPage)
		assert.Equal(t, s.hasNextPage, res.PageInfo.HasNextPage)

		sc, _ := res.Cursor(0)
		assert.Equal(t, sc, res.PageInfo.StartCursor)
		ec, _ := res.Cursor(int64(limit - 1))
		assert.Equal(t, ec, res.PageInfo.EndCursor)

		c, err := res.Count()
		assert.NoError(t, err)
		assert.Equal(t, int64(len(s.names)), c)

		var users []User
		err = res.Query(&users)
		assert.NoError(t, err)

		assert.Len(t, users, len(s.names))
		for i, n := range s.names {
			assert.Equal(t, n, users[i].Name)
		}

		nextCursor = res.PageInfo.EndCursor
	}
}

var simpleColumns = []sqlbase.Column{
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

	pg := go_paginate.New(go_paginate.Options{
		Driver: New(Options{
			Columns: simpleColumns,
		}),
	})

	csr, err := pg.Cursor("", cursor.After, 2)
	assert.NoError(t, err)

	res, err := pg.Paginate(csr, tx)
	assert.NoError(t, err)

	assert.False(t, res.PageInfo.HasPreviousPage)
	assert.False(t, res.PageInfo.HasNextPage)
	assert.Empty(t, res.PageInfo.StartCursor)
	assert.Empty(t, res.PageInfo.EndCursor)

	c, err := res.Count()
	assert.NoError(t, err)
	assert.Equal(t, int64(0), c)

	var users []User
	err = res.Query(&users)
	assert.NoError(t, err)

	assert.Len(t, users, 0)
}

func TestFactory_After_Simple(t *testing.T) {
	testPaginator(t, simpleColumns, cursor.After, 2, []spec{
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
	testPaginator(t, simpleColumns, cursor.Before, 2, []spec{
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

var compositeColumns = []sqlbase.Column{
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
	testPaginator(t, compositeColumns, cursor.After, 2, []spec{
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
	testPaginator(t, compositeColumns, cursor.Before, 2, []spec{
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
