# go-paginate
[![PkgGoDev](https://pkg.go.dev/badge/github.com/raphaelvigee/go-paginate)](https://pkg.go.dev/github.com/raphaelvigee/go-paginate)
![Test](https://github.com/raphaelvigee/go-paginate/workflows/Test/badge.svg)

An efficient go data cursor-based paginator.

- Plug and play
- Easy to use
- Fully customizable

```
go get github.com/raphaelvigee/go-paginate
```

## Why?

A lot of articles on the internet summarize very well the benefits of cursor-based pagination, but here are the highlights:

- It scales: Unlike `OFFSET`/`LIMIT`-based pagination doesn't scale well for large datasets
- Suitable for real-time data: having a fixed point in the flow of data prevents duplicates/missing entries

It does have an issue, it is hard to implement, that's why `go-paginate` exists :)

## Drivers

- [gorm](https://gorm.io):
    - Supports multiple columns with different orderings directions (ex: `ORDER BY id ASC, name DESC`)

- Implement your own: See [driver.Driver](driver/driver.go) and [base.Driver](driver/base/driver.go)

> Can't find what you are looking for? [Open an issue!](https://github.com/raphaelvigee/go-paginate/issues/new)

## Usage

### With `gorm`

> Errors omitted for brevity

Create the paginator, defining the criteria (columns and ordering):

```go
pg := paginator.New(paginator.Options{
    Driver: gorm.New(gorm.Options{
        Columns: []gorm.Column{
            {
                Name: "created_at",
            },
        },
    }),
})
```

Create the cursor instance, most likely from the request (in the initial request, the cursor is an empty string):

```go
c, err := pg.Cursor("<cursor from client>", cursor.After, 2)
```

Create a transaction with appropriate filtering etc and request the pagination info:

```go
tx := db.Model(&User{}).Where(...)
page, err := pg.Paginate(c, tx)

// That should be sent back to the client along with the data
fmt.Println(page.PageInfo.HasPreviousPage)
fmt.Println(page.PageInfo.HasNextPage)
fmt.Println(page.PageInfo.StartCursor)
fmt.Println(page.PageInfo.EndCursor)
```

Retrieve the underlying data for the page:

```go
var users []User
err := page.Query(&users)
```

A full working example can be found in [_examples/gorm](_examples/gorm/main.go).

### Custom cursor

By default, the cursor will be marshalled through `msgpack` for size concerns, and `base64` for portability.
One can choose to do differently (for example encrypting them...), see the implementation of `cursor.MsgPack` and `cursor.Base64`.

```go
pg := paginator.New(paginator.Options{
    ...
    CursorMarshaller: cursor.Chain(cursor.MsgPack(), cursor.Base64(base64.StdEncoding))
})
```

## Release

    TAG=v0.0.1 make tag
