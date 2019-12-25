package main

import (
	"fmt"

	"github.com/xwb1989/sqlparser"
)

type CompletionType int

const (
	_ int = iota
	CompletionTypeKeyword
	CompletionTypeFunction
	CompletionTypeAlias
	CompletionTypeColumn
)

var keywords = []string{
	"ACCESS", "ADD", "ALL", "ALTER TABLE", "AND", "ANY", "AS",
	"ASC", "AUTO_INCREMENT", "BEFORE", "BEGIN", "BETWEEN",
	"BIGINT", "BINARY", "BY", "CASE", "CHANGE MASTER TO", "CHAR",
	"CHARACTER SET", "CHECK", "COLLATE", "COLUMN", "COMMENT",
	"COMMIT", "CONSTRAINT", "CREATE", "CURRENT",
	"CURRENT_TIMESTAMP", "DATABASE", "DATE", "DECIMAL", "DEFAULT",
	"DELETE FROM", "DESC", "DESCRIBE", "DROP",
	"ELSE", "END", "ENGINE", "ESCAPE", "EXISTS", "FILE", "FLOAT",
	"FOR", "FOREIGN KEY", "FORMAT", "FROM", "FULL", "FUNCTION",
	"GRANT", "GROUP BY", "HAVING", "HOST", "IDENTIFIED", "IN",
	"INCREMENT", "INDEX", "INSERT INTO", "INT", "INTEGER",
	"INTERVAL", "INTO", "IS", "JOIN", "KEY", "LEFT", "LEVEL",
	"LIKE", "LIMIT", "LOCK", "LOGS", "LONG", "MASTER",
	"MEDIUMINT", "MODE", "MODIFY", "NOT", "NULL", "NUMBER",
	"OFFSET", "ON", "OPTION", "OR", "ORDER BY", "OUTER", "OWNER",
	"PASSWORD", "PORT", "PRIMARY", "PRIVILEGES", "PROCESSLIST",
	"PURGE", "REFERENCES", "REGEXP", "RENAME", "REPAIR", "RESET",
	"REVOKE", "RIGHT", "ROLLBACK", "ROW", "ROWS", "ROW_FORMAT",
	"SAVEPOINT", "SELECT", "SESSION", "SET", "SHARE", "SHOW",
	"SLAVE", "SMALLINT", "SMALLINT", "START", "STOP", "TABLE",
	"THEN", "TINYINT", "TO", "TRANSACTION", "TRIGGER", "TRUNCATE",
	"UNION", "UNIQUE", "UNSIGNED", "UPDATE", "USE", "USER",
	"USING", "VALUES", "VARCHAR", "VIEW", "WHEN", "WHERE", "WITH",
}

type Parser struct{}

// param
// uri
// position
//   start, end
//     line, charctor
//
// return
//   type
//   variables(select alias)
//   tables(table alias)
func (p *Parser) parse(text string) (statementType string, err error) {
	sql := "select `Name`, `CountryCode` from city where `ID` = 1"
	stmt, err := sqlparser.Parse(sql)
	if err != nil {
		return
	}

	switch stmt := stmt.(type) {
	case *sqlparser.Select:
		statementType = "select"
		fmt.Println(stmt)
	default:
		err = fmt.Errorf("invalid stmt")
	}
	return
}

type Completer struct {
	// dbconn
	// tables
	// columns
}

func (c *Completer) complete() ([]CompletionItem, error) {
	completionItems := []CompletionItem{}
	for _, k := range keywords {
		completionItems = append(completionItems, CompletionItem{
			Label: k,
			Kind:  KeywordCompletion,
		},
		)
	}
	return completionItems, nil
}
