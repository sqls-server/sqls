package main

import (
	"database/sql"
	"log"

	_ "github.com/go-sql-driver/mysql"
)

// func main() {
// 	db := NewMysqlDB("root:root@tcp(127.0.0.1:13306)/world")
// 	db.Open()
// 	defer db.Close()
//
// 	tables, err := db.Tables()
// 	if err != nil {
// 		log.Fatal(err)
// 	}
// 	for _, table := range tables {
// 		fmt.Println(table)
// 	}
//
// 	databases, err := db.Databases()
// 	if err != nil {
// 		log.Fatal(err)
// 	}
// 	for _, database := range databases {
// 		fmt.Println(database)
// 	}
//
// 	desc, err := db.DescribeTable("city")
// 	if err != nil {
// 		log.Fatal(err)
// 	}
// 	for _, field := range desc {
// 		fmt.Println(fmt.Sprintf("%+v", field))
// 	}
// }

const (
	DefaultMaxIdleConns = 10
	DefaultMaxOpenConns = 5
)

type DBOption struct {
	MaxIdleConns int
	MaxOpenConns int
}

type TableInfo struct {
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
	defer rows.Close()
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

func (db *MySQLDB) Tables() ([]string, error) {
	rows, err := db.Conn.Query("SHOW TABLES")
	defer rows.Close()
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

func (db *MySQLDB) DescribeTable(tableName string) ([]*TableInfo, error) {
	rows, err := db.Conn.Query("DESC " + tableName)
	defer rows.Close()
	if err != nil {
		log.Fatal(err)
	}
	tableInfos := []*TableInfo{}
	for rows.Next() {
		var tableInfo TableInfo
		if err := rows.Scan(&tableInfo.Name, &tableInfo.Type, &tableInfo.Null, &tableInfo.Key, &tableInfo.Default, &tableInfo.Extra); err != nil {
			return nil, err
		}
		tableInfos = append(tableInfos, &tableInfo)
	}
	return tableInfos, nil
}
