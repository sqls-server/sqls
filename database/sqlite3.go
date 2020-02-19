package database

import (
	"context"
	"database/sql"
	"fmt"
	"log"

	_ "github.com/mattn/go-sqlite3"
)

type SQLite3DB struct {
	ConnString string
	Option     *DBOption
	Conn       *sql.DB
}

func init() {
	Register("sqlite3", func(connString string) Database {
		return &SQLite3DB{
			ConnString: connString,
			Option:     &DBOption{},
		}
	})
}

func (db *SQLite3DB) Open() error {
	conn, err := sql.Open("sqlite3", db.ConnString)
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

func (db *SQLite3DB) Databases() ([]string, error) {
	return []string{}, nil
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

func (db *SQLite3DB) ExecuteQuery(ctx context.Context, query string) (interface{}, error) {
	return db.Conn.ExecContext(ctx, query)
}
