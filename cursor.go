package go_paginate

import (
	"encoding/base64"
	"github.com/vmihailenco/msgpack/v5"
)

type CursorType int

const (
	CursorBefore CursorType = 1 << iota
	CursorAfter
)

type Cursor struct {
	Limit int
	Type  CursorType
	Value map[string]interface{}
}

func (r *Paginator) Cursor(encoded string, typ CursorType, limit int) (Cursor, error) {
	values := make(map[string]interface{}, 0)

	if len(encoded) > 0 {
		decoded, err := base64.StdEncoding.DecodeString(encoded)
		if err != nil {
			return Cursor{}, err
		}

		var valuesArr []interface{}
		if err = msgpack.Unmarshal(decoded, &valuesArr); err != nil {
			return Cursor{}, err
		}

		for i, column := range r.Columns {
			values[column.Name] = valuesArr[i]
		}
	}

	return Cursor{
		Type:  typ,
		Limit: limit,
		Value: values,
	}, nil
}

func (r *Paginator) cursorString(v map[string]interface{}) string {
	valuesArr := make([]interface{}, len(r.Columns))
	for i, column := range r.Columns {
		valuesArr[i] = v[column.Name]
	}

	data, err := msgpack.Marshal(valuesArr)
	if err != nil {
		panic(err)
	}

	return base64.StdEncoding.EncodeToString(data)
}
