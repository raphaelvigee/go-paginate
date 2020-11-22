package sqlbase

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
