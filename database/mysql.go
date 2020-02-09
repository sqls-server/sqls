package database

import (
	"database/sql"
	"log"
	"strings"

	_ "github.com/go-sql-driver/mysql"
)

const (
	DefaultMaxIdleConns = 10
	DefaultMaxOpenConns = 5
)

type DBOption struct {
	MaxIdleConns int
	MaxOpenConns int
}

type ColumnDesc struct {
	Name    string
	Type    string
	Null    string
	Key     string
	Default sql.NullString
	Extra   string
}

type MySQLDB struct {
	ConnString string
	Option     *DBOption
	Conn       *sql.DB
}

func NewMysqlDB(connString string) *MySQLDB {
	return &MySQLDB{
		ConnString: connString,
		Option:     &DBOption{},
	}
}

func (db *MySQLDB) Open() error {
	conn, err := sql.Open("mysql", db.ConnString)
	if err != nil {
		return err
	}
	conn.SetMaxIdleConns(DefaultMaxIdleConns)
	if db.Option.MaxIdleConns != 0 {
		conn.SetMaxIdleConns(db.Option.MaxIdleConns)
	}
	conn.SetMaxOpenConns(DefaultMaxOpenConns)
	if db.Option.MaxOpenConns != 0 {
		conn.SetMaxOpenConns(db.Option.MaxOpenConns)
	}
	db.Conn = conn
	return nil
}

func (db *MySQLDB) Close() error {
	return db.Conn.Close()
}

func (db *MySQLDB) Databases() ([]string, error) {
	rows, err := db.Conn.Query("SHOW DATABASES")
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

func (db *MySQLDB) TableColumns() (map[string][]*ColumnDesc, error) {
	tableMap := map[string][]*ColumnDesc{}
	tables, err := db.Tables()
	if err != nil {
		return nil, err
	}
	for _, table := range tables {
		infos, err := db.DescribeTable(table)
		if err != nil {
			return nil, err
		}
		tableMap[strings.ToUpper(table)] = infos
	}
	return tableMap, nil
}

func (db *MySQLDB) Tables() ([]string, error) {
	rows, err := db.Conn.Query("SHOW TABLES")
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

func (db *MySQLDB) DescribeTable(tableName string) ([]*ColumnDesc, error) {
	rows, err := db.Conn.Query("DESC " + tableName)
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
