package main

import (
	"bufio"
	"bytes"
	"fmt"
	"log"
	"regexp"
	"strings"

	"github.com/lighttiger2505/sqls/dialect"
	"github.com/lighttiger2505/sqls/parser"
)

type CompletionType int

const (
	_ CompletionType = iota
	CompletionTypeKeyword
	CompletionTypeFunction
	CompletionTypeAlias
	CompletionTypeColumn
	CompletionTypeTable
	CompletionTypeView
	CompletionTypeChange
	CompletionTypeUser
	CompletionTypeDatabase
)

func (ct CompletionType) String() string {
	switch ct {
	case CompletionTypeKeyword:
		return "Keyword"
	case CompletionTypeFunction:
		return "Function"
	case CompletionTypeAlias:
		return "Alias"
	case CompletionTypeColumn:
		return "Column"
	case CompletionTypeTable:
		return "Table"
	case CompletionTypeView:
		return "View"
	case CompletionTypeChange:
		return "Change"
	case CompletionTypeUser:
		return "User"
	case CompletionTypeDatabase:
		return "Database"
	default:
		return ""
	}
}

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

type Completer struct {
	// dbconn
	// tables
	// columns
}

func (c *Completer) complete(text string, params CompletionParams) ([]CompletionItem, error) {
	completionItems := []CompletionItem{}
	log.Println(text)

	for _, k := range keywords {
		completionItems = append(completionItems, CompletionItem{
			Label: k,
			Kind:  KeywordCompletion,
		},
		)
	}
	return completionItems, nil
}

func parse(text string, line, char int) ([]CompletionType, error) {
	src := bytes.NewBuffer([]byte(text))
	parser, err := parser.NewParser(src, &dialect.GenericSQLDialect{})
	if err != nil {
		return nil, err
	}
	parsed, err := parser.Parse()
	if err != nil {
		return nil, err
	}
	return []CompletionType{}, nil
}

func getCompletionTypes(beforeTokenValue string) []CompletionType {
	var res []CompletionType
	switch strings.ToUpper(beforeTokenValue) {
	case "SET", "ORDER BY", "DISTINCT":
		res = []CompletionType{
			CompletionTypeColumn,
			CompletionTypeTable,
		}
	case "AS":
		res = []CompletionType{}
	case "TO":
		res = []CompletionType{
			CompletionTypeChange,
		}
	case "USER", "FOR":
		res = []CompletionType{
			CompletionTypeUser,
		}
	case "SELECT", "WHERE", "HAVING":
		res = []CompletionType{
			CompletionTypeColumn,
			CompletionTypeTable,
			CompletionTypeView,
			CompletionTypeFunction,
		}
	case "JOIN", "COPY", "FROM", "UPDATE", "INTO", "DESCRIBE", "TRUNCATE", "DESC", "EXPLAIN":
		res = []CompletionType{
			CompletionTypeColumn,
			CompletionTypeTable,
			CompletionTypeView,
			CompletionTypeFunction,
		}
	case "ON":
		res = []CompletionType{
			CompletionTypeColumn,
			CompletionTypeTable,
			CompletionTypeView,
			CompletionTypeFunction,
		}
	case "USE", "DATABASE", "TEMPLATE", "CONNECT":
		res = []CompletionType{
			CompletionTypeDatabase,
		}
	default:
		log.Printf("unknown token, got %s", beforeTokenValue)
		res = []CompletionType{
			CompletionTypeKeyword,
		}
	}
	return res
}

// func getLastToken(tokens []*sqltoken.Token, line, char int) (int, *sqltoken.Token) {
// 	pos := sqltoken.Pos{
// 		Line: line,
// 		Col:  char,
// 	}
// 	var curIndex int
// 	var curToken *sqltoken.Token
// 	for i, token := range tokens {
// 		if 0 <= sqltoken.ComparePos(pos, token.From) {
// 			curToken = token
// 			curIndex = i
// 			if 0 >= sqltoken.ComparePos(pos, token.To) {
// 				return curIndex, curToken
// 			}
// 		}
// 	}
// 	return curIndex, curToken
// }

func getLine(text string, line int) string {
	scanner := bufio.NewScanner(strings.NewReader(text))
	i := 1
	for scanner.Scan() {
		if i == line {
			return scanner.Text()
		}
		i++
	}
	return ""
}

func getLastWord(text string, line, char int) string {
	t := getBeforeCursorText(text, line, char)
	s := getLine(t, line)

	reg := regexp.MustCompile(`\w+`)
	ss := reg.FindAllString(s, -1)
	if len(ss) == 0 {
		return ""
	}
	return ss[len(ss)-1]
}

func getBeforeCursorText(text string, line, char int) string {
	writer := bytes.NewBufferString("")
	scanner := bufio.NewScanner(strings.NewReader(text))

	i := 1
	for scanner.Scan() {
		if i == line {
			t := scanner.Text()
			writer.Write([]byte(t[:char]))
			break
		}
		writer.Write([]byte(fmt.Sprintln(scanner.Text())))
		i++
	}
	return writer.String()
}
