package database

import (
	"context"
	"database/sql"
	"fmt"
	"log"

	_ "github.com/CodinGame/h2go"
	"github.com/sqls-server/sqls/dialect"
)

func init() {
	RegisterOpen("h2", h2Open)
	RegisterFactory("h2", NewH2DBRepository)
}

func h2Open(dbConnCfg *DBConfig) (*DBConnection, error) {
	var (
		conn *sql.DB
	)
	cfg, err := genH2Config(dbConnCfg)
	if err != nil {
		return nil, err
	}

	if dbConnCfg.SSHCfg != nil {
		return nil, fmt.Errorf("connect via SSH is not supported")
	}
	dbConn, err := sql.Open("h2", cfg)
	if err != nil {
		return nil, err
	}
	conn = dbConn

	return &DBConnection{
		Conn:   conn,
		Driver: dbConnCfg.Driver,
	}, nil
}

func genH2Config(connCfg *DBConfig) (string, error) {
	if connCfg.DataSourceName != "" {
		return connCfg.DataSourceName, nil
	}

	return "", fmt.Errorf("only DataSourceName is supported")
}

type H2DBRepository struct {
	Conn   *sql.DB
	driver dialect.DatabaseDriver
}

func NewH2DBRepository(conn *sql.DB) DBRepository {
	return &H2DBRepository{Conn: conn}
}

func (db *H2DBRepository) Driver() dialect.DatabaseDriver {
	return db.driver
}

func (db *H2DBRepository) CurrentDatabase(ctx context.Context) (string, error) {
	return "", nil
}

func (db *H2DBRepository) Databases(ctx context.Context) ([]string, error) {
	return []string{}, nil
}

func (db *H2DBRepository) CurrentSchema(ctx context.Context) (string, error) {
	return "PUBLIC", nil
}

func (db *H2DBRepository) Schemas(ctx context.Context) ([]string, error) {
	rows, err := db.Conn.QueryContext(
		ctx,
		`
	SELECT schema_name FROM information_schema.schemata
	`)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()
	schemas := []string{}
	for rows.Next() {
		var schema string
		if err := rows.Scan(&schema); err != nil {
			return nil, err
		}
		schemas = append(schemas, schema)
	}
	return schemas, nil
}

func (db *H2DBRepository) SchemaTables(ctx context.Context) (map[string][]string, error) {
	rows, err := db.Conn.QueryContext(
		ctx,
		`
	SELECT
		table_schema,
		table_name
	FROM
		information_schema.tables
	ORDER BY
		table_schema,
		table_name
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	databaseTables := map[string][]string{}
	for rows.Next() {
		var schema, table string
		if err := rows.Scan(&schema, &table); err != nil {
			return nil, err
		}

		if arr, ok := databaseTables[schema]; ok {
			databaseTables[schema] = append(arr, table)
		} else {
			databaseTables[schema] = []string{table}
		}
	}
	return databaseTables, nil
}

func (db *H2DBRepository) Tables(ctx context.Context) ([]string, error) {

	rows, err := db.Conn.QueryContext(
		ctx,
		`
	SELECT
		table_name 
	FROM
		information_schema.tables 
	WHERE
		table_schema NOT IN ('INFORMATION_SCHEMA') 
	ORDER BY
		table_name
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

func (db *H2DBRepository) DescribeDatabaseTable(ctx context.Context) ([]*ColumnDesc, error) {
	rows, err := db.Conn.QueryContext(
		ctx,
		`
	SELECT
		c.table_schema,
		c.table_name,
		c.column_name,
		c.type_name,
		c.is_nullable,
		CASE tc.constraint_type
			WHEN 'PRIMARY KEY' THEN 'YES'
			ELSE 'NO'
		END,
		c.column_default,
		''
	FROM
		information_schema.columns c
	LEFT JOIN
		information_schema.constraints tc
		ON c.table_schema = tc.table_schema
		AND c.table_name = tc.table_name
		AND REGEXP_LIKE(tc.column_list, '(^|,)' || c.column_name || '(,|$)', 'i')
	ORDER BY
		c.table_name,
		c.ordinal_position
	`)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()
	tableInfos := []*ColumnDesc{}
	for rows.Next() {
		var tableInfo ColumnDesc
		err := rows.Scan(
			&tableInfo.Schema,
			&tableInfo.Table,
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

func (db *H2DBRepository) DescribeDatabaseTableBySchema(ctx context.Context, schemaName string) ([]*ColumnDesc, error) {
	// h2go doesn't support NamedValue yet
	rows, err := db.Conn.QueryContext(
		ctx,
		fmt.Sprintf(`
	SELECT
		c.table_schema,
		c.table_name,
		c.column_name,
		c.type_name,
		c.is_nullable,
		CASE tc.constraint_type
			WHEN 'PRIMARY KEY' THEN 'YES'
			ELSE 'NO'
		END,
		c.column_default,
		''
	FROM
		information_schema.columns c
	LEFT JOIN
		information_schema.constraints tc
		ON c.table_schema = tc.table_schema
		AND c.table_name = tc.table_name
		AND REGEXP_LIKE(tc.column_list, '(^|,)' || c.column_name || '(,|$)', 'i')
	WHERE
		c.table_schema = '%s'
	ORDER BY
		c.table_name,
		c.ordinal_position
	`, schemaName))
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()
	tableInfos := []*ColumnDesc{}
	for rows.Next() {
		var tableInfo ColumnDesc
		err := rows.Scan(
			&tableInfo.Schema,
			&tableInfo.Table,
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

func (db *H2DBRepository) Exec(ctx context.Context, query string) (sql.Result, error) {
	return db.Conn.ExecContext(ctx, query)
}

func (db *H2DBRepository) Query(ctx context.Context, query string) (*sql.Rows, error) {
	return db.Conn.QueryContext(ctx, query)
}

func (db *H2DBRepository) DescribeForeignKeysBySchema(ctx context.Context, schemaName string) ([]*ForeignKey, error) {
	return nil, fmt.Errorf("describe foreign keys is not supported")
}
