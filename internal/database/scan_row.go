package database

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"time"
)

func Columns(rows *sql.Rows) ([]string, error) {
	var cols []string
	var err error

	cols, err = rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("cannot get query columns, %w", err)
	}

	for i, c := range cols {
		if strings.TrimSpace(c) == "" {
			cols[i] = fmt.Sprintf("col%d", i)
		}
	}

	return cols, nil
}

func ScanRows(rows *sql.Rows, columnLength int) ([][]string, error) {
	stringRows := [][]string{}
	for rows.Next() {
		// scan to []interface{}
		rowBuffer := make([]interface{}, columnLength)
		for i := range rowBuffer {
			rowBuffer[i] = new(interface{})
		}
		if err := rows.Scan(rowBuffer...); err != nil {
			return nil, err
		}

		stringRow := make([]string, columnLength)
		for i, buf := range rowBuffer {
			val, err := sqlValToString(buf)
			if err != nil {
				return nil, err
			}
			stringRow[i] = val
		}
		stringRows = append(stringRows, stringRow)
	}
	return stringRows, nil
}

func sqlValToString(pointer interface{}) (string, error) {
	res := ""
	if pointer == nil {
		return res, nil
	}

	val := *pointer.(*interface{})

	reflectVal := reflect.ValueOf(val)

	// extract pointer value
	if reflectVal.Kind() == reflect.Pointer {
		val = reflectVal.Elem().Interface()

		if val == nil {
			return "", nil
		}
	}

	switch v := (val).(type) {
	case []byte:
		res = string(v)
	case string:
		res = v
	case time.Time:
		res = v.Format(time.RFC3339Nano)
	case fmt.Stringer:
		res = v.String()
	case map[string]interface{}:
		buf, err := json.Marshal(v)
		if err != nil {
			return "", err
		}
		res = string(buf)
	case []interface{}:
		buf, err := json.Marshal(v)
		if err != nil {
			return "", err
		}
		res = string(buf)
	default:
		res = fmt.Sprintf("%v", v)
	}

	return res, nil
}
