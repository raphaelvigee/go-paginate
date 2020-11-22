package gorm

import (
	"fmt"
	"github.com/raphaelvigee/go-paginate/driver"
	"github.com/raphaelvigee/go-paginate/driver/sqlbase"
	"gorm.io/gorm"
)

type Column = sqlbase.Column

type Options struct {
	Columns []Column
}

func New(o Options) driver.Driver {
	return gormDriver{
		Driver: sqlbase.Driver{
			Columns: o.Columns,
			ExecutorFactory: func(args sqlbase.ExecutorFactoryArgs) sqlbase.Executor {
				otx := fork(args.Input.(*gorm.DB))
				columnNames := make([]string, len(o.Columns))

				for i, column := range o.Columns {
					order := column.Order(args.Cursor.Type)

					otx = otx.Order(fmt.Sprintf("%v %v", column.Name, order))
					columnNames[i] = column.Name
				}

				stx := fork(otx).Select(columnNames)

				return gormExecutor{
					otx: otx,
					stx: stx,
				}
			},
		},
	}
}

type gormExecutor struct {
	// Ordered transaction
	otx *gorm.DB
	// Ordered & selected transaction
	stx *gorm.DB
}

func (d gormExecutor) TakeFirst() (map[string]interface{}, error) {
	return TakeMap(fork(d.stx).Limit(1))
}

func (d gormExecutor) CountPrevious(where string, args []interface{}) (int64, error) {
	var pc int64
	return pc, fork(d.otx).Where(where, args...).Limit(1).Count(&pc).Error
}

func (d gormExecutor) FindNext(query string, args []interface{}, limit int) ([]map[string]interface{}, error) {
	return FindMap(fork(d.stx).Where(query, args...).Limit(limit))
}

func (d gormExecutor) Page(where string, args []interface{}, limit int) driver.Executor {
	tx := fork(d.otx).Where(where, args...).Limit(limit)

	return pageExecutor{tx: tx}
}

type pageExecutor struct {
	tx *gorm.DB
}

func (p pageExecutor) Query(dst interface{}) error {
	if p.tx == nil {
		return nil
	}

	return fork(p.tx).Find(dst).Error
}

func (p pageExecutor) Count() (int64, error) {
	if p.tx == nil {
		return 0, nil
	}

	var c int64
	err := fork(p.tx).Count(&c).Error

	return c, err
}

func fork(tx *gorm.DB) *gorm.DB {
	return tx.Session(&gorm.Session{})
}

type gormDriver struct {
	sqlbase.Driver
}

var _ driver.Driver = (*gormDriver)(nil)
