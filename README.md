# go-paginate

An efficient go data cursor-based paginator.
For now only supports [gorm](https://gorm.io), support for any data source to be added later™️, PRs welcome.

- Supports multiple columns with multiple orderings
- Plug and play
- Easy to use
- Efficient

## Usage

```go
package main

import (
    "fmt"
    paginator "github.com/raphaelvigee/go-paginate"
    uuid "github.com/satori/go.uuid"
    "gorm.io/driver/sqlite"
    "gorm.io/gorm"
    "time"
)

type User struct {
    Id        string `gorm:"primarykey"`
    Name      string
    CreatedAt time.Time `gorm:"index"`
}

func main() {
    // Errors omitted for brevity

    // Open the DB
    db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{NowFunc: func() time.Time { return time.Now().Local() }})
    if err != nil {
        panic(err)
    }
    db.AutoMigrate(&User{})

    // Add some data
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

    // Define the pagination criterias
    pg := paginator.New(&paginator.Paginator{
        Columns: []*paginator.Column{
            {
                Name: "created_at",
                // For SQLite the placeholder must be wrapped with `datetime()`
                Placeholder: func(*paginator.Column) string {
                    return "datetime(?)"
                },
                // For SQLite the column name must be wrapped with `datetime()`
                Reference: func(c *paginator.Column) string {
                    return fmt.Sprintf("datetime(%v)", c.Name)
                },
            },
        },
    })

    // This would typically come from the request
    cursorString := "" // must be empty for the first request
    cursorType := paginator.CursorAfter
    cursorLimit := 2

    c, err := pg.Cursor(cursorString, cursorType, cursorLimit)
    if err != nil {
        panic(err)
    }

    // Create a transaction
    tx := db.Model(&User{})
    res, err := pg.Paginate(c, tx)
    if err != nil {
        panic(err)
    }

    fmt.Println(res.PageInfo.HasPreviousPage)
    fmt.Println(res.PageInfo.HasNextPage)
    fmt.Println(res.PageInfo.StartCursor)
    fmt.Println(res.PageInfo.EndCursor)

    // Retrieve the results for the provided cursor/limit
    var users []User
    // res.Tx is nil when no results are available
    if res.Tx != nil {
        if err := res.Tx.Find(&users).Error; err != nil {
            panic(err)
        }
    }

    fmt.Println(len(users)) // Should print 2
}
```

## Roadmap

- Support different datasources (not only gorm)
- Custom cursors encode/decode

# Release

    TAG=v1.0.0 make tag
