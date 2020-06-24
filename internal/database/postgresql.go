package database

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"
	"log"
	"net"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"

	pq "github.com/lib/pq"
	"golang.org/x/crypto/ssh"
	"golang.org/x/xerrors"
)

type PostgreSQLDB struct {
	Cfg     *Config
	Option  *DBOption
	Conn    *sql.DB
	SSHConn *ssh.Client
	curDB   string
}

func init() {
	Register("postgresql", func(cfg *Config) Database {
		return &PostgreSQLDB{
			Cfg:    cfg,
			Option: &DBOption{},
		}
	})
}

func (db *PostgreSQLDB) Open() error {
	dsn, err := genPostgresConfig(db.Cfg)
	if err != nil {
		return err
	}
	if db.curDB != "" {
		dsn, err = replaceDBName(dsn, db.curDB)
		if err != nil {
			return err
		}
	}

	if db.Cfg.SSHCfg != nil {
		log.Println("via ssh connection")
		dbConn, sshConn, err := openPostgreSQLViaSSH(dsn, db.Cfg.SSHCfg)
		if err != nil {
			return err
		}
		db.Conn = dbConn
		db.SSHConn = sshConn
	} else {
		log.Println("normal connection")
		dbConn, err := sql.Open("postgres", dsn)
		if err != nil {
			return err
		}
		db.Conn = dbConn
	}
	if err := db.Conn.Ping(); err != nil {
		return err
	}

	db.Conn.SetMaxIdleConns(DefaultMaxIdleConns)
	if db.Option.MaxIdleConns != 0 {
		db.Conn.SetMaxIdleConns(db.Option.MaxIdleConns)
	}
	db.Conn.SetMaxOpenConns(DefaultMaxOpenConns)
	if db.Option.MaxOpenConns != 0 {
		db.Conn.SetMaxOpenConns(db.Option.MaxOpenConns)
	}
	return nil
}

type PostgreSQLViaSSHDialer struct {
	client *ssh.Client
}

func (d *PostgreSQLViaSSHDialer) Open(s string) (_ driver.Conn, err error) {
	return pq.DialOpen(d, s)
}

func (d *PostgreSQLViaSSHDialer) Dial(network, address string) (net.Conn, error) {
	return d.client.Dial(network, address)
}

func (d *PostgreSQLViaSSHDialer) DialTimeout(network, address string, timeout time.Duration) (net.Conn, error) {
	return d.client.Dial(network, address)
}

var (
	sshConnCount int = 1
)

func openPostgreSQLViaSSH(dsn string, sshCfg *SSHConfig) (*sql.DB, *ssh.Client, error) {
	sshConfig, err := sshCfg.ClientConfig()
	if err != nil {
		return nil, nil, err
	}
	sshConn, err := ssh.Dial("tcp", sshCfg.Endpoint(), sshConfig)
	if err != nil {
		return nil, nil, xerrors.Errorf("cannot ssh dial, %+v", err)
	}

	// NOTE: This is a workaround to avoid the panic that occurs in the specifications of sql.driver
	// See https://pkg.go.dev/database/sql#Register
	// > If Register is called twice with the same name or if driver is nil, it panics
	viaSSHDriver := "postgres+ssh" + strconv.Itoa(sshConnCount)
	sql.Register(viaSSHDriver, &PostgreSQLViaSSHDialer{sshConn})
	sshConnCount++

	conn, err := sql.Open(viaSSHDriver, dsn)
	if err != nil {
		return nil, nil, xerrors.Errorf("cannot connect database, %+v", err)
	}
	return conn, sshConn, nil
}

func (db *PostgreSQLDB) Close() error {
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

func (db *PostgreSQLDB) Databases() ([]string, error) {
	rows, err := db.Conn.Query(`
	SELECT datname FROM pg_database
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

func (db *PostgreSQLDB) DatabaseTables() (map[string][]string, error) {
	return nil, ErrNotImplementation
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

func (db *PostgreSQLDB) DescribeDatabaseTable() ([]*ColumnDesc, error) {
	return nil, ErrNotImplementation
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
	db.curDB = dbName
	return nil
}

func genPostgresConfig(connCfg *Config) (string, error) {
	if connCfg.DataSourceName != "" {
		return connCfg.DataSourceName, nil
	}

	q := url.Values{}
	q.Set("user", connCfg.User)
	q.Set("password", connCfg.Passwd)
	q.Set("dbname", connCfg.DBName)

	switch connCfg.Proto {
	case ProtoTCP:
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
	default:
		return "", xerrors.Errorf("default addr for network %s unknown", connCfg.Proto)
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
//     genOptions(u.Query(), "", "=", ";", ",")
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

// The parsing code is based on conninfo_parse from libpq's fe-connect.c
func parseOpts(name string, o values) error {
	s := newScanner(name)

	for {
		var (
			keyRunes, valRunes []rune
			r                  rune
			ok                 bool
		)

		if r, ok = s.SkipSpaces(); !ok {
			break
		}

		// Scan the key
		for !unicode.IsSpace(r) && r != '=' {
			keyRunes = append(keyRunes, r)
			if r, ok = s.Next(); !ok {
				break
			}
		}

		// Skip any whitespace if we're not at the = yet
		if r != '=' {
			r, ok = s.SkipSpaces()
		}

		// The current character should be =
		if r != '=' || !ok {
			return fmt.Errorf(`missing "=" after %q in connection info string"`, string(keyRunes))
		}

		// Skip any whitespace after the =
		if r, ok = s.SkipSpaces(); !ok {
			// If we reach the end here, the last value is just an empty string as per libpq.
			o[string(keyRunes)] = ""
			break
		}

		if r != '\'' {
			for !unicode.IsSpace(r) {
				if r == '\\' {
					if r, ok = s.Next(); !ok {
						return fmt.Errorf(`missing character after backslash`)
					}
				}
				valRunes = append(valRunes, r)

				if r, ok = s.Next(); !ok {
					break
				}
			}
		} else {
		quote:
			for {
				if r, ok = s.Next(); !ok {
					return fmt.Errorf(`unterminated quoted string literal in connection string`)
				}
				switch r {
				case '\'':
					break quote
				case '\\':
					r, _ = s.Next()
					fallthrough
				default:
					valRunes = append(valRunes, r)
				}
			}
		}

		o[string(keyRunes)] = string(valRunes)
	}

	return nil
}

func parseURL(opts values) (string, error) {
	var kvs []string
	escaper := strings.NewReplacer(` `, `\ `, `'`, `\'`, `\`, `\\`)
	accrue := func(k, v string) {
		if v != "" {
			kvs = append(kvs, k+"="+escaper.Replace(v))
		}
	}

	for k, v := range opts {
		accrue(k, v)
	}

	sort.Strings(kvs) // Makes testing easier (not a performance concern)
	return strings.Join(kvs, " "), nil
}

func replaceDBName(dsn, dbName string) (string, error) {
	o := make(values)
	if err := parseOpts(dsn, o); err != nil {
		return "", err
	}
	o["dbname"] = dbName
	newDSN, err := parseURL(o)
	if err != nil {
		return "", err
	}
	return newDSN, nil
}
