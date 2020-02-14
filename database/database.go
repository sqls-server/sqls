package database

import (
	"database/sql"
	"errors"
)

type Database interface {
	Open() error
	Close() error
	Databases() ([]string, error)
	Tables() ([]string, error)
	DescribeTable(tableName string) ([]*ColumnDesc, error)
}

const (
	DefaultMaxIdleConns = 10
	DefaultMaxOpenConns = 5
)

type DBOption struct {
	MaxIdleConns int
	MaxOpenConns int
}

type ColumnDesc struct {
	Name    string
	Type    string
	Null    string
	Key     string
	Default sql.NullString
	Extra   string
}

type Opener func(string) Database

var drivers = make(map[string]Opener)

func Register(name string, f Opener) {
	drivers[name] = f
}

func Open(driver string, dataSourceName string) (Database, error) {
	d, ok := drivers[driver]
	if !ok {
		return nil, errors.New("driver not found")
	}
	return d(dataSourceName), nil
}
