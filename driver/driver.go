package driver

import "github.com/raphaelvigee/go-paginate/cursor"

// Allows to transform the driver cursor value data to a potential smaller
// form for marshaling (ex: use an array instead of a map, since we know
// the column names, see sqlbase.cursorEncoder for details)
type CursorEncoder interface {
	// Used to convert data from the driver layer to the cursor layer
	// This should ideally be the smallest representation of the data
	// (ex: prefer literal, over array, over map...)
	CursorEncode(input interface{}) (interface{}, error)

	// Used to convert data from the cursor layer to the driver layer
	// Should return nil if the input is nil
	// input can be nil
	CursorDecode(input interface{}) (interface{}, error)
}

type Driver interface {
	CursorEncoder

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
