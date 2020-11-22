package sqlbase

import (
	"errors"
	"fmt"
	"github.com/raphaelvigee/go-paginate/cursor"
	"github.com/raphaelvigee/go-paginate/driver"
	"reflect"
)

type Executor interface {
	TakeFirst() (map[string]interface{}, error)
	CountPrevious(where string, args []interface{}) (int64, error)
	FindNext(query string, args []interface{}, limit int) ([]map[string]interface{}, error)

	Page(query string, args []interface{}, limit int) driver.Executor
}

type ExecutorFactoryArgs struct {
	Input  interface{}
	Cursor cursor.Cursor
}

type Driver struct {
	Columns         []Column
	ExecutorFactory func(args ExecutorFactoryArgs) Executor
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

func (d Driver) Paginate(c cursor.Cursor, input interface{}) (driver.Page, error) {
	executor := d.ExecutorFactory(ExecutorFactoryArgs{
		Input:  input,
		Cursor: c,
	})
	limit := c.Limit

	cvalue := c.Value.(map[string]interface{})
	isFirst := len(cvalue) == 0

	if isFirst {
		m, err := executor.TakeFirst()
		if err != nil {
			return nil, err
		}

		if len(m) == 0 {
			return noResultPage{}, nil
		}

		cvalue = m
	}

	pop := OpLt
	nop := OpGt

	if isFirst {
		nop = nop.Inclusive()
	}

	pq, pargs := d.GenerateCondition(c.Type, cvalue, pop)
	nq, nargs := d.GenerateCondition(c.Type, cvalue, nop)

	pc, err := executor.CountPrevious(pq, pargs)
	if err != nil {
		return nil, err
	}

	nvalues, err := executor.FindNext(nq, nargs, limit+1)
	if err != nil {
		return nil, err
	}

	nc := len(nvalues)
	hasPreviousPage := pc > 0
	hasNextPage := nc > limit

	if nc == 0 {
		return noResultPage{hasPreviousPage}, nil
	}

	mi := nc - 1

	si := 0
	ei := limit - 1
	if ei > mi {
		ei = mi
	}

	sm := nvalues[si]
	em := nvalues[ei]

	sc, err := d.CursorEncode(sm)
	if err != nil {
		return nil, err
	}
	ec, err := d.CursorEncode(em)
	if err != nil {
		return nil, err
	}

	sq, sargs := d.GenerateCondition(c.Type, sm, nop.Inclusive())
	eq, eargs := d.GenerateCondition(c.Type, em, pop.Inclusive())
	aargs := append(sargs, eargs...)

	pExecutor := executor.Page(fmt.Sprintf("(%v AND %v)", sq, eq), aargs, limit)

	return page{
		Executor: pExecutor,
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

type noResultPage struct {
	hasPrevious bool
}

func (n noResultPage) Query(interface{}) error {
	return nil
}

func (n noResultPage) Count() (int64, error) {
	return 0, nil
}

func (n noResultPage) Cursor(int64) (interface{}, error) {
	return nil, errors.New("no cursor available")
}

func (n noResultPage) Info() driver.PageInfo {
	return driver.PageInfo{
		HasPreviousPage: n.hasPrevious,
		HasNextPage:     false,
		StartCursor:     nil,
		EndCursor:       nil,
	}
}

type page struct {
	driver.Executor
	pageInfo   driver.PageInfo
	cursorFunc func(i int64) (interface{}, error)
}

func (p page) Cursor(i int64) (interface{}, error) {
	return p.cursorFunc(i)
}

func (p page) Info() driver.PageInfo {
	return p.pageInfo
}

func (d Driver) GenerateCondition(typ cursor.Type, values map[string]interface{}, op Op) (string, []interface{}) {
	s := "1=0"
	columns := d.Columns

	args := make([]interface{}, 0, len(columns)*2)

	for i := len(columns) - 1; i >= 0; i-- {
		column := columns[i]

		cop := op
		if column.Order(typ) == OrderDesc {
			cop = cop.Opposite()
		}

		c := column.Reference(column)
		v := values[column.Name]
		vp := column.Placeholder(column)

		// https://stackoverflow.com/a/38017813
		// col op ? AND (col op ? OR (previous))
		s = fmt.Sprintf("( %v %v %v AND ( %v %v %v OR ( %s ) ) )", c, cop.Inclusive(), vp, c, cop, vp, s)
		args = append([]interface{}{v, v}, args...)
	}

	return s, args
}
