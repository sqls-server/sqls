package database

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net"
	"net/url"
	"strconv"
	"strings"

	clickhouse "github.com/ClickHouse/clickhouse-go/v2"
	"github.com/sqls-server/sqls/dialect"
	"golang.org/x/crypto/ssh"
)

func init() {
	RegisterOpen("clickhouse", clickhouseOpen)
	RegisterFactory("clickhouse", NewClickhouseRepository)
}

func clickhouseOpen(dbConnCfg *DBConfig) (*DBConnection, error) {
	var (
		conn    *sql.DB
		sshConn *ssh.Client
	)

	dsn, err := genClickhouseDsn(dbConnCfg)

	if err != nil {
		return nil, err
	}

	if dbConnCfg.SSHCfg != nil {
		dbConn, dbSSHConn, err := openClickhouseViaSSH(dsn, dbConnCfg.SSHCfg)
		if err != nil {
			return nil, err
		}
		conn = dbConn
		sshConn = dbSSHConn
	} else {
		dbConn, err := sql.Open("clickhouse", dsn)
		if err != nil {
			return nil, err
		}
		conn = dbConn
	}

	if err = conn.PingContext(context.Background()); err != nil {
		return nil, err
	}

	conn.SetMaxIdleConns(DefaultMaxIdleConns)
	conn.SetMaxOpenConns(DefaultMaxOpenConns)

	return &DBConnection{
		Conn:    conn,
		SSHConn: sshConn,
	}, nil
}

func openClickhouseViaSSH(dsn string, sshCfg *SSHConfig) (*sql.DB, *ssh.Client, error) {
	sshConfig, err := sshCfg.ClientConfig()
	if err != nil {
		return nil, nil, err
	}
	sshConn, err := ssh.Dial("tcp", sshCfg.Endpoint(), sshConfig)
	if err != nil {
		return nil, nil, fmt.Errorf("cannot ssh dial, %w", err)
	}

	conf, err := clickhouse.ParseDSN(dsn)
	if err != nil {
		return nil, nil, err
	}
	conf.DialContext = func(ctx context.Context, addr string) (net.Conn, error) {
		return sshConn.DialContext(ctx, "tcp", addr)
	}

	conn := clickhouse.OpenDB(conf)

	return conn, sshConn, nil
}

func genClickhouseDsn(dbConfig *DBConfig) (string, error) {
	if dbConfig.DataSourceName != "" {
		return dbConfig.DataSourceName, nil
	}

	u := url.URL{}

	switch dbConfig.Proto {
	case ProtoTCP:
		u.Scheme = "clickhouse"

	case ProtoHTTP:
		u.Scheme = "http"

	case ProtoUDP, ProtoUnix:
		return "", fmt.Errorf("unsupported protocol %s", dbConfig.Proto)
	}

	if dbConfig.Passwd == "" {
		u.User = url.User(dbConfig.User)
	} else {
		u.User = url.UserPassword(dbConfig.User, dbConfig.Passwd)
	}

	u.Host = fmt.Sprintf("%s:%d", dbConfig.Host, dbConfig.Port)

	if dbConfig.DBName != "" {
		u.Path = dbConfig.DBName
	}

	values := u.Query()

	for k, v := range dbConfig.Params {
		values.Set(k, v)
	}

	u.RawQuery = values.Encode()

	return u.String(), nil
}

func genClickhouseConfig(dbConfig *DBConfig) (*clickhouse.Options, error) {
	if dbConfig.DataSourceName != "" {
		return clickhouse.ParseDSN(dbConfig.DataSourceName)
	}

	cfg := &clickhouse.Options{}

	cfg.Auth.Username = dbConfig.User
	cfg.Auth.Password = dbConfig.Passwd
	cfg.Auth.Database = dbConfig.DBName

	switch dbConfig.Proto {
	case ProtoTCP, ProtoHTTP:
		host, port := dbConfig.Host, dbConfig.Port
		if host == "" {
			host = "127.0.0.1"
		}
		if port == 0 {
			port = 8443
		}
		cfg.Addr = append(cfg.Addr, host+":"+strconv.Itoa(port))

		if dbConfig.Proto == ProtoTCP {
			cfg.Protocol = clickhouse.Native
		} else {
			cfg.Protocol = clickhouse.HTTP
		}
	case ProtoUnix, ProtoUDP:
	default:
		return nil, fmt.Errorf("default addr for network %s unknown", dbConfig.Proto)
	}

	cfg.Settings = parseClickhouseSettings(dbConfig.Params)

	return nil, nil
}

func parseClickhouseSettings(params map[string]string) clickhouse.Settings {
	result := clickhouse.Settings{}

	for k, v := range params {
		switch p := strings.ToLower(v); p {
		case "true":
			result[k] = int(1)
		case "false":
			result[k] = int(0)
		default:
			if n, err := strconv.Atoi(p); err == nil {
				result[k] = n
			} else {
				result[k] = p
			}
		}
	}

	return result
}

type clickhouseSQLDBRepository struct {
	Conn *sql.DB
}

func NewClickhouseRepository(conn *sql.DB) DBRepository {
	return &clickhouseSQLDBRepository{Conn: conn}
}

func (db *clickhouseSQLDBRepository) CurrentDatabase(ctx context.Context) (string, error) {
	row := db.Conn.QueryRowContext(ctx, "SELECT currentDatabase()")
	var database string
	if err := row.Scan(&database); err != nil {
		return "", err
	}
	return database, nil
}

func (db *clickhouseSQLDBRepository) CurrentSchema(ctx context.Context) (string, error) {
	return db.CurrentDatabase(ctx)
}

func (db *clickhouseSQLDBRepository) Databases(ctx context.Context) ([]string, error) {
	rows, err := db.Conn.QueryContext(ctx, "SHOW databases")
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

func (db *clickhouseSQLDBRepository) DescribeDatabaseTable(ctx context.Context) ([]*ColumnDesc, error) {
	rows, err := db.Conn.QueryContext(
		ctx,
		`
SELECT 
    c.database ,
    c.table ,
    c.name,
    c.type ,
    CASE 
        WHEN c.type LIKE 'Nullable(%)' THEN 'YES'
        ELSE 'NO'
    END,
    CASE 
        WHEN c.is_in_primary_key THEN 'YES'
        ELSE 'NO'
    END ,
    c.default_expression,
    ''
FROM 
    system.columns c
WHERE  ( c.database = currentDatabase()
          OR c.database = '' )
       AND c.table NOT LIKE '%inner%'
`)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()
	tableInfos := []*ColumnDesc{}
	for rows.Next() {
		var tableInfo ColumnDesc
		var columnName *string
		err := rows.Scan(
			&tableInfo.Schema,
			&tableInfo.Table,
			&columnName,
			&tableInfo.Type,
			&tableInfo.Null,
			&tableInfo.Key,
			&tableInfo.Default,
			&tableInfo.Extra,
		)
		if err != nil {
			return nil, err
		}
		if columnName != nil {
			tableInfo.Name = *columnName
		}
		tableInfos = append(tableInfos, &tableInfo)
	}
	return tableInfos, nil
}

func (db *clickhouseSQLDBRepository) DescribeDatabaseTableBySchema(ctx context.Context, schemaName string) ([]*ColumnDesc, error) {
	rows, err := db.Conn.QueryContext(
		ctx,
		`
SELECT 
    c.database ,
    c.table ,
    c.name,
    c.type ,
    CASE 
        WHEN c.type LIKE 'Nullable(%)' THEN 'YES'
        ELSE 'NO'
    END,
    CASE 
        WHEN c.is_in_primary_key THEN 'YES'
        ELSE 'NO'
    END ,
    c.default_expression,
    ''
FROM 
    system.columns c
WHERE  ( c.database = ?
          OR c.database = '' )
       AND c.table NOT LIKE '%inner%'
`, schemaName)
	if err != nil {
		log.Println("schema", schemaName, err.Error())
		return nil, err
	}
	defer rows.Close()
	tableInfos := []*ColumnDesc{}
	for rows.Next() {
		var tableInfo ColumnDesc
		var columnName *string
		err := rows.Scan(
			&tableInfo.Schema,
			&tableInfo.Table,
			&columnName,
			&tableInfo.Type,
			&tableInfo.Null,
			&tableInfo.Key,
			&tableInfo.Default,
			&tableInfo.Extra,
		)
		if err != nil {
			return nil, err
		}
		if columnName != nil {
			tableInfo.Name = *columnName
		}
		tableInfos = append(tableInfos, &tableInfo)
	}
	return tableInfos, nil
}

func (*clickhouseSQLDBRepository) DescribeForeignKeysBySchema(ctx context.Context, schemaName string) ([]*ForeignKey, error) {
	// clickhouse doesn't support foreign keys
	return nil, nil
}

func (*clickhouseSQLDBRepository) Driver() dialect.DatabaseDriver {
	return dialect.DatabaseDriverClickhouse
}

func (db *clickhouseSQLDBRepository) Exec(ctx context.Context, query string) (sql.Result, error) {
	return db.Conn.ExecContext(ctx, query)
}

func (db *clickhouseSQLDBRepository) Query(ctx context.Context, query string) (*sql.Rows, error) {
	return db.Conn.QueryContext(ctx, query)
}

func (db *clickhouseSQLDBRepository) SchemaTables(ctx context.Context) (map[string][]string, error) {
	rows, err := db.Conn.QueryContext(
		ctx,
		`
    SELECT table_schema, table_name
      FROM information_schema.tables
     ORDER BY table_schema, table_name
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

func (db *clickhouseSQLDBRepository) Schemas(ctx context.Context) ([]string, error) {
	return db.Databases(ctx)
}
