package gorm

import (
	"github.com/raphaelvigee/go-paginate/driver"
	"gorm.io/gorm"
)

type gormDriverPage struct {
	tx         *gorm.DB
	pageInfo   driver.PageInfo
	cursorFunc func(i int64) (interface{}, error)
}

var _ driver.Page = (*gormDriverPage)(nil)

func (g gormDriverPage) Cursor(i int64) (interface{}, error) {
	return g.cursorFunc(i)
}

func (g gormDriverPage) Query(dst interface{}) error {
	if g.tx == nil {
		return nil
	}

	return g.tx.Find(dst).Error
}

func (g gormDriverPage) Count() (int64, error) {
	if g.tx == nil {
		return 0, nil
	}

	var c int64
	err := g.tx.Count(&c).Error

	return c, err
}

func (g gormDriverPage) Info() driver.PageInfo {
	return g.pageInfo
}
