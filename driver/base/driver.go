package base

import (
	"errors"
	"github.com/raphaelvigee/go-paginate/cursor"
	"github.com/raphaelvigee/go-paginate/driver"
)

type ExecutorFactoryArgs struct {
	Input  interface{}
	Cursor cursor.Cursor
}

var ErrNoResult = errors.New("no result")

type Executor interface {
	// Must throw ErrNoResult if no result can be found
	TakeFirst() (interface{}, error)
	CountPrevious(cvalue interface{}) (int64, error)
	FindNext(cvalue interface{}, isFirst bool) ([]interface{}, error)

	Page(sm interface{}, em interface{}) driver.Executor
}

type Driver struct {
	driver.CursorEncoder

	ExecutorFactory func(ExecutorFactoryArgs) Executor
}

var _ driver.Driver = (*Driver)(nil)

func (d Driver) Paginate(c cursor.Cursor, input interface{}) (driver.Page, error) {
	executor := d.ExecutorFactory(ExecutorFactoryArgs{
		Input:  input,
		Cursor: c,
	})
	limit := c.Limit

	cvalue := c.Value
	isFirst := cvalue == nil

	if isFirst {
		m, err := executor.TakeFirst()
		if err != nil {
			if errors.Is(err, ErrNoResult) {
				return noResultPage{}, nil
			}

			return nil, err
		}

		cvalue = m
	}

	pc, err := executor.CountPrevious(cvalue)
	if err != nil {
		return nil, err
	}

	nvalues, err := executor.FindNext(cvalue, isFirst)
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

	pExecutor := executor.Page(sm, em)

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
