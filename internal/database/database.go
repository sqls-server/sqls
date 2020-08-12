package database

import (
	"bytes"
	"database/sql"
	"errors"
	"fmt"
	"io/ioutil"
	"strings"

	"golang.org/x/crypto/ssh"
	"golang.org/x/xerrors"
)

var (
	ErrNotImplementation error = errors.New("not implementation")
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
