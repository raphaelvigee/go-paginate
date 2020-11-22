package gorm

import "github.com/raphaelvigee/go-paginate/cursor"

type Order string

const (
	OrderAsc  Order = "asc"
	OrderDesc Order = "desc"
)

func (o Order) Invert() Order {
	switch o {
	case OrderAsc:
		return OrderDesc
	case OrderDesc:
		return OrderAsc
	}

	panic("invalid order: " + string(o))
}

type Column struct {
	// Column name
	Name string
	// ASC when false, DESC when true
	Desc bool
	// Prints column name in the SQL statement, default to the column name
	Reference func(column Column) string
	// Prints the placeholder for prepared request, defaults to "?"
	Placeholder func(column Column) string
}

func (c Column) Order(t cursor.Type) Order {
	order := OrderAsc
	if c.Desc {
		order = OrderDesc
	}

	if t == cursor.Before {
		order = order.Invert()
	}

	return order
}
