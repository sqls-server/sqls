package database

import (
	"context"
	"database/sql"
	"log"

	_ "github.com/lib/pq"
)

type PostgreSQLDB struct {
	DataSourceName string
	Option         *DBOption
	Conn           *sql.DB
}

func init() {
	Register("postgresql", func(dataSourceName, dbName string) Database {
		return &PostgreSQLDB{
			DataSourceName: dataSourceName,
			Option:         &DBOption{},
		}
	})
}

func (db *PostgreSQLDB) Open() error {
	conn, err := sql.Open("postgres", db.DataSourceName)
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

func (db *PostgreSQLDB) Close() error {
	return db.Conn.Close()
}

func (db *PostgreSQLDB) Databases() ([]string, error) {
	rows, err := db.Conn.Query(`
	SELECT
	  schema_name 
	FROM
	  information_schema.schemata
	WHERE
	  schema_name NOT IN ('pg_catalog', 'information_schema') 
	`)
	if err != nil {
		log.Fatal(err)
	}
	databases := []string{}
	for rows.Next() {
		var database string
		if err := rows.Scan(&database); err != nil {
			return nil, err
		}
		databases = append(databases, database)
	}
	return databases, nil
}

func (db *PostgreSQLDB) Tables() ([]string, error) {
	rows, err := db.Conn.Query(`
	SELECT
	  table_name 
	FROM
	  information_schema.tables 
	WHERE
	  table_type = 'BASE TABLE' 
	  AND table_schema NOT IN ('pg_catalog', 'information_schema') 
	ORDER BY
	  table_name
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

func (db *PostgreSQLDB) DescribeTable(tableName string) ([]*ColumnDesc, error) {
	rows, err := db.Conn.Query(`
	SELECT
	  c.column_name
	  , c.data_type
	  , c.is_nullable
	  , CASE tc.constraint_type 
		WHEN 'PRIMARY KEY' THEN 'YES' 
		ELSE 'NO' 
		END
	  , c.column_default
	  , '' 
	FROM
	  information_schema.columns c 
	  LEFT JOIN information_schema.constraint_column_usage ccu 
		ON c.table_name = ccu.table_name 
		AND c.column_name = ccu.column_name 
	  LEFT JOIN information_schema.table_constraints tc 
		ON tc.table_catalog = c.table_catalog 
		AND tc.table_schema = c.table_schema 
		AND tc.table_name = c.table_name 
		AND tc.constraint_name = ccu.constraint_name 
	WHERE
	  c.table_name = $1
	ORDER BY
	  c.table_name
	  , c.ordinal_position
	`, tableName)
	if err != nil {
		log.Fatal(err)
	}
	tableInfos := []*ColumnDesc{}
	for rows.Next() {
		var tableInfo ColumnDesc
		err := rows.Scan(
			&tableInfo.Name,
			&tableInfo.Type,
			&tableInfo.Null,
			&tableInfo.Key,
			&tableInfo.Default,
			&tableInfo.Extra,
		)
		if err != nil {
			return nil, err
		}
		tableInfos = append(tableInfos, &tableInfo)
	}
	return tableInfos, nil
}

func (db *PostgreSQLDB) Exec(ctx context.Context, query string) (sql.Result, error) {
	return db.Conn.ExecContext(ctx, query)
}

func (db *PostgreSQLDB) Query(ctx context.Context, query string) (*sql.Rows, error) {
	return db.Conn.QueryContext(ctx, query)
}

func (db *PostgreSQLDB) SwitchDB(dbName string) error {
	return ErrNotImplementation
}
