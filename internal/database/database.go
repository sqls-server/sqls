package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

var (
	ErrNotImplementation error = errors.New("not implementation")
)

type Database interface {
	Open() error
	Close() error
	Databases() ([]string, error)
	Tables() ([]string, error)
	DescribeTable(tableName string) ([]*ColumnDesc, error)
	Exec(ctx context.Context, query string) (sql.Result, error)
	Query(ctx context.Context, query string) (*sql.Rows, error)
	SwitchDB(dbName string) error
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

type Opener func(string, string) Database

var drivers = make(map[string]Opener)

func Register(name string, f Opener) {
	drivers[name] = f
}

func Open(driver string, dataSourceName, dbName string) (Database, error) {
	d, ok := drivers[driver]
	if !ok {
		return nil, fmt.Errorf("driver not found: %v", driver)
	}
	return d(dataSourceName, dbName), nil
}

type Proto string

const (
	ProtoTCP  Proto = "tcp"
	ProtoUDP  Proto = "udp"
	ProtoUnix Proto = "unix"
)

type Config struct {
	Driver         string            `json:"driver" yaml:"driver"`
	DataSourceName string            `json:"dataSourceName" yaml:"dataSourceName"`
	Proto          Proto             `json:"proto" yaml:"proto"`
	User           string            `json:"user" yaml:"user"`
	Passwd         string            `json:"passwd" yaml:"passwd"`
	Host           string            `json:"host" yaml:"host"`
	Port           int               `json:"port" yaml:"port"`
	Path           string            `json:"path" yaml:"path"`
	DBName         string            `json:"dbName" yaml:"dbName"`
	Params         map[string]string `json:"params" yaml:"params"`
}
