package gorm

import (
	"github.com/raphaelvigee/go-paginate/driver/sql"
	"gorm.io/gorm"
)

func TakeMap(tx *gorm.DB) (map[string]interface{}, error) {
	rows, err := tx.Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	if !rows.Next() {
		return nil, nil
	}

	m, err := sql.RowMap(rows)
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
	defer rows.Close()

	return sql.RowsMap(rows)
}
