package gorm

import (
	"context"
	"fmt"
	"github.com/raphaelvigee/go-paginate/driver"
	"github.com/raphaelvigee/go-paginate/driver/base"
	"github.com/raphaelvigee/go-paginate/driver/sqlbase"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Column = sqlbase.Column

type Options struct {
	Columns []Column
}

func New(o Options) driver.Driver {
	return sqlbase.New(sqlbase.Options{
		Columns: o.Columns,
		ExecutorFactory: func(args sqlbase.ExecutorFactoryArgs) sqlbase.Executor {
			otx := fork(args.Input.(*gorm.DB))
			orders := clause.Expr{}
			selects := clause.Expr{}

			for _, column := range o.Columns {
				order := column.Order(args.Cursor.Type)

				s, vars := column.Reference(column)

				// Order
				if orders.SQL != "" {
					orders.SQL += ","
				}
				orders.SQL += fmt.Sprintf("%v %v", s, order)
				orders.Vars = append(orders.Vars, vars...)

				// Select
				if selects.SQL != "" {
					selects.SQL += ","
				}
				selects.SQL += s
				if column.Name != s {
					selects.SQL += " AS " + column.Name
				}
				selects.Vars = append(selects.Vars, vars...)
			}

			otx.Statement.AddClause(clause.OrderBy{
				Expression: orders,
			})

			stx := fork(otx)
			stx.Statement.AddClause(clause.Select{
				Expression: selects,
			})

			return gormExecutor{
				otx: otx,
				stx: stx,
			}
		},
	})
}

type gormExecutor struct {
	// Ordered transaction
	otx *gorm.DB
	// Ordered & selected transaction
	stx *gorm.DB
}

func (d gormExecutor) TakeFirst() (map[string]interface{}, error) {
	m, err := TakeMap(fork(d.stx).Limit(1))
	if err != nil {
		return nil, err
	}

	if len(m) == 0 {
		return nil, base.ErrNoResult
	}

	return m, nil
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
	ctx := tx.Statement.Context
	if ctx == nil { // Force stmt clone
		ctx = context.Background()
	}
	return tx.Session(&gorm.Session{Context: ctx})
}
