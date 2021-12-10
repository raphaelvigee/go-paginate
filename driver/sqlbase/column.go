package sqlbase

import (
	"github.com/raphaelvigee/go-paginate/cursor"
)

type Column struct {
	// Column name
	Name string
	// ASC when false, DESC when true
	Desc bool
	// Prints column name in the SQL statement, default to the column name
	Reference func(column Column) (string, []interface{})
	// Prints the placeholder for prepared request, defaults to "?"
	Placeholder func(column Column) string
}

func (c Column) Order(t cursor.Type) Order {
	order := OrderAsc
	if c.Desc {
		order = OrderDesc
	}

	if t == cursor.Before {
		return order.Invert()
	}

	return order
}
