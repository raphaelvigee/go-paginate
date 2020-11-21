package go_paginate

import (
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

	// Will default to cursor.MsgPackBase64EncoderDecoder
	CursorEncoderDecoder cursor.EncoderDecoder
}

func New(o Options) *Paginator {
	p := &Paginator{Options: o}

	if p.CursorEncoderDecoder == nil {
		p.CursorEncoderDecoder = cursor.MsgPackBase64EncoderDecoder()
	}

	p.Driver.Init()

	return p
}

type Paginator struct {
	Options
}

func (p *Paginator) Cursor(encoded string, typ cursor.Type, limit int) (cursor.Cursor, error) {
	data, err := p.CursorEncoderDecoder.Decode(encoded)
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

	sc, err := p.CursorEncoderDecoder.Encode(info.StartCursor)
	if err != nil {
		return Page{}, err
	}

	ec, err := p.CursorEncoderDecoder.Encode(info.EndCursor)
	if err != nil {
		return Page{}, err
	}

	return Page{
		Executor: dp,
		PageInfo: PageInfo{
			HasNextPage:     info.HasNextPage,
			HasPreviousPage: info.HasPreviousPage,
			StartCursor:     sc,
			EndCursor:       ec,
		},
		cursorFunc: func(i int64) (string, error) {
			rc, err := dp.Cursor(i)
			if err != nil {
				return "", err
			}

			return p.CursorEncoderDecoder.Encode(rc)
		},
	}, nil
}
