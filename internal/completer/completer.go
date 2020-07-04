package completer

import (
	"bufio"
	"bytes"
	"fmt"
	"regexp"
	"strings"

	"github.com/lighttiger2505/sqls/ast"
	"github.com/lighttiger2505/sqls/ast/astutil"
	"github.com/lighttiger2505/sqls/dialect"
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
	CompletionTypeAlias
	CompletionTypeColumn
	CompletionTypeTable
	CompletionTypeView
	CompletionTypeSubQueryView
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
	DBCache *database.DatabaseCache
}

func NewCompleter(dbCache *database.DatabaseCache) *Completer {
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

func parse(text string) (ast.TokenList, error) {
	src := bytes.NewBuffer([]byte(text))
	p, err := parser.NewParser(src, &dialect.GenericSQLDialect{})
	if err != nil {
		return nil, err
	}
	parsed, err := p.Parse()
	if err != nil {
		return nil, err
	}
	return parsed, nil
}

func (c *Completer) Complete(text string, params lsp.CompletionParams) ([]lsp.CompletionItem, error) {
	parsed, err := parse(text)
	if err != nil {
		return nil, err
	}

	pos := token.Pos{Line: params.Position.Line, Col: params.Position.Character}
	nodeWalker := parseutil.NewNodeWalker(parsed, pos)
	ctx := getCompletionTypes(nodeWalker)
	if err != nil {
		return nil, err
	}

	definedTables, err := parseutil.ExtractTable(parsed, pos)
	if err != nil {
		return nil, err
	}
	definedSubQuery, err := parseutil.ExtractSubQueryView(parsed, pos)
	if err != nil {
		return nil, err
	}

	items := []lsp.CompletionItem{}
	if completionTypeIs(ctx.types, CompletionTypeColumn) {
		items = append(items, c.columnCandidates(definedTables, ctx.parent)...)
	}
	if completionTypeIs(ctx.types, CompletionTypeAlias) {
		items = append(items, c.aliasCandidates(definedTables)...)
	}
	if completionTypeIs(ctx.types, CompletionTypeTable) {
		items = append(items, c.TableCandidates(ctx.parent)...)
	}
	if completionTypeIs(ctx.types, CompletionTypeSchema) {
		items = append(items, c.SchemaCandidates()...)
	}
	if completionTypeIs(ctx.types, CompletionTypeSubQueryColumn) {
		items = append(items, c.SubQueryColumnCandidates(definedSubQuery)...)
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

type compltionParent struct {
	Type ParentType
	Name string
}

var noneParent = &compltionParent{Type: ParentTypeNone}

var memberIdentifierMatcher = astutil.NodeMatcher{
	NodeTypes: []ast.NodeType{ast.TypeMemberIdentifer},
}

type CompletionContext struct {
	types  []completionType
	parent *compltionParent
}

func getCompletionTypes(nw *parseutil.NodeWalker) *CompletionContext {
	syntaxPos := checkSyntaxPosition(nw)
	t := []completionType{}
	p := noneParent
	switch {
	case syntaxPos == ColName:
		if nw.CurNodeIs(memberIdentifierMatcher) {
			// has parent
			mi := nw.CurNodeTopMatched(memberIdentifierMatcher).(*ast.MemberIdentifer)
			t = []completionType{
				CompletionTypeColumn,
				CompletionTypeSubQueryColumn,
				CompletionTypeView,
				CompletionTypeFunction,
			}
			p = &compltionParent{
				Type: ParentTypeTable,
				Name: mi.Parent.String(),
			}
		} else {
			t = []completionType{
				CompletionTypeColumn,
				CompletionTypeTable,
				CompletionTypeSubQueryColumn,
				CompletionTypeSubQueryView,
				CompletionTypeAlias,
				CompletionTypeView,
				CompletionTypeFunction,
				CompletionTypeKeyword,
			}
			p = noneParent
		}
	case syntaxPos == AliasName:
	case nw.PrevNodesIs(true, genKeywordMatcher([]string{"AS"})):
		// pass
	case syntaxPos == SelectExpr || syntaxPos == CaseValue:
		if nw.CurNodeIs(memberIdentifierMatcher) {
			// has parent
			mi := nw.CurNodeTopMatched(memberIdentifierMatcher).(*ast.MemberIdentifer)
			t = []completionType{
				CompletionTypeColumn,
				CompletionTypeView,
				CompletionTypeSubQueryColumn,
				CompletionTypeFunction,
			}
			p = &compltionParent{
				Type: ParentTypeTable,
				Name: mi.Parent.String(),
			}
		} else {
			t = []completionType{
				CompletionTypeColumn,
				CompletionTypeTable,
				CompletionTypeAlias,
				CompletionTypeView,
				CompletionTypeSubQueryColumn,
				CompletionTypeSubQueryView,
				CompletionTypeFunction,
				CompletionTypeKeyword,
			}
		}
	case syntaxPos == TableReference:
		if nw.CurNodeIs(memberIdentifierMatcher) {
			// has parent
			mi := nw.CurNodeTopMatched(memberIdentifierMatcher).(*ast.MemberIdentifer)
			t = []completionType{
				CompletionTypeTable,
				CompletionTypeView,
				CompletionTypeSubQueryColumn,
				CompletionTypeFunction,
			}
			p = &compltionParent{
				Type: ParentTypeSchema,
				Name: mi.Parent.String(),
			}
		} else {
			t = []completionType{
				CompletionTypeColumn,
				CompletionTypeTable,
				CompletionTypeSchema,
				CompletionTypeView,
				CompletionTypeSubQueryColumn,
				CompletionTypeSubQueryView,
				CompletionTypeFunction,
				CompletionTypeKeyword,
			}
		}
	case syntaxPos == WhereCondition:
		if nw.CurNodeIs(memberIdentifierMatcher) {
			// has parent
			mi := nw.CurNodeTopMatched(memberIdentifierMatcher).(*ast.MemberIdentifer)
			t = []completionType{
				CompletionTypeTable,
				CompletionTypeView,
				CompletionTypeSubQueryColumn,
				CompletionTypeFunction,
			}
			p = &compltionParent{
				Type: ParentTypeSchema,
				Name: mi.Parent.String(),
			}
		} else {
			t = []completionType{
				CompletionTypeColumn,
				CompletionTypeTable,
				CompletionTypeSchema,
				CompletionTypeView,
				CompletionTypeSubQueryColumn,
				CompletionTypeSubQueryView,
				CompletionTypeFunction,
				CompletionTypeKeyword,
			}
		}
	case nw.CurNodeIs(genTokenMatcher([]token.Kind{token.LParen})) || nw.PrevNodesIs(true, genTokenMatcher([]token.Kind{token.LParen})):
		// for insert columns
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

func genTokenMatcher(tokens []token.Kind) astutil.NodeMatcher {
	return astutil.NodeMatcher{
		ExpectTokens: tokens,
	}
}

func genKeywordMatcher(keywords []string) astutil.NodeMatcher {
	return astutil.NodeMatcher{
		ExpectKeyword: keywords,
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

type SyntaxPosition string

const (
	ColName        SyntaxPosition = "col_name"
	SelectExpr     SyntaxPosition = "select_expr"
	AliasName      SyntaxPosition = "alias_name"
	WhereCondition SyntaxPosition = "where_conditon"
	CaseValue      SyntaxPosition = "case_value"
	TableReference SyntaxPosition = "table_reference"
	Unknown        SyntaxPosition = "unknown"
)

func checkSyntaxPosition(nw *parseutil.NodeWalker) SyntaxPosition {
	var res SyntaxPosition
	switch {
	case nw.PrevNodesIs(true, genKeywordMatcher([]string{
		// INSERT Statement
		"SET",
		// SELECT Statement
		"ORDER BY",
		"GROUP BY",
	})):
		res = ColName
	case nw.PrevNodesIs(true, genKeywordMatcher([]string{
		// SELECT Statement
		"ALL",
		"DISTINCT",
		"DISTINCTROW",
		"SELECT",
	})):
		res = SelectExpr
	case nw.PrevNodesIs(true, genKeywordMatcher([]string{
		// Alias
		"AS",
	})):
		res = AliasName
	case nw.PrevNodesIs(true, genKeywordMatcher([]string{
		// WHERE Clause
		"WHERE",
		"HAVING",
		// JOIN Clause
		"ON",
		// Operator
		"AND",
		"OR",
		"XOR",
	})):
		res = WhereCondition
	case nw.PrevNodesIs(true, genKeywordMatcher([]string{
		// CASE Statement
		"CASE",
		"WHEN",
		"THEN",
		"ELSE",
	})):
		res = CaseValue
	case nw.PrevNodesIs(true, genKeywordMatcher([]string{
		// SELECT Statement
		"FROM",
		// UPDATE Statement
		"UPDATE",
		// DELETE Statement
		"DELETE FROM",
		// INSERT Statement
		"INSERT INTO",
		// JOIN Clause
		"JOIN",
		// DESCRIBE Statement
		"DESCRIBE",
		"DESC",
		// TRUNCATE Statement
		"TRUNCATE",
	})):
		res = TableReference
	default:
		res = Unknown
	}
	return res
}
