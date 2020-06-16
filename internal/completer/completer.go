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

type CompletionType int

const (
	_ CompletionType = iota
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
	DBCache *database.DatabaseCache
}

func NewCompleter(dbCache *database.DatabaseCache) *Completer {
	return &Completer{
		DBCache: dbCache,
	}
}

func completionTypeIs(completionTypes []CompletionType, expect CompletionType) bool {
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
	cTypes, pare, err := getCompletionTypes(text, pos)
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
	if completionTypeIs(cTypes, CompletionTypeColumn) {
		items = append(items, c.columnCandidates(definedTables, pare)...)
	}
	if completionTypeIs(cTypes, CompletionTypeAlias) {
		items = append(items, c.aliasCandidates(definedTables)...)
	}
	if completionTypeIs(cTypes, CompletionTypeTable) {
		items = append(items, c.TableCandidates()...)
	}
	if completionTypeIs(cTypes, CompletionTypeSubQueryColumn) {
		items = append(items, c.SubQueryColumnCandidates(definedSubQuery)...)
	}
	if completionTypeIs(cTypes, CompletionTypeKeyword) {
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

type parent struct {
	Type ParentType
	Name string
}

var noneParent = &parent{Type: ParentTypeNone}

var memberIdentifierMatcher = astutil.NodeMatcher{
	NodeTypes: []ast.NodeType{ast.TypeMemberIdentifer},
}

func getCompletionTypes(text string, pos token.Pos) ([]CompletionType, *parent, error) {
	parsed, err := parse(text)
	if err != nil {
		return nil, nil, err
	}
	nodeWalker := parseutil.NewNodeWalker(parsed, pos)

	switch {
	case nodeWalker.PrevNodesIs(true, genKeywordMatcher([]string{"SET", "ORDER BY", "GROUP BY", "DISTINCT"})):
		if nodeWalker.CurNodeIs(memberIdentifierMatcher) {
			// has parent
			mi := nodeWalker.CurNodeTopMatched(memberIdentifierMatcher).(*ast.MemberIdentifer)
			cType := []CompletionType{
				CompletionTypeColumn,
				CompletionTypeSubQueryColumn,
				CompletionTypeView,
				CompletionTypeFunction,
			}
			tableParent := &parent{
				Type: ParentTypeTable,
				Name: mi.Parent.String(),
			}
			return cType, tableParent, nil
		}
		return []CompletionType{
			CompletionTypeColumn,
			CompletionTypeTable,
			CompletionTypeSubQueryColumn,
			CompletionTypeSubQueryView,
			CompletionTypeAlias,
			CompletionTypeView,
			CompletionTypeFunction,
			CompletionTypeKeyword,
		}, noneParent, nil
	// case nodeWalker.PrevNodesIs(true, genKeywordMatcher([]string{"AS"})):
	// 	res = []CompletionType{}
	// case nodeWalker.PrevNodesIs(true, genKeywordMatcher([]string{"TO"})):
	// 	res = []CompletionType{
	// 		CompletionTypeChange,
	// 	}
	// case nodeWalker.PrevNodesIs(true, genKeywordMatcher([]string{"USER", "FOR"})):
	// 	res = []CompletionType{
	// 		CompletionTypeUser,
	// 	}
	case nodeWalker.PrevNodesIs(true, genKeywordMatcher([]string{"SELECT", "WHERE", "HAVING", "ON", "CASE", "WHEN", "THEN", "ELSE"})):
		if nodeWalker.CurNodeIs(memberIdentifierMatcher) {
			// has parent
			mi := nodeWalker.CurNodeTopMatched(memberIdentifierMatcher).(*ast.MemberIdentifer)
			cType := []CompletionType{
				CompletionTypeColumn,
				CompletionTypeView,
				CompletionTypeSubQueryColumn,
				CompletionTypeFunction,
			}
			tableParent := &parent{
				Type: ParentTypeTable,
				Name: mi.Parent.String(),
			}
			return cType, tableParent, nil
		}
		return []CompletionType{
			CompletionTypeColumn,
			CompletionTypeTable,
			CompletionTypeAlias,
			CompletionTypeView,
			CompletionTypeSubQueryColumn,
			CompletionTypeSubQueryView,
			CompletionTypeFunction,
			CompletionTypeKeyword,
		}, noneParent, nil
	case nodeWalker.PrevNodesIs(true, genKeywordMatcher([]string{"JOIN", "COPY", "FROM", "DELETE FROM", "UPDATE", "INSERT INTO", "DESCRIBE", "TRUNCATE", "DESC", "EXPLAIN", "AND", "OR", "XOR"})):
		return []CompletionType{
			CompletionTypeColumn,
			CompletionTypeTable,
			CompletionTypeView,
			CompletionTypeSubQueryColumn,
			CompletionTypeSubQueryView,
			CompletionTypeFunction,
			CompletionTypeKeyword,
		}, noneParent, nil
	case nodeWalker.CurNodeIs(genTokenMatcher([]token.Kind{token.LParen})) || nodeWalker.PrevNodesIs(true, genTokenMatcher([]token.Kind{token.LParen})):
		// for insert columns
		return []CompletionType{
			CompletionTypeColumn,
			CompletionTypeTable,
			CompletionTypeView,
		}, noneParent, nil
	// case nodeWalker.PrevNodesIs(true, genKeywordMatcher([]string{"USE", "DATABASE", "TEMPLATE", "CONNECT"})):
	// 	res = []CompletionType{
	// 		CompletionTypeDatabase,
	// 	}
	default:
		return []CompletionType{
			CompletionTypeKeyword,
		}, noneParent, nil
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

func (c *Completer) keywordCandidates() []lsp.CompletionItem {
	candidates := []lsp.CompletionItem{}
	for _, k := range keywords {
		candidate := lsp.CompletionItem{
			Label:  k,
			Kind:   lsp.KeywordCompletion,
			Detail: "keyword",
		}
		candidates = append(candidates, candidate)
	}
	return candidates
}

func (c *Completer) columnCandidates(targetTables []*parseutil.TableInfo, pare *parent) []lsp.CompletionItem {
	candidates := []lsp.CompletionItem{}

	switch pare.Type {
	case ParentTypeNone:
		for _, table := range targetTables {
			if table.Name == "" {
				continue
			}
			if c.DBCache == nil {
				continue
			}
			columns, ok := c.DBCache.ColumnDescs(table.Name)
			if !ok {
				continue
			}
			for _, column := range columns {
				detail := strings.Join(
					[]string{
						"column",
						"`" + table.Name + "`",
						"(" + column.OnelineDesc() + ")",
					},
					" ",
				)
				candidate := lsp.CompletionItem{
					Label:  column.Name,
					Kind:   lsp.FieldCompletion,
					Detail: detail,
				}
				candidates = append(candidates, candidate)
			}
		}
	case ParentTypeSchema:
	case ParentTypeTable:
		for _, table := range targetTables {
			if table.Name != pare.Name && table.Alias != pare.Name {
				continue
			}
			if c.DBCache == nil {
				continue
			}
			columns, ok := c.DBCache.ColumnDescs(table.Name)
			if !ok {
				continue
			}
			for _, column := range columns {
				detail := strings.Join(
					[]string{
						"column",
						"`" + table.Name + "`",
						"(" + column.OnelineDesc() + ")",
					},
					" ",
				)
				candidate := lsp.CompletionItem{
					Label:  column.Name,
					Kind:   lsp.FieldCompletion,
					Detail: detail,
				}
				candidates = append(candidates, candidate)
			}
		}
	}
	return candidates
}

func (c *Completer) TableCandidates() []lsp.CompletionItem {
	candidates := []lsp.CompletionItem{}
	if c.DBCache == nil {
		return candidates
	}
	tables := c.DBCache.SortedTables()
	for _, tableName := range tables {
		candidate := lsp.CompletionItem{
			Label:  tableName,
			Kind:   lsp.FieldCompletion,
			Detail: "table",
		}
		candidates = append(candidates, candidate)
	}
	return candidates
}

func (c *Completer) aliasCandidates(targetTables []*parseutil.TableInfo) []lsp.CompletionItem {
	candidates := []lsp.CompletionItem{}
	for _, table := range targetTables {
		if table.Alias == "" {
			continue
		}
		detail := strings.Join(
			[]string{
				"alias",
				"`" + table.Name + "`",
			},
			" ",
		)
		candidate := lsp.CompletionItem{
			Label:  table.Alias,
			Kind:   lsp.FieldCompletion,
			Detail: detail,
		}
		candidates = append(candidates, candidate)
	}
	return candidates
}

func (c *Completer) SubQueryColumnCandidates(info *parseutil.SubQueryInfo) []lsp.CompletionItem {
	candidates := []lsp.CompletionItem{}
	for _, view := range info.Views {
		for _, colmun := range view.Columns {
			candidate := lsp.CompletionItem{
				Label:  colmun,
				Kind:   lsp.FieldCompletion,
				Detail: "subQuery",
			}
			candidates = append(candidates, candidate)
		}
	}
	return candidates
}

func (c *Completer) DatabaseCandidates() []lsp.CompletionItem {
	candidates := []lsp.CompletionItem{}
	for _, databaseName := range c.DBCache.SortedDatabases() {
		candidate := lsp.CompletionItem{
			Label:  databaseName,
			Kind:   lsp.FieldCompletion,
			Detail: "Database",
		}
		candidates = append(candidates, candidate)

	}
	return candidates
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
