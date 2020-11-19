package go_paginate

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

	panic("invalid order")
}

type Column struct {
	Name string
	Desc bool
	// Prints column name in the SQL statement, default to the column name
	Reference func(column *Column) string
	// Prints the placeholder for prepared request, defaults to ?
	Placeholder func(column *Column) string
}

func (c Column) Order(t CursorType) Order {
	order := OrderAsc
	if c.Desc {
		order = OrderDesc
	}

	if t == CursorBefore {
		order = order.Invert()
	}

	return order
}
