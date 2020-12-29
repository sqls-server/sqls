package database

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/lighttiger2505/sqls/parser/parseutil"
)

var (
	ErrNotImplementation error = errors.New("not implementation")
)

const (
	DefaultMaxIdleConns = 10
	DefaultMaxOpenConns = 5
)

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

type DBOption struct {
	MaxIdleConns int
	MaxOpenConns int
}

type ColumnDesc struct {
	Schema  string
	Table   string
	Name    string
	Type    string
	Null    string
	Key     string
	Default sql.NullString
	Extra   string
}

func (cd *ColumnDesc) OnelineDesc() string {
	items := []string{}
	if cd.Type != "" {
		items = append(items, cd.Type)
	}
	if cd.Key != "" {
		items = append(items, cd.Key)
	}
	if cd.Extra != "" {
		items = append(items, cd.Extra)
	}
	return strings.Join(items, " ")
}

func (cd *ColumnDesc) OnelineDescWithName() string {
	return fmt.Sprintf("%s: %s", cd.Name, cd.OnelineDesc())
}

func ColumnDoc(tableName string, colDesc *ColumnDesc) string {
	buf := new(bytes.Buffer)
	fmt.Fprintf(buf, "%s.%s column", tableName, colDesc.Name)
	fmt.Fprintln(buf)
	fmt.Fprintln(buf)
	fmt.Fprintln(buf, colDesc.OnelineDesc())
	return buf.String()
}

func TableDoc(tableName string, cols []*ColumnDesc) string {
	buf := new(bytes.Buffer)
	fmt.Fprintf(buf, "%s table", tableName)
	fmt.Fprintln(buf)
	fmt.Fprintln(buf)
	for _, col := range cols {
		fmt.Fprintf(buf, "- %s", col.OnelineDescWithName())
		fmt.Fprintln(buf)
	}
	return buf.String()
}

func SubqueryDoc(name string, views []*parseutil.SubQueryView, dbCache *DBCache) string {
	buf := new(bytes.Buffer)
	fmt.Fprintf(buf, "%s subquery", name)
	fmt.Fprintln(buf)
	fmt.Fprintln(buf)
	for _, view := range views {
		for _, colmun := range view.SubQueryColumns {
			if colmun.ColumnName == "*" {
				tableCols, ok := dbCache.ColumnDescs(colmun.ParentTable.Name)
				if !ok {
					continue
				}
				for _, tableCol := range tableCols {
					fmt.Fprintf(buf, "- %s(%s.%s): %s", tableCol.Name, colmun.ParentTable.Name, tableCol.Name, tableCol.OnelineDesc())
					fmt.Fprintln(buf)
				}
			} else {
				columnDesc, ok := dbCache.Column(colmun.ParentTable.Name, colmun.ColumnName)
				if !ok {
					continue
				}
				fmt.Fprintf(buf, "- %s(%s.%s): %s", colmun.DisplayName(), colmun.ParentTable.Name, colmun.ColumnName, columnDesc.OnelineDesc())
				fmt.Fprintln(buf)

			}
		}
	}
	return buf.String()
}

func SubqueryColumnDoc(identName string, views []*parseutil.SubQueryView, dbCache *DBCache) string {
	buf := new(bytes.Buffer)
	fmt.Fprintf(buf, "%s subquery column", identName)
	fmt.Fprintln(buf)
	fmt.Fprintln(buf)
	for _, view := range views {
		for _, colmun := range view.SubQueryColumns {
			if colmun.ColumnName == "*" {
				tableCols, ok := dbCache.ColumnDescs(colmun.ParentTable.Name)
				if !ok {
					continue
				}
				for _, tableCol := range tableCols {
					if identName == tableCol.Name {
						fmt.Fprintf(buf, "- %s(%s.%s): %s", identName, colmun.ParentTable.Name, tableCol.Name, tableCol.OnelineDesc())
						fmt.Fprintln(buf)
						continue
					}
				}
			} else {
				if identName != colmun.ColumnName && identName != colmun.AliasName {
					continue
				}
				columnDesc, ok := dbCache.Column(colmun.ParentTable.Name, colmun.ColumnName)
				if !ok {
					continue
				}
				fmt.Fprintf(buf, "- %s(%s.%s): %s", identName, colmun.ParentTable.Name, colmun.ColumnName, columnDesc.OnelineDesc())
				fmt.Fprintln(buf)
			}
		}
	}
	return buf.String()
}
