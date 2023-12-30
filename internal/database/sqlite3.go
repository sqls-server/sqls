package database

import (
	"context"
	"database/sql"
	"fmt"
	"log"

	_ "github.com/mattn/go-sqlite3"
	"github.com/sqls-server/sqls/dialect"
)

func init() {
	RegisterOpen(dialect.DatabaseDriverSQLite3, sqlite3Open)
	RegisterFactory(dialect.DatabaseDriverSQLite3, NewSQLite3DBRepository)
}

func sqlite3Open(connCfg *DBConfig) (*DBConnection, error) {
	conn, err := sql.Open("sqlite3", connCfg.DataSourceName)
	if err != nil {
		return nil, err
	}
	conn.SetMaxIdleConns(DefaultMaxIdleConns)
	conn.SetMaxOpenConns(DefaultMaxOpenConns)
	return &DBConnection{
		Conn: conn,
	}, nil
}

type SQLite3DBRepository struct {
	Conn *sql.DB
}

func NewSQLite3DBRepository(conn *sql.DB) DBRepository {
	return &SQLite3DBRepository{Conn: conn}
}

func (db *SQLite3DBRepository) Driver() dialect.DatabaseDriver {
	return dialect.DatabaseDriverSQLite3
}

func (db *SQLite3DBRepository) CurrentDatabase(ctx context.Context) (string, error) {
	return "", nil
}

func (db *SQLite3DBRepository) Databases(ctx context.Context) ([]string, error) {
	return []string{}, nil
}

func (db *SQLite3DBRepository) CurrentSchema(ctx context.Context) (string, error) {
	return db.CurrentDatabase(ctx)
}

func (db *SQLite3DBRepository) Schemas(ctx context.Context) ([]string, error) {
	return db.Databases(ctx)
}

func (db *SQLite3DBRepository) SchemaTables(ctx context.Context) (map[string][]string, error) {
	tables, err := db.Tables(ctx)
	if err != nil {
		return nil, err
	}
	return map[string][]string{"": tables}, nil
}

func (db *SQLite3DBRepository) Tables(ctx context.Context) ([]string, error) {
	rows, err := db.Conn.QueryContext(ctx, `
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
	defer rows.Close()
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

func (db *SQLite3DBRepository) describeTable(ctx context.Context, tableName string) ([]*ColumnDesc, error) {
	rows, err := db.Conn.QueryContext(ctx, fmt.Sprintf("PRAGMA table_info(%s);", tableName))
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()
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
		tableInfo.Table = tableName
		if nonnull != 0 {
			tableInfo.Null = "NO"
		} else {
			tableInfo.Null = "YES"
		}
		tableInfos = append(tableInfos, &tableInfo)
	}
	return tableInfos, nil
}

func (db *SQLite3DBRepository) DescribeDatabaseTable(ctx context.Context) ([]*ColumnDesc, error) {
	tables, err := db.Tables(ctx)
	if err != nil {
		return nil, err
	}
	all := []*ColumnDesc{}
	for _, table := range tables {
		descs, err := db.describeTable(ctx, table)
		if err != nil {
			return nil, err
		}
		all = append(all, descs...)
	}
	return all, nil
}

func (db *SQLite3DBRepository) DescribeDatabaseTableBySchema(ctx context.Context, _ string) ([]*ColumnDesc, error) {
	return db.DescribeDatabaseTable(ctx)
}

func (db *SQLite3DBRepository) DescribeForeignKeysBySchema(ctx context.Context, schemaName string) ([]*ForeignKey, error) {
	rows, err := db.Conn.QueryContext(
		ctx,
		`
	SELECT m.name || p."id",
       m.name,
       p."from",
       p."table",
       p."to"
	FROM sqlite_master m
			 JOIN pragma_foreign_key_list(m.name) p ON m.name != p."table"
	WHERE m.type = 'table'
	ORDER BY 1, p."seq"
		`)
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = rows.Close() }()
	return parseForeignKeys(rows, schemaName)
}

func (db *SQLite3DBRepository) Exec(ctx context.Context, query string) (sql.Result, error) {
	return db.Conn.ExecContext(ctx, query)
}

func (db *SQLite3DBRepository) Query(ctx context.Context, query string) (*sql.Rows, error) {
	return db.Conn.QueryContext(ctx, query)
}
