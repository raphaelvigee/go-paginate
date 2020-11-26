package sqlbase

type Op string

const (
	OpGt  Op = ">"
	OpGte    = ">="
	OpLt     = "<"
	OpLte    = "<="
)

var inclusive = map[Op]Op{
	OpLt: OpLte,
	OpGt: OpGte,
}

var exclusive = map[Op]Op{
	OpLte: OpLt,
	OpGte: OpGt,
}

func (o Op) IsInclusive() bool {
	switch o {
	case OpLte, OpGte:
		return true
	}

	return false
}

func (o Op) Inclusive() Op {
	if o.IsInclusive() {
		return o
	}

	return inclusive[o]
}

func (o Op) Exclusive() Op {
	if !o.IsInclusive() {
		return o
	}

	return exclusive[o]
}

func (o Op) Opposite() Op {
	switch o {
	case OpLt:
		return OpGt
	case OpLte:
		return OpGte
	case OpGt:
		return OpLt
	case OpGte:
		return OpLte
	}

	panic("invalid op: " + string(o))
}
