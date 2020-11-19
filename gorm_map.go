package go_paginate

import (
	"database/sql"
	"gorm.io/gorm"
)

func TakeMap(tx *gorm.DB) (map[string]interface{}, error) {
	rows, err := tx.Rows()
	if err != nil {
		return nil, err
	}

	if !rows.Next() {
		return nil, nil
	}

	m, err := rowMap(rows)
	if err != nil {
		return nil, err
	}

	return m, err
}

func FindMap(tx *gorm.DB) ([]map[string]interface{}, error) {
	rows, err := tx.Rows()
	if err != nil {
		return nil, err
	}

	return rowsMap(rows)
}

func rowsMap(rows *sql.Rows) ([]map[string]interface{}, error) {
	r := make([]map[string]interface{}, 0)

	for rows.Next() {
		m, err := rowMap(rows)
		if err != nil {
			return nil, err
		}

		r = append(r, m)
	}

	return r, nil
}

func rowMap(rows *sql.Rows) (map[string]interface{}, error) {
	cols, _ := rows.Columns()

	columns := make([]interface{}, len(cols))
	columnPointers := make([]interface{}, len(cols))
	for i := range columns {
		columnPointers[i] = &columns[i]
	}

	// Scan the result into the column pointers...
	if err := rows.Scan(columnPointers...); err != nil {
		return nil, err
	}

	// Create our map, and retrieve the value for each column from the pointers slice,
	// storing it in the map with the name of the column as the key.
	m := make(map[string]interface{})
	for i, colName := range cols {
		val := columnPointers[i].(*interface{})
		m[colName] = *val
	}

	return m, nil
}
