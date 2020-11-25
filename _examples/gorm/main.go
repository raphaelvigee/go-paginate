package main

import (
	"fmt"
	paginator "github.com/raphaelvigee/go-paginate"
	"github.com/raphaelvigee/go-paginate/cursor"
	"github.com/raphaelvigee/go-paginate/driver/gorm"
	uuid "github.com/satori/go.uuid"
	"gorm.io/driver/sqlite"
	gormdb "gorm.io/gorm"
	"time"
)

type User struct {
	Id        string `gorm:"primarykey"`
	Name      string
	CreatedAt time.Time `gorm:"index"`
}

func main() {
	// Open the DB
	db, err := gormdb.Open(sqlite.Open("file::memory:?cache=shared"), &gormdb.Config{NowFunc: func() time.Time { return time.Now().Local() }})
	if err != nil {
		panic(err)
	}
	db.AutoMigrate(&User{})

	// Add some data
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

	// Define the pagination criteria
	pg := paginator.New(paginator.Options{
		Driver: gorm.New(gorm.Options{
			Columns: []gorm.Column{
				{
					Name: "created_at",
					// For SQLite the placeholder must be wrapped with `datetime()`
					Placeholder: func(gorm.Column) string {
						return "datetime(?)"
					},
					// For SQLite the column name must be wrapped with `datetime()`
					Reference: func(c gorm.Column) string {
						return fmt.Sprintf("datetime(%v)", c.Name)
					},
				},
			},
		}),
	})

	// This would typically come from the request
	cursorString := "" // must be empty for the first request
	cursorType := cursor.After
	cursorLimit := 2

	c, err := pg.Cursor(cursorString, cursorType, cursorLimit)
	if err != nil {
		panic(err)
	}

	// Create a transaction
	tx := db.Model(&User{})
	page, err := pg.Paginate(c, tx)
	if err != nil {
		panic(err)
	}

	fmt.Println(page.PageInfo.HasPreviousPage)
	fmt.Println(page.PageInfo.HasNextPage)
	fmt.Println(page.PageInfo.StartCursor)
	fmt.Println(page.PageInfo.EndCursor)

	// Retrieve the results for the current page
	var users []User
	if err := page.Query(&users); err != nil {
		panic(err)
	}

	fmt.Println(len(users)) // Should print 2
}
