package go_paginate

import (
	"fmt"
	"strings"
)

type Op string

const (
	OpGt Op = ">"
	OpLt Op = "<"
)

func (o Op) IsInclusive() bool {
	return strings.HasSuffix(string(o), "=")
}

func (o Op) Inclusive() Op {
	if o.IsInclusive() {
		return o
	}

	return Op(fmt.Sprintf("%v=", o))
}

func (o Op) Exclusive() Op {
	if o.IsInclusive() {
		return Op(o[0])
	}

	return o
}

func (o Op) Opposite() Op {
	switch o.Exclusive() {
	case OpGt:
		if o.IsInclusive() {
			return OpLt.Inclusive()
		}
		return OpLt
	case OpLt:
		if o.IsInclusive() {
			return OpGt.Inclusive()
		}
		return OpGt
	}

	panic("invalid op")
}
