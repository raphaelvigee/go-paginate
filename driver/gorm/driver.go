package gorm

import (
	"errors"
	"fmt"
	"github.com/raphaelvigee/go-paginate/cursor"
	"github.com/raphaelvigee/go-paginate/driver"
	"gorm.io/gorm"
	"reflect"
)

type Driver struct {
	Columns []Column
}

var _ driver.Driver = (*Driver)(nil)

func (d Driver) CursorEncode(input interface{}) (interface{}, error) {
	switch reflect.TypeOf(input).Kind() {
	case reflect.Map:
		s := reflect.ValueOf(input)

		values := make([]interface{}, len(d.Columns))
		for i, column := range d.Columns {
			values[i] = s.MapIndex(reflect.ValueOf(column.Name)).Interface()
		}

		return values, nil
	default:
		return "", errors.New("gorm: cursor: encode: only map are handled")
	}
}

func (d Driver) CursorDecode(input interface{}) (interface{}, error) {
	if input == nil {
		return make(map[string]interface{}, 0), nil
	}

	switch reflect.TypeOf(input).Kind() {
	case reflect.Slice, reflect.Array:
		s := reflect.ValueOf(input)

		values := make(map[string]interface{}, 0)
		for i, column := range d.Columns {
			values[column.Name] = s.Index(i).Interface()
		}

		return values, nil
	default:
		return "", errors.New("gorm: cursor: decode: only slice/array are handled")
	}
}

func (d Driver) Init() {
	for i := 0; i < len(d.Columns); i++ {
		if d.Columns[i].Reference == nil {
			d.Columns[i].Reference = func(column Column) string {
				return column.Name
			}
		}

		if d.Columns[i].Placeholder == nil {
			d.Columns[i].Placeholder = func(Column) string {
				return "?"
			}
		}
	}
}

func (d Driver) fork(tx *gorm.DB) *gorm.DB {
	return tx.Session(&gorm.Session{})
}

func (d Driver) generateCondition(typ cursor.Type, values map[string]interface{}, op Op) (string, []interface{}) {
	s := "1=0"

	args := make([]interface{}, 0, len(d.Columns)*2)

	for i := len(d.Columns) - 1; i >= 0; i-- {
		column := d.Columns[i]

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

func (d Driver) Paginate(c cursor.Cursor, input interface{}) (driver.Page, error) {
	tx := input.(*gorm.DB)

	limit := c.Limit

	otx := d.fork(tx)
	columnNames := make([]string, len(d.Columns))
	for i, column := range d.Columns {
		order := column.Order(c.Type)

		otx = otx.Order(fmt.Sprintf("%v %v", column.Name, order))
		columnNames[i] = column.Name
	}

	tx.Logger.Info(tx.Statement.Context, "columns: %v", columnNames)

	stx := d.fork(otx).Select(columnNames)

	cvalue := c.Value.(map[string]interface{})
	isFirst := len(cvalue) == 0

	if isFirst {
		tx.Logger.Info(tx.Statement.Context, "first query")

		m, err := TakeMap(d.fork(stx).Limit(1))
		if err != nil {
			return nil, err
		}

		tx.Logger.Info(tx.Statement.Context, "first cvalue: %v", m)

		if len(m) == 0 {
			return &gormDriverPage{
				pageInfo: driver.PageInfo{
					HasNextPage:     false,
					HasPreviousPage: false,
					StartCursor:     nil,
					EndCursor:       nil,
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

	pq, pargs := d.generateCondition(c.Type, cvalue, pop)
	ptx := d.fork(stx).Where(pq, pargs...)
	nq, nargs := d.generateCondition(c.Type, cvalue, nop)
	ntx := d.fork(stx).Where(nq, nargs...)

	var pc int64
	if err := d.fork(ptx).Limit(1).Count(&pc).Error; err != nil {
		return nil, err
	}

	tx.Logger.Info(tx.Statement.Context, "pc: %v", pc)

	nvalues, err := FindMap(d.fork(ntx).Limit(limit + 1))
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
	var sc, ec interface{}
	if len(nvalues) > 0 {
		mi := len(nvalues) - 1

		si := 0
		ei := limit - 1
		if ei > mi {
			ei = mi
		}

		var err error
		sc, err = d.CursorEncode(nvalues[si])
		if err != nil {
			return nil, err
		}
		ec, err = d.CursorEncode(nvalues[ei])
		if err != nil {
			return nil, err
		}

		tx.Logger.Info(tx.Statement.Context, "sc: %v", sc)
		tx.Logger.Info(tx.Statement.Context, "ec: %v", ec)

		sm := nvalues[si]
		em := nvalues[ei]

		sq, sargs := d.generateCondition(c.Type, sm, nop.Inclusive())
		eq, eargs := d.generateCondition(c.Type, em, pop.Inclusive())
		aargs := append(sargs, eargs...)

		rtx = d.fork(otx).Limit(limit).Where(fmt.Sprintf("(%v) AND (%v)", sq, eq), aargs...)
	}

	return &gormDriverPage{
		tx: rtx,
		cursorFunc: func(i int64) (interface{}, error) {
			return d.CursorEncode(nvalues[i])
		},
		pageInfo: driver.PageInfo{
			HasNextPage:     hasNextPage,
			HasPreviousPage: hasPreviousPage,
			StartCursor:     sc,
			EndCursor:       ec,
		},
	}, nil
}
