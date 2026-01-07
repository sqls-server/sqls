package database

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/url"
	"os"
	"strconv"

	_ "github.com/denisenkom/go-mssqldb"
	"github.com/jfcote87/sshdb"
	"github.com/jfcote87/sshdb/mssql"
	"github.com/sqls-server/sqls/dialect"
	"golang.org/x/crypto/ssh"
)

func init() {
	RegisterOpen("mssql", mssqlOpen)
	RegisterFactory("mssql", NewMssqlDBRepository)
}

func mssqlOpen(dbConnCfg *DBConfig) (*DBConnection, error) {
	var (
		conn *sql.DB
	)
	dsn, err := genMssqlConfig(dbConnCfg)
	if err != nil {
		return nil, err
	}

	if dbConnCfg.SSHCfg != nil {
		key, err := os.ReadFile(dbConnCfg.SSHCfg.PrivateKey)
		if err != nil {
			return nil, fmt.Errorf("unable to open private key")
		}

		signer, err := ssh.ParsePrivateKeyWithPassphrase(key, []byte(dbConnCfg.SSHCfg.PassPhrase))
		if err != nil {
			return nil, fmt.Errorf("unable to decrypt private key")
		}

		cfg := &ssh.ClientConfig{
			User: dbConnCfg.SSHCfg.User,
			Auth: []ssh.AuthMethod{
				ssh.PublicKeys(signer),
			},
			HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		}

		remoteAddr := fmt.Sprintf("%s:%d", dbConnCfg.SSHCfg.Host, dbConnCfg.SSHCfg.Port)

		tunnel, err := sshdb.New(cfg, remoteAddr)
		if err != nil {
			return nil, fmt.Errorf("%w", err)
		}

		connector, err := tunnel.OpenConnector(mssql.TunnelDriver, dsn)
		if err != nil {
			return nil, err
		}

		conn = sql.OpenDB(connector)
	} else {
		conn, err = sql.Open("mssql", dsn)
		if err != nil {
			return nil, err
		}
	}

	if err = conn.PingContext(context.Background()); err != nil {
		return nil, err
	}

	conn.SetMaxIdleConns(DefaultMaxIdleConns)
	conn.SetMaxOpenConns(DefaultMaxOpenConns)

	return &DBConnection{
		Conn: conn,
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
	SELECT SCHEMA_NAME FROM INFORMATION_SCHEMA.SCHEMATA
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
		TABLE_SCHEMA,
		TABLE_NAME
	FROM
		INFORMATION_SCHEMA.TABLES
	ORDER BY
		TABLE_SCHEMA,
		TABLE_NAME
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
	  TABLE_NAME
	FROM
	  INFORMATION_SCHEMA.TABLES
	WHERE
	  TABLE_TYPE = 'BASE TABLE'
	  AND TABLE_SCHEMA NOT IN ('INFORMATION_SCHEMA', 'information_schema')
	ORDER BY
	  TABLE_NAME
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
		c.TABLE_SCHEMA,
		c.TABLE_NAME,
		c.COLUMN_NAME,
		c.DATA_TYPE,
		c.IS_NULLABLE,
		CASE tc.CONSTRAINT_TYPE
			WHEN 'PRIMARY KEY' THEN 'YES'
			ELSE 'NO'
		END,
		c.COLUMN_DEFAULT,
		''
	FROM
		INFORMATION_SCHEMA.COLUMNS c
	LEFT JOIN
		INFORMATION_SCHEMA.CONSTRAINT_COLUMN_USAGE ccu
		ON c.TABLE_NAME = ccu.TABLE_NAME
		AND c.COLUMN_NAME = ccu.COLUMN_NAME
	LEFT JOIN INFORMATION_SCHEMA.TABLE_CONSTRAINTS tc ON
		tc.TABLE_CATALOG = c.TABLE_CATALOG
		AND tc.TABLE_SCHEMA = c.TABLE_SCHEMA
		AND tc.TABLE_NAME = c.TABLE_NAME
		AND tc.CONSTRAINT_NAME = ccu.CONSTRAINT_NAME
	ORDER BY
		c.TABLE_NAME,
		c.ORDINAL_POSITION
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
		c.TABLE_SCHEMA,
		c.TABLE_NAME,
		c.COLUMN_NAME,
		c.DATA_TYPE,
		c.IS_NULLABLE,
		CASE tc.CONSTRAINT_TYPE
			WHEN 'PRIMARY KEY' THEN 'YES'
			ELSE 'NO'
		END,
		c.COLUMN_DEFAULT,
		''
	FROM
		INFORMATION_SCHEMA.COLUMNS c
	LEFT JOIN
		INFORMATION_SCHEMA.CONSTRAINT_COLUMN_USAGE ccu
		ON c.TABLE_NAME = ccu.TABLE_NAME
		AND c.COLUMN_NAME = ccu.COLUMN_NAME
	LEFT JOIN INFORMATION_SCHEMA.TABLE_CONSTRAINTS tc ON
		tc.TABLE_CATALOG = c.TABLE_CATALOG
		AND tc.TABLE_SCHEMA = c.TABLE_SCHEMA
		AND tc.TABLE_NAME = c.TABLE_NAME
		AND tc.CONSTRAINT_NAME = ccu.CONSTRAINT_NAME
	WHERE
		c.TABLE_SCHEMA = @p1
	ORDER BY
		c.TABLE_NAME,
		c.ORDINAL_POSITION
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

func (db *MssqlDBRepository) DescribeForeignKeysBySchema(ctx context.Context, schemaName string) ([]*ForeignKey, error) {
	rows, err := db.Conn.QueryContext(
		ctx,
		`
		SELECT fk.name,
		   src_tbl.name,
		   src_col.name,
		   dst_tbl.name,
		   dst_col.name
	FROM sys.foreign_key_columns fkc
			 JOIN sys.objects fk on fk.object_id = fkc.constraint_object_id
			 JOIN sys.tables src_tbl
				  ON src_tbl.object_id = fkc.parent_object_id
			 JOIN sys.schemas sch
				  ON src_tbl.schema_id = sch.schema_id
			 JOIN sys.columns src_col
				  ON src_col.column_id = parent_column_id AND src_col.object_id = src_tbl.object_id
			 JOIN sys.tables dst_tbl
				  ON dst_tbl.object_id = fkc.referenced_object_id
			 JOIN sys.columns dst_col
				  ON dst_col.column_id = referenced_column_id AND dst_col.object_id = dst_tbl.object_id
	where sch.name = @p1
	order by fk.name, fkc.constraint_object_id
		`, schemaName)
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = rows.Close() }()
	return parseForeignKeys(rows, schemaName)
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
	case ProtoUDP, ProtoUnix, ProtoHTTP:
	default:
		return "", fmt.Errorf("default addr for network %s unknown", connCfg.Proto)
	}

	for k, v := range connCfg.Params {
		q.Set(k, v)
	}

	return genOptions(q, "", "=", ";", ",", true), nil
}
