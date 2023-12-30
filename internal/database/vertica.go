package database

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/sqls-server/sqls/dialect"
	_ "github.com/vertica/vertica-sql-go"
	"log"
	"strconv"
)

func init() {
	RegisterOpen("vertica", verticaOpen)
	RegisterFactory("vertica", NewVerticaDBRepository)
}

func verticaOpen(dbConnCfg *DBConfig) (*DBConnection, error) {
	var (
		conn *sql.DB
	)
	DSName, err := genVerticaConfig(dbConnCfg)
	if err != nil {
		return nil, err
	}

	conn, err = sql.Open("vertica", DSName)
	if err != nil {
		return nil, err
	}

	conn.SetMaxIdleConns(DefaultMaxIdleConns)
	conn.SetMaxOpenConns(DefaultMaxOpenConns)

	return &DBConnection{
		Conn:   conn,
		Driver: dialect.DatabaseDriverVertica,
	}, nil
}

func genVerticaConfig(connCfg *DBConfig) (string, error) {
	if connCfg.DataSourceName != "" {
		return connCfg.DataSourceName, nil
	}

	host, port := connCfg.Host, connCfg.Port
	if host == "" {
		host = "127.0.0.1"
	}
	if port == 0 {
		port = 5433
	}

	DSName := connCfg.User + "/" + connCfg.Passwd + "@" + host + ":" + strconv.Itoa(port) + "/" + connCfg.DBName
	return DSName, nil
}

type VerticaDBRepository struct {
	Conn *sql.DB
}

func NewVerticaDBRepository(conn *sql.DB) DBRepository {
	return &VerticaDBRepository{Conn: conn}
}

func (db *VerticaDBRepository) Driver() dialect.DatabaseDriver {
	return dialect.DatabaseDriverVertica
}

func (db *VerticaDBRepository) CurrentDatabase(ctx context.Context) (string, error) {
	row := db.Conn.QueryRowContext(ctx, "SELECT CURRENT_SCHEMA()")
	var database string
	if err := row.Scan(&database); err != nil {
		return "", err
	}
	return database, nil
}

func (db *VerticaDBRepository) Databases(ctx context.Context) ([]string, error) {
	rows, err := db.Conn.QueryContext(ctx, "SELECT schema_name FROM v_catalog.schemata")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
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

func (db *VerticaDBRepository) CurrentSchema(ctx context.Context) (string, error) {
	return db.CurrentDatabase(ctx)
}

func (db *VerticaDBRepository) Schemas(ctx context.Context) ([]string, error) {
	return db.Databases(ctx)
}

func (db *VerticaDBRepository) SchemaTables(ctx context.Context) (map[string][]string, error) {
	rows, err := db.Conn.QueryContext(
		ctx,
		`
    SELECT schema_name, TABLE_NAME
      FROM v_catalog.all_tables
     ORDER BY schema_name, TABLE_NAME
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

func (db *VerticaDBRepository) Tables(ctx context.Context) ([]string, error) {
	rows, err := db.Conn.QueryContext(ctx, "SELECT table_name FROM v_catalog.tables ORDER BY 1")
	if err != nil {
		return nil, err
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

func (db *VerticaDBRepository) DescribeDatabaseTable(ctx context.Context) ([]*ColumnDesc, error) {
	rows, err := db.Conn.QueryContext(
		ctx,
		`
SELECT table_schema,
       table_name,
       column_name,
       data_type,
       is_nullable,
       '',
       column_default,
       ''
  FROM v_catalog.columns
`)
	if err != nil {
		return nil, err
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

func (db *VerticaDBRepository) DescribeDatabaseTableBySchema(ctx context.Context, schemaName string) ([]*ColumnDesc, error) {
	rows, err := db.Conn.QueryContext(
		ctx,
		`
        SELECT table_schema,
               table_name,
               column_name,
               data_type,
               CASE is_nullable
               WHEN true THEN 'YES'
               ELSE 'NO'
               END AS is_nullable,
               '1' AS COLUMN_KEY,
               column_default,
               '1' AS EXTRA
          FROM v_catalog.columns
         WHERE table_schema = ?
`, schemaName)
	if err != nil {
		log.Println("schema", schemaName, err.Error())
		return nil, err
	}
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
	defer rows.Close()
	return tableInfos, nil
}

func (db *VerticaDBRepository) Exec(ctx context.Context, query string) (sql.Result, error) {
	return db.Conn.ExecContext(ctx, query)
}

func (db *VerticaDBRepository) Query(ctx context.Context, query string) (*sql.Rows, error) {
	return db.Conn.QueryContext(ctx, query)
}

func (db *VerticaDBRepository) DescribeForeignKeysBySchema(ctx context.Context, schemaName string) ([]*ForeignKey, error) {
	return nil, fmt.Errorf("describe foreign keys is not supported")
}
