package main

import (
	"log"

	"github.com/akito0107/xsqlparser/sqlast"
	"github.com/akito0107/xsqlparser/sqltoken"
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

// func getCompletionTypes(tokens []*sqltoken.Token, curIndex int, curToken *sqltoken.Token) []CompletionType {
// 	log.Println(fmt.Sprintf("current token. idx %d, token `%s`", curIndex, getTokenString(curToken)))
// 	var beforeToken *sqltoken.Token
// 	if curIndex != 0 {
// 		beforeToken = tokens[curIndex-1]
// 	}
//
// 	if beforeToken == nil {
// 		return []CompletionType{CompletionTypeKeyword}
// 	}
//
// 	beforeTokenValue := getTokenString(beforeToken)
// 	log.Println(fmt.Sprintf("current token. idx %d, token `%s`", curIndex-1, beforeTokenValue))
//
// 	var res []CompletionType
// 	switch strings.ToUpper(beforeTokenValue) {
// 	case "SET", "ORDER BY", "DISTINCT":
// 		res = []CompletionType{
// 			CompletionTypeColumn,
// 			CompletionTypeTable,
// 		}
// 	case "AS":
// 		res = []CompletionType{}
// 	case "TO":
// 		res = []CompletionType{
// 			CompletionTypeChange,
// 		}
// 	case "USER", "FOR":
// 		res = []CompletionType{
// 			CompletionTypeUser,
// 		}
// 	case "SELECT", "WHERE", "HAVING":
// 		res = []CompletionType{
// 			CompletionTypeColumn,
// 			CompletionTypeTable,
// 			CompletionTypeView,
// 			CompletionTypeFunction,
// 		}
// 	case "JOIN", "COPY", "FROM", "UPDATE", "INTO", "DESCRIBE", "TRUNCATE", "DESC", "EXPLAIN":
// 		res = []CompletionType{
// 			CompletionTypeColumn,
// 			CompletionTypeTable,
// 			CompletionTypeView,
// 			CompletionTypeFunction,
// 		}
// 	case "ON":
// 		res = []CompletionType{
// 			CompletionTypeColumn,
// 			CompletionTypeTable,
// 			CompletionTypeView,
// 			CompletionTypeFunction,
// 		}
// 	case "USE", "DATABASE", "TEMPLATE", "CONNECT":
// 		res = []CompletionType{
// 			CompletionTypeDatabase,
// 		}
// 	default:
// 		log.Printf("unknown token, got %s", beforeTokenValue)
// 		res = []CompletionType{
// 			CompletionTypeKeyword,
// 		}
// 	}
// 	return res
// }
//
// func getTokenString(token *sqltoken.Token) string {
// 	log.Println(token.Kind.String())
// 	switch v := token.Value.(type) {
// 	case *sqltoken.SQLWord:
// 		return v.String()
// 	case string:
// 		return v
// 	default:
// 		log.Printf("unknown token value, got %T", v)
// 		return " "
// 	}
// }
//
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
//
// func getLine(text string, line int) string {
// 	scanner := bufio.NewScanner(strings.NewReader(text))
// 	i := 1
// 	for scanner.Scan() {
// 		if i == line {
// 			return scanner.Text()
// 		}
// 		i++
// 	}
// 	return ""
// }
//
// func getLastWord(text string, line, char int) string {
// 	t := getBeforeCursorText(text, line, char)
// 	s := getLine(t, line)
//
// 	reg := regexp.MustCompile(`\w+`)
// 	ss := reg.FindAllString(s, -1)
// 	if len(ss) == 0 {
// 		return ""
// 	}
// 	return ss[len(ss)-1]
// }
//
// func getBeforeCursorText(text string, line, char int) string {
// 	writer := bytes.NewBufferString("")
// 	scanner := bufio.NewScanner(strings.NewReader(text))
//
// 	i := 1
// 	for scanner.Scan() {
// 		if i == line {
// 			t := scanner.Text()
// 			writer.Write([]byte(t[:char]))
// 			break
// 		}
// 		writer.Write([]byte(fmt.Sprintln(scanner.Text())))
// 		i++
// 	}
// 	return writer.String()
// }

func PathEnclosingInterval(root sqlast.Node, start, end sqltoken.Pos) (path []sqlast.Node, exact bool) {
	// fmt.Printf("EnclosingInterval %d %d\n", start, end) // debugging

	// Precondition: node.[Pos..End) and adjoining whitespace contain [start, end).
	var visit func(node sqlast.Node) bool
	visit = func(node sqlast.Node) bool {
		path = append(path, node)

		nodePos := node.Pos()
		nodeEnd := node.End()

		// fmt.Printf("visit(%T, %d, %d)\n", node, nodePos, nodeEnd) // debugging

		// Intersect [start, end) with interval of node.
		if 0 > sqltoken.ComparePos(start, nodePos) {
			start = nodePos
		}
		if 0 < sqltoken.ComparePos(end, nodeEnd) {
			end = nodeEnd
		}

		// Find sole child that contains [start, end).
		children := childrenOf(node)
		l := len(children)
		for i, child := range children {
			// [childPos, childEnd) is unaugmented interval of child.
			childPos := child.Pos()
			childEnd := child.End()

			// [augPos, augEnd) is whitespace-augmented interval of child.
			augPos := childPos
			augEnd := childEnd
			if i > 0 {
				augPos = children[i-1].End() // start of preceding whitespace
			}
			if i < l-1 {
				nextChildPos := children[i+1].Pos()
				// Does [start, end) lie between child and next child?
				if 0 <= sqltoken.ComparePos(start, augEnd) && 0 >= sqltoken.ComparePos(end, nextChildPos) {
					start = nodePos
				}
				// if start >= augEnd && end <= nextChildPos {
				// 	return false // inexact match
				// }
				augEnd = nextChildPos // end of following whitespace
			}

			// fmt.Printf("\tchild %d: [%d..%d)\tcontains interval [%d..%d)?\n",
			// 	i, augPos, augEnd, start, end) // debugging

			// Does augmented child strictly contain [start, end)?
			// equals augPos <= start && end <= augEnd {
			if 0 >= sqltoken.ComparePos(augPos, start) && 0 >= sqltoken.ComparePos(end, augEnd) {
				_, isToken := child.(sqlast.Node)
				return isToken || visit(child)
			}

			// Does [start, end) overlap multiple children?
			// i.e. left-augmented child contains start
			// but LR-augmented child does not contain end.

			// equals [start < childEnd && end > augEnd]
			if 0 > sqltoken.ComparePos(start, childEnd) && 0 < sqltoken.ComparePos(end, augEnd) {
				break
			}
		}

		// No single child contained [start, end),
		// so node is the result.  Is it exact?

		// (It's tempting to put this condition before the
		// child loop, but it gives the wrong result in the
		// case where a node (e.g. ExprStmt) and its sole
		// child have equal intervals.)
		if start == nodePos && end == nodeEnd {
			return true // exact match
		}

		return false // inexact: overlaps multiple children
	}

	// start > end
	if 0 < sqltoken.ComparePos(start, end) {
		start, end = end, start
	}

	// start < root.End() && end > root.Pos()
	if 0 > sqltoken.ComparePos(start, root.End()) && 0 < sqltoken.ComparePos(end, root.Pos()) {
		if start == end {
			end.Col = start.Col + 1 // empty interval => interval of size 1
		}
		exact = visit(root)

		// Reverse the path:
		for i, l := 0, len(path); i < l/2; i++ {
			path[i], path[l-1-i] = path[l-1-i], path[i]
		}
	} else {
		// Selection lies within whitespace preceding the
		// first (or following the last) declaration in the file.
		// The result nonetheless always includes the ast.File.
		path = append(path, root)
	}

	return
}

// childrenOf returns the direct non-nil children of ast.Node n.
// It may include fake ast.Node implementations for bare tokens.
// it is not safe to call (e.g.) ast.Walk on such nodes.
//
func childrenOf(n sqlast.Node) []sqlast.Node {
	var children []sqlast.Node
	return children
}
