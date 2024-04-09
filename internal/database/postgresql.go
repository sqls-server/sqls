package database

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"unicode"

	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/stdlib"
	"github.com/sqls-server/sqls/dialect"
	"golang.org/x/crypto/ssh"
)

func init() {
	RegisterOpen("postgresql", postgreSQLOpen)
	RegisterFactory("postgresql", NewPostgreSQLDBRepository)
}

func postgreSQLOpen(dbConnCfg *DBConfig) (*DBConnection, error) {
	var (
		conn    *sql.DB
		sshConn *ssh.Client
	)
	dsn, err := genPostgresConfig(dbConnCfg)
	if err != nil {
		return nil, err
	}

	if dbConnCfg.SSHCfg != nil {
		dbConn, dbSSHConn, err := openPostgreSQLViaSSH(dsn, dbConnCfg.SSHCfg)
		if err != nil {
			return nil, err
		}
		conn = dbConn
		sshConn = dbSSHConn
	} else {
		dbConn, err := sql.Open("pgx", dsn)
		if err != nil {
			return nil, err
		}
		conn = dbConn
	}
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

func openPostgreSQLViaSSH(dsn string, sshCfg *SSHConfig) (*sql.DB, *ssh.Client, error) {
	sshConfig, err := sshCfg.ClientConfig()
	if err != nil {
		return nil, nil, err
	}
	sshConn, err := ssh.Dial("tcp", sshCfg.Endpoint(), sshConfig)
	if err != nil {
		return nil, nil, fmt.Errorf("cannot ssh dial, %w", err)
	}

	conf, err := pgx.ParseConfig(dsn)
	if err != nil {
		return nil, nil, err
	}
	conf.DialFunc = func(ctx context.Context, network, addr string) (net.Conn, error) {
		return sshConn.Dial(network, addr)
	}

	conn := stdlib.OpenDB(*conf)

	return conn, sshConn, nil
}

type PostgreSQLDBRepository struct {
	Conn *sql.DB
}

func NewPostgreSQLDBRepository(conn *sql.DB) DBRepository {
	return &PostgreSQLDBRepository{Conn: conn}
}

func (db *PostgreSQLDBRepository) Driver() dialect.DatabaseDriver {
	return dialect.DatabaseDriverPostgreSQL
}

func (db *PostgreSQLDBRepository) CurrentDatabase(ctx context.Context) (string, error) {
	row := db.Conn.QueryRowContext(ctx, "SELECT current_database()")
	var database string
	if err := row.Scan(&database); err != nil {
		return "", err
	}
	return database, nil
}

func (db *PostgreSQLDBRepository) Databases(ctx context.Context) ([]string, error) {
	rows, err := db.Conn.QueryContext(
		ctx,
		`
	SELECT datname FROM pg_database
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

func (db *PostgreSQLDBRepository) CurrentSchema(ctx context.Context) (string, error) {
	row := db.Conn.QueryRowContext(ctx, "SELECT current_schema()")
	var database sql.NullString
	if err := row.Scan(&database); err != nil {
		return "", err
	}
	if database.Valid {
		return database.String, nil
	}
	return "", nil
}

func (db *PostgreSQLDBRepository) Schemas(ctx context.Context) ([]string, error) {
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

func (db *PostgreSQLDBRepository) SchemaTables(ctx context.Context) (map[string][]string, error) {
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

func (db *PostgreSQLDBRepository) Tables(ctx context.Context) ([]string, error) {
	rows, err := db.Conn.QueryContext(
		ctx,
		`
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

func (db *PostgreSQLDBRepository) DescribeDatabaseTable(ctx context.Context) ([]*ColumnDesc, error) {
	rows, err := db.Conn.QueryContext(
		ctx,
		`
	SELECT
		c.table_schema,
		c.table_name,
		c.column_name,
		c.data_type,
		c.is_nullable,
		CASE t.constraint_type
			WHEN 'PRIMARY KEY' THEN 'YES'
			ELSE 'NO'
		END,
		c.column_default,
		''
	FROM
		information_schema.columns c
	LEFT JOIN (
		SELECT
			ccu.table_schema as table_schema,
			ccu.table_name as table_name,
			ccu.column_name as column_name,
			tc.constraint_type as constraint_type
		FROM information_schema.constraint_column_usage ccu
		LEFT JOIN information_schema.table_constraints tc ON
			tc.table_schema = ccu.table_schema
			AND tc.table_name = ccu.table_name
			AND tc.constraint_name = ccu.constraint_name
		WHERE
			tc.constraint_type = 'PRIMARY KEY'
	) as t
		ON c.table_schema = t.table_schema
		AND c.table_name = t.table_name
		AND c.column_name = t.column_name
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

func (db *PostgreSQLDBRepository) DescribeDatabaseTableBySchema(ctx context.Context, schemaName string) ([]*ColumnDesc, error) {
	rows, err := db.Conn.QueryContext(
		ctx,
		`
	SELECT
		c.table_schema,
		c.table_name,
		c.column_name,
		c.data_type,
		c.is_nullable,
		CASE t.constraint_type
			WHEN 'PRIMARY KEY' THEN 'YES'
			ELSE 'NO'
		END,
		c.column_default,
		''
	FROM
		information_schema.columns c
	LEFT JOIN (
		SELECT
			ccu.table_schema as table_schema,
			ccu.table_name as table_name,
			ccu.column_name as column_name,
			tc.constraint_type as constraint_type
		FROM information_schema.constraint_column_usage ccu
		LEFT JOIN information_schema.table_constraints tc ON
			tc.table_schema = ccu.table_schema
			AND tc.table_name = ccu.table_name
			AND tc.constraint_name = ccu.constraint_name
		WHERE
			ccu.table_schema = $1
			AND tc.constraint_type = 'PRIMARY KEY'
	) as t
		ON c.table_schema = t.table_schema
		AND c.table_name = t.table_name
		AND c.column_name = t.column_name
	WHERE
		c.table_schema = $2
	ORDER BY
		c.table_name,
		c.ordinal_position
	`, schemaName, schemaName)
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

func (db *PostgreSQLDBRepository) DescribeForeignKeysBySchema(ctx context.Context, schemaName string) ([]*ForeignKey, error) {
	rows, err := db.Conn.QueryContext(
		ctx,
		`
		SELECT fk.conname AS constraint_name, c1.relname AS table_name, a1.attname AS column_name, c2.relname AS
		    foreign_table_name, a2.attname AS foreign_column_name
		FROM pg_catalog.pg_constraint fk
		    JOIN pg_catalog.pg_class c1 ON c1.oid = fk.conrelid
		    JOIN pg_catalog.pg_attribute a1 ON a1.attrelid = c1.oid
			AND a1.attnum = ANY (fk.conkey)
		    JOIN pg_catalog.pg_class c2 ON c2.oid = fk.confrelid
		    JOIN pg_catalog.pg_attribute a2 ON a2.attrelid = c2.oid
			AND a2.attnum = ANY (fk.confkey)
		WHERE fk.contype = 'f'
		    AND fk.connamespace = $1::regnamespace::oid
		    AND (pg_has_role(c1.relowner, 'USAGE')
			OR has_table_privilege(c1.oid, 'INSERT, UPDATE, DELETE, TRUNCATE, REFERENCES, TRIGGER')
			OR has_any_column_privilege(c1.oid, 'INSERT, UPDATE, REFERENCES'))
		    AND (pg_has_role(c2.relowner, 'USAGE')
			OR has_table_privilege(c2.oid, 'INSERT, UPDATE, DELETE, TRUNCATE, REFERENCES, TRIGGER')
			OR has_any_column_privilege(c2.oid, 'INSERT, UPDATE, REFERENCES'))
		ORDER BY constraint_name, a1.attnum;
		`, schemaName)
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = rows.Close() }()
	return parseForeignKeys(rows, schemaName)
}

func (db *PostgreSQLDBRepository) Exec(ctx context.Context, query string) (sql.Result, error) {
	return db.Conn.ExecContext(ctx, query)
}

func (db *PostgreSQLDBRepository) Query(ctx context.Context, query string) (*sql.Rows, error) {
	return db.Conn.QueryContext(ctx, query)
}

func genPostgresConfig(connCfg *DBConfig) (string, error) {
	if connCfg.DataSourceName != "" {
		return connCfg.DataSourceName, nil
	}

	q := url.Values{}
	q.Set("user", connCfg.User)
	q.Set("password", connCfg.Passwd)
	q.Set("dbname", connCfg.DBName)

	switch connCfg.Proto {
	case ProtoTCP, ProtoUDP:
		host, port := connCfg.Host, connCfg.Port
		if host == "" {
			host = "127.0.0.1"
		}
		if port == 0 {
			port = 5432
		}
		q.Set("host", host)
		q.Set("port", strconv.Itoa(port))
	case ProtoUnix:
		q.Set("host", connCfg.Path)
	case ProtoHTTP:
	default:
		return "", fmt.Errorf("default addr for network %s unknown", connCfg.Proto)
	}

	for k, v := range connCfg.Params {
		q.Set(k, v)
	}

	return genOptions(q, "", "=", " ", ",", true), nil
}

// genOptions takes URL values and generates options, joining together with
// joiner, and separated by sep, with any multi URL values joined by valSep,
// ignoring any values with keys in ignore.
//
// For example, to build a "ODBC" style connection string, use like the following:
//
//	genOptions(u.Query(), "", "=", ";", ",")
func genOptions(q url.Values, joiner, assign, sep, valSep string, skipWhenEmpty bool, ignore ...string) string {
	qlen := len(q)
	if qlen == 0 {
		return ""
	}

	// make ignore map
	ig := make(map[string]bool, len(ignore))
	for _, v := range ignore {
		ig[strings.ToLower(v)] = true
	}

	// sort keys
	s := make([]string, len(q))
	var i int
	for k := range q {
		s[i] = k
		i++
	}
	sort.Strings(s)

	var opts []string
	for _, k := range s {
		if !ig[strings.ToLower(k)] {
			val := strings.Join(q[k], valSep)
			if !skipWhenEmpty || val != "" {
				if val != "" {
					val = assign + val
				}
				opts = append(opts, k+val)
			}
		}
	}

	if len(opts) != 0 {
		return joiner + strings.Join(opts, sep)
	}

	return ""
}

type values map[string]string

// scanner implements a tokenizer for libpq-style option strings.
type scanner struct {
	s []rune
	i int
}

// Next returns the next rune.
// It returns 0, false if the end of the text has been reached.
func (s *scanner) Next() (rune, bool) {
	if s.i >= len(s.s) {
		return 0, false
	}
	r := s.s[s.i]
	s.i++
	return r, true
}

// newScanner returns a new scanner initialized with the option string s.
func newScanner(s string) *scanner {
	return &scanner{[]rune(s), 0}
}

// SkipSpaces returns the next non-whitespace rune.
// It returns 0, false if the end of the text has been reached.
func (s *scanner) SkipSpaces() (rune, bool) {
	r, ok := s.Next()
	for unicode.IsSpace(r) && ok {
		r, ok = s.Next()
	}
	return r, ok
}
