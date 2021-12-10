package sqlbase

import (
	"errors"
	"fmt"
	"github.com/raphaelvigee/go-paginate/cursor"
	"github.com/raphaelvigee/go-paginate/driver"
	"github.com/raphaelvigee/go-paginate/driver/base"
	"reflect"
	"strings"
)

type Executor interface {
	TakeFirst() (map[string]interface{}, error)
	CountPrevious(where string, args []interface{}) (int64, error)
	FindNext(query string, args []interface{}, limit int) ([]map[string]interface{}, error)

	Page(query string, args []interface{}, limit int) driver.Executor
}

type ExecutorFactoryArgs struct {
	base.ExecutorFactoryArgs
}

type Options struct {
	Columns         []Column
	ExecutorFactory func(args ExecutorFactoryArgs) Executor
}

type cursorEncoder struct {
	Columns []Column
}

func (d cursorEncoder) CursorEncode(input interface{}) (interface{}, error) {
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

func (d cursorEncoder) CursorDecode(input interface{}) (interface{}, error) {
	if input == nil {
		return nil, nil
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

func New(o Options) driver.Driver {
	for i := 0; i < len(o.Columns); i++ {
		if o.Columns[i].Reference == nil {
			o.Columns[i].Reference = func(column Column) (string, []interface{}) {
				return column.Name, nil
			}
		}

		if o.Columns[i].Placeholder == nil {
			o.Columns[i].Placeholder = func(Column) string {
				return "?"
			}
		}
	}

	return base.Driver{
		CursorEncoder: cursorEncoder{
			o.Columns,
		},
		ExecutorFactory: func(args base.ExecutorFactoryArgs) base.Executor {
			return sqlExecutor{
				ExecutorFactoryArgs: args,
				executor:            o.ExecutorFactory(ExecutorFactoryArgs{args}),
				columns:             o.Columns,
				pop:                 OpLt,
				nop:                 OpGt,
			}
		},
	}
}

type sqlExecutor struct {
	base.ExecutorFactoryArgs
	executor Executor
	columns  []Column

	pop Op
	nop Op
}

var _ base.Executor = (*sqlExecutor)(nil)

func (e sqlExecutor) TakeFirst() (interface{}, error) {
	return e.executor.TakeFirst()
}

func (e sqlExecutor) CountPrevious(cvalue interface{}) (int64, error) {
	pq, pargs := e.GenerateCondition(e.Cursor.Type, cvalue.(map[string]interface{}), e.pop)

	return e.executor.CountPrevious(pq, pargs)
}

func (e sqlExecutor) FindNext(cvalue interface{}, isFirst bool) ([]interface{}, error) {
	if isFirst {
		e.nop = e.nop.Inclusive()
	}

	nq, nargs := e.GenerateCondition(e.Cursor.Type, cvalue.(map[string]interface{}), e.nop)
	nvalues, err := e.executor.FindNext(nq, nargs, e.Cursor.Limit+1)
	if err != nil {
		return nil, err
	}

	arr := make([]interface{}, len(nvalues))
	for i := range nvalues {
		arr[i] = nvalues[i]
	}

	return arr, nil
}

func (e sqlExecutor) Page(sm, em interface{}) driver.Executor {
	sq, sargs := e.GenerateCondition(e.Cursor.Type, sm.(map[string]interface{}), e.nop.Inclusive())
	eq, eargs := e.GenerateCondition(e.Cursor.Type, em.(map[string]interface{}), e.pop.Inclusive())
	aargs := append(sargs, eargs...)

	return e.executor.Page(fmt.Sprintf("(%v AND %v)", sq, eq), aargs, e.Cursor.Limit)
}

func (e sqlExecutor) GenerateCondition(typ cursor.Type, values map[string]interface{}, op Op) (string, []interface{}) {
	s := "@@@previous@@@"
	origS := s

	argsa := make([][]interface{}, len(e.columns)*2)
	for i := len(e.columns) - 1; i >= 0; i-- {
		column := e.columns[i]

		cop := op
		if column.Order(typ) == OrderDesc {
			cop = cop.Opposite()
		}

		c, vars := column.Reference(column)
		v := values[column.Name]
		vp := column.Placeholder(column)

		// https://stackoverflow.com/a/38017813
		// col op ? AND (col op ? OR (previous))
		s = fmt.Sprintf("(%v %v %v AND (%v %v %v OR (%s)))", c, cop.Inclusive(), vp, c, cop, vp, s)

		args := make([]interface{}, 0)
		args = append(args, vars...) // c
		args = append(args, v)       // vp
		args = append(args, vars...) // c
		args = append(args, v)       // vp

		copy(argsa[i*2:], [][]interface{}{args})
	}

	s = strings.Replace(s, fmt.Sprintf(" OR (%v)", origS), "", 1) // Remove the useless root previous

	args := make([]interface{}, 0)
	for _, a := range argsa {
		args = append(args, a...)
	}

	return s, args
}
