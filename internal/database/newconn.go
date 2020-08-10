package database

import (
	"context"
	"database/sql"
	"fmt"

	"golang.org/x/crypto/ssh"
	"golang.org/x/xerrors"
)

var driverOpeners = make(map[string]ConnOpener)

type ConnOpener func(*Config) (*DBConn, error)

func init() {
	RegisterConn("mysql", mysqlConn)
}

type DBConn struct {
	Conn    *sql.DB
	SSHConn *ssh.Client
}

func (db *DBConn) Close() error {
	if db == nil {
		return nil
	}
	if err := db.Conn.Close(); err != nil {
		return err
	}
	if db.SSHConn != nil {
		if err := db.SSHConn.Close(); err != nil {
			return err
		}
	}
	return nil
}

func RegisterConn(name string, opener ConnOpener) {
	if _, ok := driverOpeners[name]; ok {
		panic(fmt.Sprintf("driver %s is already registered", name))
	}
	driverOpeners[name] = opener
}

func RegisteredConn(name string) bool {
	_, ok := drivers[name]
	return ok
}

func OpenConn(cfg *Config) (*DBConn, error) {
	d, ok := driverOpeners[cfg.Driver]
	if !ok {
		return nil, xerrors.Errorf("driver not found, %v", cfg.Driver)
	}
	return d(cfg)
}

func mysqlConn(connCfg *Config) (*DBConn, error) {
	var (
		conn    *sql.DB
		sshConn *ssh.Client
	)
	cfg, err := genMysqlConfig(connCfg)
	if err != nil {
		return nil, err
	}

	if connCfg.SSHCfg != nil {
		dbConn, dbSSHConn, err := openMySQLViaSSH(cfg.FormatDSN(), connCfg.SSHCfg)
		if err != nil {
			return nil, err
		}
		conn = dbConn
		sshConn = dbSSHConn
	} else {
		dbConn, err := sql.Open("mysql", cfg.FormatDSN())
		if err != nil {
			return nil, err
		}
		conn = dbConn
	}
	if err := conn.Ping(); err != nil {
		return nil, err
	}

	return &DBConn{
		Conn:    conn,
		SSHConn: sshConn,
	}, nil
}

type DBRepository interface {
	CurrentDatabase(ctx context.Context) (string, error)
	Databases(ctx context.Context) ([]string, error)
	CurrentSchema(ctx context.Context) (string, error)
	Schemas(ctx context.Context) ([]string, error)
	SchemaTables(ctx context.Context) (map[string][]string, error)
	DescribeDatabaseTable(ctx context.Context) ([]*ColumnDesc, error)
	DescribeDatabaseTableBySchema(ctx context.Context, schemaName string) ([]*ColumnDesc, error)
	Exec(ctx context.Context, query string) (sql.Result, error)
	Query(ctx context.Context, query string) (*sql.Rows, error)
}

type MySQLDBRepository struct {
	Conn *sql.DB
}

func NewMySQLDBRepository(conn *sql.DB) DBRepository {
	return &MySQLDBRepository{Conn: conn}
}

func (db *MySQLDBRepository) CurrentDatabase(ctx context.Context) (string, error) {
	row := db.Conn.QueryRowContext(ctx, "SELECT DATABASE()")
	var database string
	if err := row.Scan(&database); err != nil {
		return "", err
	}
	return database, nil
}

func (db *MySQLDBRepository) Databases(ctx context.Context) ([]string, error) {
	rows, err := db.Conn.QueryContext(ctx, "select SCHEMA_NAME from information_schema.SCHEMATA")
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

func (db *MySQLDBRepository) CurrentSchema(ctx context.Context) (string, error) {
	return db.CurrentDatabase(ctx)
}

func (db *MySQLDBRepository) Schemas(ctx context.Context) ([]string, error) {
	return db.Databases(ctx)
}

func (db *MySQLDBRepository) SchemaTables(ctx context.Context) (map[string][]string, error) {
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

func (db *MySQLDBRepository) Tables(ctx context.Context) ([]string, error) {
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

func (db *MySQLDBRepository) DescribeDatabaseTable(ctx context.Context) ([]*ColumnDesc, error) {
	rows, err := db.Conn.QueryContext(
		ctx,
		`
SELECT
	TABLE_SCHEMA,
	TABLE_NAME,
	COLUMN_NAME,
	COLUMN_TYPE,
	IS_NULLABLE,
	COLUMN_KEY,
	COLUMN_DEFAULT,
	EXTRA
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

func (db *MySQLDBRepository) DescribeDatabaseTableBySchema(ctx context.Context, schemaName string) ([]*ColumnDesc, error) {
	rows, err := db.Conn.QueryContext(
		ctx,
		`
SELECT
	TABLE_SCHEMA,
	TABLE_NAME,
	COLUMN_NAME,
	COLUMN_TYPE,
	IS_NULLABLE,
	COLUMN_KEY,
	COLUMN_DEFAULT,
	EXTRA
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

func (db *MySQLDBRepository) Exec(ctx context.Context, query string) (sql.Result, error) {
	return db.Conn.ExecContext(ctx, query)
}

func (db *MySQLDBRepository) Query(ctx context.Context, query string) (*sql.Rows, error) {
	return db.Conn.QueryContext(ctx, query)
}
