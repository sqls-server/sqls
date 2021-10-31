package database

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/url"
	"strconv"

	_ "github.com/denisenkom/go-mssqldb"
	"github.com/lighttiger2505/sqls/dialect"
	"golang.org/x/crypto/ssh"
)

func init() {
	RegisterOpen("mssql", mssqlOpen)
	RegisterFactory("mssql", NewMssqlDBRepository)
}

func mssqlOpen(dbConnCfg *DBConfig) (*DBConnection, error) {
	var (
		conn    *sql.DB
		sshConn *ssh.Client
	)
	dsn, err := genMssqlConfig(dbConnCfg)
	if err != nil {
		return nil, err
	}

	if dbConnCfg.SSHCfg != nil {
		return nil, fmt.Errorf("connect via SSH is not supported")
	}
	dbConn, err := sql.Open("sqlserver", dsn)
	if err != nil {
		return nil, err
	}
	conn = dbConn
	if err = conn.Ping(); err != nil {
		return nil, err
	}

	conn.SetMaxIdleConns(DefaultMaxIdleConns)
	conn.SetMaxOpenConns(DefaultMaxOpenConns)

	return &DBConnection{
		Conn:    conn,
		SSHConn: sshConn,
	}, nil
}

type MssqlDBRepository struct {
	Conn *sql.DB
}

func NewMssqlDBRepository(conn *sql.DB) DBRepository {
	return &MssqlDBRepository{Conn: conn}
}

func (db *MssqlDBRepository) Driver() dialect.DatabaseDriver {
	return dialect.DatabaseDriverMssql
}

func (db *MssqlDBRepository) CurrentDatabase(ctx context.Context) (string, error) {
	row := db.Conn.QueryRowContext(ctx, "SELECT DB_NAME()")
	var database string
	if err := row.Scan(&database); err != nil {
		return "", err
	}
	return database, nil
}

func (db *MssqlDBRepository) Databases(ctx context.Context) ([]string, error) {
	rows, err := db.Conn.QueryContext(
		ctx,
		`
	SELECT name FROM sys.databases
	`)
	if err != nil {
		log.Fatal(err)
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

func (db *MssqlDBRepository) CurrentSchema(ctx context.Context) (string, error) {
	row := db.Conn.QueryRowContext(ctx, "SELECT SCHEMA_NAME()")
	var database sql.NullString
	if err := row.Scan(&database); err != nil {
		return "", err
	}
	if database.Valid {
		return database.String, nil
	}
	return "", nil
}

func (db *MssqlDBRepository) Schemas(ctx context.Context) ([]string, error) {
	rows, err := db.Conn.QueryContext(
		ctx,
		`
	SELECT schema_name FROM information_schema.schemata
	`)
	if err != nil {
		log.Fatal(err)
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

func (db *MssqlDBRepository) SchemaTables(ctx context.Context) (map[string][]string, error) {
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

func (db *MssqlDBRepository) Tables(ctx context.Context) ([]string, error) {
	rows, err := db.Conn.QueryContext(
		ctx,
		`
	SELECT
	  table_name
	FROM
	  information_schema.tables
	WHERE
	  table_type = 'BASE TABLE'
	  AND table_schema NOT IN ('information_schema')
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

func (db *MssqlDBRepository) DescribeDatabaseTable(ctx context.Context) ([]*ColumnDesc, error) {
	rows, err := db.Conn.QueryContext(
		ctx,
		`
	SELECT
		c.table_schema,
		c.table_name,
		c.column_name,
		c.data_type,
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
		information_schema.constraint_column_usage ccu
		ON c.table_name = ccu.table_name
		AND c.column_name = ccu.column_name
	LEFT JOIN information_schema.table_constraints tc ON
		tc.table_catalog = c.table_catalog
		AND tc.table_schema = c.table_schema
		AND tc.table_name = c.table_name
		AND tc.constraint_name = ccu.constraint_name
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

func (db *MssqlDBRepository) DescribeDatabaseTableBySchema(ctx context.Context, schemaName string) ([]*ColumnDesc, error) {
	rows, err := db.Conn.QueryContext(
		ctx,
		`
	SELECT
		c.table_schema,
		c.table_name,
		c.column_name,
		c.data_type,
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
		information_schema.constraint_column_usage ccu
		ON c.table_name = ccu.table_name
		AND c.column_name = ccu.column_name
	LEFT JOIN information_schema.table_constraints tc ON
		tc.table_catalog = c.table_catalog
		AND tc.table_schema = c.table_schema
		AND tc.table_name = c.table_name
		AND tc.constraint_name = ccu.constraint_name
	WHERE
		c.table_schema = $1
	ORDER BY
		c.table_name,
		c.ordinal_position
	`, schemaName)
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

func (db *MssqlDBRepository) Exec(ctx context.Context, query string) (sql.Result, error) {
	return db.Conn.ExecContext(ctx, query)
}

func (db *MssqlDBRepository) Query(ctx context.Context, query string) (*sql.Rows, error) {
	return db.Conn.QueryContext(ctx, query)
}

func genMssqlConfig(connCfg *DBConfig) (string, error) {
	if connCfg.DataSourceName != "" {
		return connCfg.DataSourceName, nil
	}

	q := url.Values{}
	q.Set("user", connCfg.User)
	q.Set("password", connCfg.Passwd)
	q.Set("database", connCfg.DBName)

	switch connCfg.Proto {
	case ProtoTCP:
		host, port := connCfg.Host, connCfg.Port
		if host == "" {
			host = "127.0.0.1"
		}
		if port == 0 {
			port = 1433
		}
		q.Set("server", host)
		q.Set("port", strconv.Itoa(port))
	case ProtoUDP, ProtoUnix:
	default:
		return "", fmt.Errorf("default addr for network %s unknown", connCfg.Proto)
	}

	for k, v := range connCfg.Params {
		q.Set(k, v)
	}

	return genOptions(q, "", "=", ";", ",", true), nil
}
