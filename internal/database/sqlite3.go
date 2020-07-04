package database

import (
	"context"
	"database/sql"
	"fmt"
	"log"

	_ "github.com/mattn/go-sqlite3"
)

type SQLite3DB struct {
	Cfg    *Config
	Option *DBOption
	Conn   *sql.DB
}

func init() {
	Register("sqlite3", func(cfg *Config) Database {
		return &SQLite3DB{
			Cfg:    cfg,
			Option: &DBOption{},
		}
	})
}

func (db *SQLite3DB) Open() error {
	conn, err := sql.Open("sqlite3", db.Cfg.DataSourceName)
	if err != nil {
		return err
	}
	conn.SetMaxIdleConns(DefaultMaxIdleConns)
	if db.Option.MaxIdleConns != 0 {
		conn.SetMaxIdleConns(db.Option.MaxIdleConns)
	}
	conn.SetMaxOpenConns(DefaultMaxOpenConns)
	if db.Option.MaxOpenConns != 0 {
		conn.SetMaxOpenConns(db.Option.MaxOpenConns)
	}
	db.Conn = conn
	return nil
}

func (db *SQLite3DB) Close() error {
	return db.Conn.Close()
}

func (db *SQLite3DB) Database() (string, error) {
	return "", ErrNotImplementation
}

func (db *SQLite3DB) Databases() ([]string, error) {
	return nil, ErrNotImplementation
}

func (db *SQLite3DB) Schema() (string, error) {
	return "", ErrNotImplementation
}

func (db *SQLite3DB) Schemas() ([]string, error) {
	return nil, ErrNotImplementation
}

func (db *SQLite3DB) DatabaseTables() (map[string][]string, error) {
	return nil, nil
}

func (db *SQLite3DB) Tables() ([]string, error) {
	rows, err := db.Conn.Query(`
	SELECT
	  name 
	FROM
	  sqlite_master
	WHERE
	  type = 'table' 
	ORDER BY
	  name
	`)
	if err != nil {
		log.Fatal(err)
	}
	tables := []string{}
	for rows.Next() {
		var table string
		if err := rows.Scan(&table); err != nil {
			return nil, err
		}
		tables = append(tables, table)
	}
	return tables, nil
}

func (db *SQLite3DB) DescribeTable(tableName string) ([]*ColumnDesc, error) {
	rows, err := db.Conn.Query(fmt.Sprintf(`
	PRAGMA table_info(%s);
	`, tableName))
	if err != nil {
		log.Fatal(err)
	}
	tableInfos := []*ColumnDesc{}
	for rows.Next() {
		var id int
		var nonnull int
		var tableInfo ColumnDesc
		err := rows.Scan(
			&id,
			&tableInfo.Name,
			&tableInfo.Type,
			&nonnull,
			&tableInfo.Default,
			&tableInfo.Key,
		)
		if err != nil {
			return nil, err
		}
		if nonnull != 0 {
			tableInfo.Null = "NO"
		} else {
			tableInfo.Null = "YES"
		}
		tableInfos = append(tableInfos, &tableInfo)
	}
	return tableInfos, nil
}

func (db *SQLite3DB) DescribeDatabaseTable() ([]*ColumnDesc, error) {
	return nil, ErrNotImplementation
}

func (db *SQLite3DB) Exec(ctx context.Context, query string) (sql.Result, error) {
	return db.Conn.ExecContext(ctx, query)
}

func (db *SQLite3DB) Query(ctx context.Context, query string) (*sql.Rows, error) {
	return db.Conn.QueryContext(ctx, query)
}

func (db *SQLite3DB) SwitchDB(dbName string) error {
	return ErrNotImplementation
}
