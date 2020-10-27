package completer

import (
	"bufio"
	"bytes"
	"fmt"
	"regexp"
	"strings"

	"github.com/lighttiger2505/sqls/ast"
	"github.com/lighttiger2505/sqls/ast/astutil"
	"github.com/lighttiger2505/sqls/internal/database"
	"github.com/lighttiger2505/sqls/internal/lsp"
	"github.com/lighttiger2505/sqls/parser"
	"github.com/lighttiger2505/sqls/parser/parseutil"
	"github.com/lighttiger2505/sqls/token"
)

type completionType int

const (
	_ completionType = iota
	CompletionTypeKeyword
	CompletionTypeFunction
	CompletionTypeColumn
	CompletionTypeTable
	CompletionTypeReferencedTable
	CompletionTypeView
	CompletionTypeSubQuery
	CompletionTypeSubQueryColumn
	CompletionTypeChange
	CompletionTypeUser
	CompletionTypeSchema
)

func (ct completionType) String() string {
	switch ct {
	case CompletionTypeKeyword:
		return "Keyword"
	case CompletionTypeFunction:
		return "Function"
	case CompletionTypeColumn:
		return "Column"
	case CompletionTypeTable:
		return "Table"
	case CompletionTypeReferencedTable:
		return "ReferencedTable"
	case CompletionTypeView:
		return "View"
	case CompletionTypeChange:
		return "Change"
	case CompletionTypeUser:
		return "User"
	case CompletionTypeSchema:
		return "Schema"
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
	DBCache *database.DBCache
}

func NewCompleter(dbCache *database.DBCache) *Completer {
	return &Completer{
		DBCache: dbCache,
	}
}

func completionTypeIs(completionTypes []completionType, expect completionType) bool {
	for _, t := range completionTypes {
		if t == expect {
			return true
		}
	}
	return false
}

func (c *Completer) Complete(text string, params lsp.CompletionParams) ([]lsp.CompletionItem, error) {
	parsed, err := parser.Parse(text)
	if err != nil {
		return nil, err
	}

	var pos token.Pos
	// NOTE work around
	if params.Position.Line == 0 {
		pos = token.Pos{
			Line: params.Position.Line,
			Col:  params.Position.Character,
		}
	} else {
		pos = token.Pos{
			Line: params.Position.Line,
			Col:  params.Position.Character + 1,
		}
	}

	nodeWalker := parseutil.NewNodeWalker(parsed, pos)
	ctx := getCompletionTypes(nodeWalker)
	if err != nil {
		return nil, err
	}

	definedTables, err := parseutil.ExtractTable(parsed, pos)
	if err != nil {
		return nil, err
	}
	definedSubQuerys, err := parseutil.ExtractSubQueryViews(parsed, pos)
	if err != nil {
		return nil, err
	}

	items := []lsp.CompletionItem{}
	if completionTypeIs(ctx.types, CompletionTypeColumn) {
		items = append(items, c.columnCandidates(definedTables, ctx.parent)...)
	}
	if completionTypeIs(ctx.types, CompletionTypeReferencedTable) {
		items = append(items, c.ReferencedTableCandidates(definedTables)...)
	}
	if completionTypeIs(ctx.types, CompletionTypeTable) {
		items = append(items, c.TableCandidates(ctx.parent, definedTables)...)
	}
	if completionTypeIs(ctx.types, CompletionTypeSchema) {
		items = append(items, c.SchemaCandidates()...)
	}
	if completionTypeIs(ctx.types, CompletionTypeSubQuery) {
		items = append(items, c.SubQueryCandidates(definedSubQuerys)...)
	}
	if completionTypeIs(ctx.types, CompletionTypeSubQueryColumn) {
		items = append(items, c.SubQueryColumnCandidates(definedSubQuerys)...)
	}
	if completionTypeIs(ctx.types, CompletionTypeKeyword) {
		items = append(items, c.keywordCandidates()...)
	}

	lastWord := getLastWord(text, params.Position.Line+1, params.Position.Character)
	items = filterCandidates(items, lastWord)

	return items, nil
}

type ParentType int

const (
	_ ParentType = iota
	ParentTypeNone
	ParentTypeSchema
	ParentTypeTable
	ParentTypeSubQuery
)

type completionParent struct {
	Type ParentType
	Name string
}

var noneParent = &completionParent{Type: ParentTypeNone}

type CompletionContext struct {
	types  []completionType
	parent *completionParent
}

func getCompletionTypes(nw *parseutil.NodeWalker) *CompletionContext {
	memberIdentifierMatcher := astutil.NodeMatcher{
		NodeTypes: []ast.NodeType{ast.TypeMemberIdentifer},
	}

	syntaxPos := parseutil.CheckSyntaxPosition(nw)
	t := []completionType{}
	p := noneParent
	switch {
	case syntaxPos == parseutil.ColName:
		if nw.CurNodeIs(memberIdentifierMatcher) {
			// has parent
			mi := nw.CurNodeTopMatched(memberIdentifierMatcher).(*ast.MemberIdentifer)
			t = []completionType{
				CompletionTypeColumn,
				CompletionTypeSubQueryColumn,
				CompletionTypeView,
				CompletionTypeFunction,
			}
			p = &completionParent{
				Type: ParentTypeTable,
				Name: mi.Parent.String(),
			}
		} else {
			t = []completionType{
				CompletionTypeColumn,
				CompletionTypeTable,
				CompletionTypeReferencedTable,
				CompletionTypeSubQueryColumn,
				CompletionTypeSubQuery,
				CompletionTypeView,
				CompletionTypeFunction,
				CompletionTypeKeyword,
			}
			p = noneParent
		}
	case syntaxPos == parseutil.AliasName:
		// pass
	case syntaxPos == parseutil.SelectExpr || syntaxPos == parseutil.CaseValue:
		if nw.CurNodeIs(memberIdentifierMatcher) {
			// has parent
			mi := nw.CurNodeTopMatched(memberIdentifierMatcher).(*ast.MemberIdentifer)
			t = []completionType{
				CompletionTypeColumn,
				CompletionTypeView,
				CompletionTypeSubQueryColumn,
				CompletionTypeFunction,
			}
			p = &completionParent{
				Type: ParentTypeTable,
				Name: mi.Parent.String(),
			}
		} else {
			t = []completionType{
				CompletionTypeColumn,
				CompletionTypeTable,
				CompletionTypeReferencedTable,
				CompletionTypeView,
				CompletionTypeSubQueryColumn,
				CompletionTypeSubQuery,
				CompletionTypeFunction,
				CompletionTypeKeyword,
			}
		}
	case syntaxPos == parseutil.TableReference:
		if nw.CurNodeIs(memberIdentifierMatcher) {
			// has parent
			mi := nw.CurNodeTopMatched(memberIdentifierMatcher).(*ast.MemberIdentifer)
			t = []completionType{
				CompletionTypeTable,
				CompletionTypeView,
				CompletionTypeSubQueryColumn,
				CompletionTypeFunction,
			}
			p = &completionParent{
				Type: ParentTypeSchema,
				Name: mi.Parent.String(),
			}
		} else {
			t = []completionType{
				CompletionTypeTable,
				CompletionTypeReferencedTable,
				CompletionTypeSchema,
				CompletionTypeView,
				CompletionTypeSubQuery,
				CompletionTypeKeyword,
			}
		}
	case syntaxPos == parseutil.WhereCondition:
		if nw.CurNodeIs(memberIdentifierMatcher) {
			// has parent
			mi := nw.CurNodeTopMatched(memberIdentifierMatcher).(*ast.MemberIdentifer)
			t = []completionType{
				CompletionTypeColumn,
				CompletionTypeView,
				CompletionTypeSubQueryColumn,
				CompletionTypeFunction,
			}
			p = &completionParent{
				Type: ParentTypeTable,
				Name: mi.Parent.String(),
			}
		} else {
			t = []completionType{
				CompletionTypeColumn,
				CompletionTypeTable,
				CompletionTypeReferencedTable,
				CompletionTypeView,
				CompletionTypeSubQueryColumn,
				CompletionTypeSubQuery,
				CompletionTypeFunction,
				CompletionTypeKeyword,
			}
		}
	case syntaxPos == parseutil.InsertValue:
		t = []completionType{
			CompletionTypeColumn,
			CompletionTypeTable,
			CompletionTypeView,
		}
	default:
		t = []completionType{
			CompletionTypeKeyword,
		}
	}
	return &CompletionContext{
		types:  t,
		parent: p,
	}
}

func filterCandidates(candidates []lsp.CompletionItem, lastWord string) []lsp.CompletionItem {
	filterd := []lsp.CompletionItem{}
	for _, candidate := range candidates {
		if strings.HasPrefix(strings.ToUpper(candidate.Label), strings.ToUpper(lastWord)) {
			filterd = append(filterd, candidate)
		}
	}
	return filterd
}

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

	reg := regexp.MustCompile(`\w+$`)
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
