package driver

import "github.com/raphaelvigee/go-paginate/cursor"

type Driver interface {
	// Runs when the paginator is created
	Init()

	// Used to convert data from the driver layer to the cursor layer
	// This should ideally be the smallest representation of the data
	// (ex: prefer literal, over array, over map...)
	CursorEncode(input interface{}) (interface{}, error)

	// Used to convert data from the cursor layer to the driver layer
	// input can be nil
	CursorDecode(input interface{}) (interface{}, error)

	Paginate(c cursor.Cursor, input interface{}) (Page, error)
}

type Executor interface {
	Query(dst interface{}) error
	Count() (int64, error)
}

type PageInfo struct {
	HasPreviousPage bool
	HasNextPage     bool
	StartCursor     interface{}
	EndCursor       interface{}
}

type Page interface {
	Executor
	Cursor(i int64) (interface{}, error)
	Info() PageInfo
}
