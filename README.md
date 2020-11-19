# go-paginate

An efficient go data paginator.
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
    db, _ := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{NowFunc: func() time.Time { return time.Now().Local() }})
    db.AutoMigrate(&User{})

    // Define the pagination criterias
    pg := paginator.New(&paginator.Paginator{
        Columns: []*paginator.Column{
            {
                Name:        "created_at",
                // For SQLite the placeholder must be wrapped with `datetime()`
                Placeholder: func (*paginator.Column) string {
                    return "datetime(?)"
                },
                // For SQLite the column name must be wrapped with `datetime()`
                Reference:   func (c *paginator.Column) string {
                    return fmt.Sprintf("datetime(%v)", c.Name)
                },
            },
        },
    })
    
    // the cursor string must be empty for the first request
    c, _ := pg.Cursor("get the cursor from the request", paginator.CursorAfter, 2)

    // Create a transaction
    tx := db.Model(&User{})
    res, _ := pg.Paginate(c, tx)

    println(res.PageInfo.HasPreviousPage)
    println(res.PageInfo.HasNextPage)
    println(res.PageInfo.StartCursor)
    println(res.PageInfo.EndCursor)

    var users []User
    // Retrieve the results for the provided cursor/limit
    // res.tx is nil when no results are available
    if res.Tx != nil {
        if err := res.Tx.Find(&users).Error; err != nil {
            // Do something with the error
        }
    }
}
```
