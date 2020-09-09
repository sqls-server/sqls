package database

import (
	"context"
	"database/sql"
	"fmt"

	snowflake "github.com/snowflakedb/gosnowflake"
	"golang.org/x/crypto/ssh"
)

func init() {
	RegisterOpen("snowflake", snowflakeOpen)
	RegisterFactory("snowflake", NewSnowFlakeDBRepository)
}

func snowflakeOpen(dbConnCfg *DBConfig) (*DBConnection, error) {
	var (
		conn    *sql.DB
		sshConn *ssh.Client
	)

	cfg, err := genSnowFlakeConfig(dbConnCfg)
	if err != nil {
		return nil, err
	}

	dsn, _ := snowflake.DSN(cfg)

	if dbConnCfg.SSHCfg != nil {
		return nil, fmt.Errorf("currently not supporting ssh connection")
	} else {
		dbConn, err := sql.Open("snowflake", dsn)
		if err != nil {
			return nil, err
		}
		conn = dbConn
	}
	if err := conn.Ping(); err != nil {
		return nil, err
	}

	return &DBConnection{
		Conn:    conn,
		SSHConn: sshConn,
	}, nil
}

func genSnowFlakeConfig(connCfg *DBConfig) (*snowflake.Config, error) {
	if connCfg.DataSourceName != "" {
		return snowflake.ParseDSN(connCfg.DataSourceName)
	}

	cfg := &snowflake.Config{
		Account: connCfg.Params["account"],
		User: connCfg.User,
		Password: connCfg.Passwd,
		Database: connCfg.DBName,
		Schema: connCfg.Params["schema"],
	}
	return cfg, nil
}

type SnowFlakeDBRepository struct {
	Conn *sql.DB
}

func NewSnowFlakeDBRepository(conn *sql.DB) DBRepository {
	return &SnowFlakeDBRepository{Conn: conn}
}

func (db *SnowFlakeDBRepository) CurrentDatabase(ctx context.Context) (string, error) {
	row := db.Conn.QueryRowContext(ctx, "SELECT CURRENT_DATABASE()")
	var database string
	if err := row.Scan(&database); err != nil {
		return "", err
	}
	return database, nil
}

func (db *SnowFlakeDBRepository) Databases(ctx context.Context) ([]string, error) {
	rows, err := db.Conn.QueryContext(ctx, "select DATABASE_NAME from information_schema.databases")
	if err != nil {
		return nil, err
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

func (db *SnowFlakeDBRepository) CurrentSchema(ctx context.Context) (string, error) {
	row := db.Conn.QueryRowContext(ctx, "SELECT COALESCE(CURRENT_SCHEMA(), '')")
	var schema string
	if err := row.Scan(&schema); err != nil {
		return "", err
	}
	return schema, nil
}

func (db *SnowFlakeDBRepository) Schemas(ctx context.Context) ([]string, error) {
	schemas := []string{}
	rows, err := db.Conn.QueryContext(ctx, "select SCHEMA_NAME from information_schema.schemata")
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var schema string
		if err := rows.Scan(&schema); err != nil {
			return nil, err
		}
		schemas = append(schemas, schema)
	}
	return schemas, nil
}

func (db *SnowFlakeDBRepository) Exec(ctx context.Context, query string) (sql.Result, error) {
	return db.Conn.ExecContext(ctx, query)
}

func (db *SnowFlakeDBRepository) Query(ctx context.Context, query string) (*sql.Rows, error) {
	return db.Conn.QueryContext(ctx, query)
}


func (db *SnowFlakeDBRepository) SchemaTables(ctx context.Context) (map[string][]string, error) {
	rows, err := db.Conn.QueryContext(
		ctx,
		`
	SELECT
		TABLE_SCHEMA,
		TABLE_NAME
	FROM
		information_schema.TABLES
	ORDER BY
		TABLE_SCHEMA,
		TABLE_NAME
	`)
	if err != nil {
		return nil, err
	}
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

func (db *SnowFlakeDBRepository) Tables(ctx context.Context) ([]string, error) {
	rows, err := db.Conn.QueryContext(ctx, "SHOW TABLES")
	if err != nil {
		return nil, err
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
func (db *SnowFlakeDBRepository) DescribeDatabaseTable(ctx context.Context) ([]*ColumnDesc, error) {
	rows, err := db.Conn.QueryContext(
		ctx,
		`
SELECT
	TABLE_SCHEMA,
	TABLE_NAME,
	COLUMN_NAME,
	DATA_TYPE AS COLUMN_TYPE,
	IS_NULLABLE,
	'' AS COLUMN_KEY,
	COLUMN_DEFAULT,
	'' AS EXTRA
FROM information_schema.COLUMNS
`)
	if err != nil {
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
	return tableInfos, nil
}

func (db *SnowFlakeDBRepository) DescribeDatabaseTableBySchema(ctx context.Context, schemaName string) ([]*ColumnDesc, error) {
	rows, err := db.Conn.QueryContext(
		ctx,
		`
SELECT
	TABLE_SCHEMA,
	TABLE_NAME,
	COLUMN_NAME,
	DATA_TYPE AS COLUMN_TYPE,
	IS_NULLABLE,
	'' AS COLUMN_KEY,
	COLUMN_DEFAULT,
	'' AS EXTRA
FROM information_schema.COLUMNS
WHERE information_schema.COLUMNS.TABLE_SCHEMA = ?
`, schemaName)
	if err != nil {
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
	return tableInfos, nil
}
