package go_paginate

import (
	"encoding/base64"
	"github.com/raphaelvigee/go-paginate/cursor"
	"github.com/raphaelvigee/go-paginate/driver"
)

type PageInfo struct {
	HasNextPage     bool   `json:"hasNextPage"`
	HasPreviousPage bool   `json:"hasPreviousPage"`
	StartCursor     string `json:"startCursor"`
	EndCursor       string `json:"endCursor"`
}

type Page struct {
	driver.Executor
	PageInfo

	cursorFunc func(i int64) (string, error)
}

func (p Page) Cursor(i int64) (string, error) {
	return p.cursorFunc(i)
}

type Options struct {
	driver.Driver

	// Will default to cursor.Chain(cursor.MsgPack(), cursor.Base64(base64.StdEncoding))
	CursorMarshaller cursor.Marshaller
}

func New(o Options) *Paginator {
	p := &Paginator{Options: o}

	if p.CursorMarshaller == nil {
		p.CursorMarshaller = cursor.Chain(cursor.MsgPack(), cursor.Base64(base64.StdEncoding))
	}

	return p
}

type Paginator struct {
	Options
}

func (p *Paginator) Cursor(encoded string, typ cursor.Type, limit int) (cursor.Cursor, error) {
	data, err := p.CursorMarshaller.Unmarshal([]byte(encoded))
	if err != nil {
		return cursor.Cursor{}, err
	}

	value, err := p.Driver.CursorDecode(data)
	if err != nil {
		return cursor.Cursor{}, err
	}

	return cursor.Cursor{
		Type:  typ,
		Limit: limit,
		Value: value,
	}, nil
}

func (p *Paginator) Paginate(c cursor.Cursor, input interface{}) (Page, error) {
	dp, err := p.Driver.Paginate(c, input)
	if err != nil {
		return Page{}, err
	}

	info := dp.Info()

	sc, err := p.CursorMarshaller.Marshal(info.StartCursor)
	if err != nil {
		return Page{}, err
	}

	ec, err := p.CursorMarshaller.Marshal(info.EndCursor)
	if err != nil {
		return Page{}, err
	}

	return Page{
		Executor: dp,
		PageInfo: PageInfo{
			HasNextPage:     info.HasNextPage,
			HasPreviousPage: info.HasPreviousPage,
			StartCursor:     string(sc),
			EndCursor:       string(ec),
		},
		cursorFunc: func(i int64) (string, error) {
			rc, err := dp.Cursor(i)
			if err != nil {
				return "", err
			}

			m, err := p.CursorMarshaller.Marshal(rc)
			if err != nil {
				return "", err
			}

			return string(m), nil
		},
	}, nil
}
