package go_paginate

import (
	"fmt"
	"gorm.io/gorm"
)

type PageInfo struct {
	HasNextPage     bool   `json:"hasNextPage"`
	HasPreviousPage bool   `json:"hasPreviousPage"`
	StartCursor     string `json:"startCursor"`
	EndCursor       string `json:"endCursor"`
}

type Response struct {
	PageInfo PageInfo

	Cursors []string
	Tx      *gorm.DB
}

type Paginator struct {
	Columns []*Column
}

func New(pg *Paginator) *Paginator {
	for _, c := range pg.Columns {
		if c.Reference == nil {
			c.Reference = func(column *Column) string {
				return column.Name
			}
		}

		if c.Placeholder == nil {
			c.Placeholder = func(*Column) string {
				return "?"
			}
		}
	}

	return pg
}

func fork(tx *gorm.DB) *gorm.DB {
	return tx.Session(&gorm.Session{})
}

func (r *Paginator) generateCondition(typ CursorType, values map[string]interface{}, op Op) (string, []interface{}) {
	s := "1=0"

	args := make([]interface{}, 0)

	for i := len(r.Columns) - 1; i >= 0; i-- {
		column := r.Columns[i]

		cop := op
		if column.Order(typ) == OrderDesc {
			cop = cop.Opposite()
		}

		c := column.Reference(column)
		v := values[column.Name]
		vp := column.Placeholder(column)

		// https://stackoverflow.com/a/38017813
		// col op ? AND (col op ? OR (previous))
		s = fmt.Sprintf("%v %v %v AND ( %v %v %v OR ( %s ) )", c, cop.Inclusive(), vp, c, cop, vp, s)
		args = append([]interface{}{v, v}, args...)
	}

	return s, args
}

func (r *Paginator) Paginate(c Cursor, tx *gorm.DB) (*Response, error) {
	limit := c.Limit

	otx := fork(tx)
	columnNames := make([]string, 0)
	for _, column := range r.Columns {
		order := column.Order(c.Type)

		otx = otx.Order(fmt.Sprintf("%v %v", column.Name, order))
		columnNames = append(columnNames, column.Name)
	}

	tx.Logger.Info(tx.Statement.Context, "columns: %v", columnNames)

	stx := fork(otx).Select(columnNames)

	cvalue := c.Value
	isFirst := len(cvalue) == 0

	if isFirst {
		tx.Logger.Info(tx.Statement.Context, "first query")

		m, err := TakeMap(fork(stx).Limit(1))
		if err != nil {
			return nil, err
		}

		tx.Logger.Info(tx.Statement.Context, "first cvalue: %v", m)

		if len(m) == 0 {
			return &Response{
				PageInfo: PageInfo{
					HasNextPage:     false,
					HasPreviousPage: false,
					StartCursor:     "",
					EndCursor:       "",
				},
			}, nil
		}

		cvalue = m
	}

	pop := OpLt
	nop := OpGt

	if isFirst {
		nop = nop.Inclusive()
	}

	tx.Logger.Info(tx.Statement.Context, "cvalue: %v", cvalue)

	pq, pargs := r.generateCondition(c.Type, cvalue, pop)
	ptx := fork(stx).Where(pq, pargs...)
	nq, nargs := r.generateCondition(c.Type, cvalue, nop)
	ntx := fork(stx).Where(nq, nargs...)

	var pc int64
	if err := fork(ptx).Limit(1).Count(&pc).Error; err != nil {
		return nil, err
	}

	tx.Logger.Info(tx.Statement.Context, "pc: %v", pc)

	nvalues, err := FindMap(fork(ntx).Limit(limit + 1))
	if err != nil {
		return nil, err
	}

	tx.Logger.Info(tx.Statement.Context, "nvalues: %v", nvalues)

	nc := len(nvalues)
	hasNextPage := nc > limit
	hasPreviousPage := pc > 0

	tx.Logger.Info(tx.Statement.Context, "hasPreviousPage: %v", hasPreviousPage)
	tx.Logger.Info(tx.Statement.Context, "hasNextPage: %v", hasNextPage)

	var rtx *gorm.DB
	var sc, ec string
	var cursors []string
	if len(nvalues) > 0 {
		for _, v := range nvalues {
			cursors = append(cursors, r.cursorString(v))
		}

		mi := len(nvalues) - 1

		si := 0
		ei := limit - 1
		if ei > mi {
			ei = mi
		}

		sc = cursors[si]
		ec = cursors[ei]

		tx.Logger.Info(tx.Statement.Context, "sc: %v", sc)
		tx.Logger.Info(tx.Statement.Context, "ec: %v", ec)

		sm := nvalues[si]
		em := nvalues[ei]

		sq, sargs := r.generateCondition(c.Type, sm, nop.Inclusive())
		eq, eargs := r.generateCondition(c.Type, em, pop.Inclusive())
		aargs := append(sargs, eargs...)

		rtx = fork(otx).Limit(limit).Where(fmt.Sprintf("(%v) AND (%v)", sq, eq), aargs...)
	}

	return &Response{
		PageInfo: PageInfo{
			HasNextPage:     hasNextPage,
			HasPreviousPage: hasPreviousPage,
			StartCursor:     sc,
			EndCursor:       ec,
		},
		Cursors: cursors,
		Tx:      rtx,
	}, nil
}
