package main

import (
	"bufio"
	"bytes"
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/lighttiger2505/sqls/ast"
	"github.com/lighttiger2505/sqls/ast/astutil"
	"github.com/lighttiger2505/sqls/database"
	"github.com/lighttiger2505/sqls/dialect"
	"github.com/lighttiger2505/sqls/parser"
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

type DatabaseInfo struct {
	Databases map[string]string
	Tables    map[string]string
	Columns   map[string][]*database.ColumnDesc
}

func (di *DatabaseInfo) Database(databaseName string) (db string, ok bool) {
	db, ok = di.Databases[strings.ToUpper(databaseName)]
	return
}

func (di *DatabaseInfo) SortedDatabases() []string {
	dbs := []string{}
	for _, db := range di.Databases {
		dbs = append(dbs, db)
	}
	sort.Strings(dbs)
	return dbs
}

func (di *DatabaseInfo) Table(databaseName string) (tbl string, ok bool) {
	tbl, ok = di.Tables[strings.ToUpper(databaseName)]
	return
}

func (di *DatabaseInfo) SortedTables() []string {
	tbls := []string{}
	for _, tbl := range di.Tables {
		tbls = append(tbls, tbl)
	}
	sort.Strings(tbls)
	return tbls
}

func (di *DatabaseInfo) ColumnDescs(databaseName string) (cols []*database.ColumnDesc, ok bool) {
	cols, ok = di.Columns[strings.ToUpper(databaseName)]
	return
}

type Completer struct {
	Conn   database.Database
	DBInfo *DatabaseInfo
}

func NewCompleter(db database.Database) *Completer {
	return &Completer{
		Conn:   db,
		DBInfo: &DatabaseInfo{},
	}
}

func (c *Completer) Init() error {
	if err := c.Conn.Open(); err != nil {
		return err
	}
	defer c.Conn.Close()

	dbs, err := c.Conn.Databases()
	if err != nil {
		return err
	}
	databaseMap := map[string]string{}
	for _, db := range dbs {
		databaseMap[strings.ToUpper(db)] = db
	}
	c.DBInfo.Databases = databaseMap

	tbls, err := c.Conn.Tables()
	if err != nil {
		return err
	}
	tableMap := map[string]string{}
	for _, tbl := range tbls {
		tableMap[strings.ToUpper(tbl)] = tbl
	}
	c.DBInfo.Tables = tableMap

	columnMap := map[string][]*database.ColumnDesc{}
	for _, tbl := range tbls {
		columnDescs, err := c.Conn.DescribeTable(tbl)
		if err != nil {
			return err
		}
		columnMap[strings.ToUpper(tbl)] = columnDescs
	}
	c.DBInfo.Columns = columnMap

	return nil
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

func (c *Completer) complete(text string, params CompletionParams) ([]CompletionItem, error) {
	parsed, err := parse(text)
	if err != nil {
		return nil, err
	}

	pos := token.Pos{Line: params.Position.Line + 1, Col: params.Position.Character}
	cTypes, pare, err := getCompletionTypes(text, pos)
	if err != nil {
		return nil, err
	}

	definedTables, err := parser.ExtractTable(parsed, pos)
	if err != nil {
		return nil, err
	}
	items := []CompletionItem{}
	if completionTypeIs(cTypes, CompletionTypeKeyword) {
		items = append(items, c.keywordCandidates()...)
	}
	if completionTypeIs(cTypes, CompletionTypeColumn) {
		items = append(items, c.columnCandidates(definedTables, pare)...)
	}
	if completionTypeIs(cTypes, CompletionTypeAlias) {
		items = append(items, c.aliasCandidates(definedTables)...)
	}
	if completionTypeIs(cTypes, CompletionTypeTable) {
		items = append(items, c.TableCandidates()...)
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
)

type parent struct {
	Type ParentType
	Name string
}

var noneParent = &parent{Type: ParentTypeNone}

var memberIdentifierMatcher = astutil.NodeMatcher{
	NodeTypeMatcherFunc: func(node interface{}) bool {
		if _, ok := node.(*ast.MemberIdentifer); ok {
			return true
		}
		return false
	},
}

func getCompletionTypes(text string, pos token.Pos) ([]CompletionType, *parent, error) {
	parsed, err := parse(text)
	if err != nil {
		return nil, nil, err
	}
	nodeWalker := parser.NewNodeWalker(parsed, pos)

	switch {
	case nodeWalker.PrevNodesIs(true, genKeywordMatcher([]string{"SET", "ORDER BY", "GROUP BY", "DISTINCT"})):
		if nodeWalker.CurNodeIs(memberIdentifierMatcher) {
			// has parent
			mi := nodeWalker.CurNodeMatched(memberIdentifierMatcher).(*ast.MemberIdentifer)
			cType := []CompletionType{
				CompletionTypeColumn,
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
			CompletionTypeAlias,
			CompletionTypeView,
			CompletionTypeFunction,
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
	case nodeWalker.PrevNodesIs(true, genKeywordMatcher([]string{"SELECT", "WHERE", "HAVING", "ON"})):
		if nodeWalker.CurNodeIs(memberIdentifierMatcher) {
			// has parent
			mi := nodeWalker.CurNodeMatched(memberIdentifierMatcher).(*ast.MemberIdentifer)
			cType := []CompletionType{
				CompletionTypeColumn,
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
			CompletionTypeAlias,
			CompletionTypeView,
			CompletionTypeFunction,
		}, noneParent, nil
	case nodeWalker.PrevNodesIs(true, genKeywordMatcher([]string{"JOIN", "COPY", "FROM", "UPDATE", "INTO", "DESCRIBE", "TRUNCATE", "DESC", "EXPLAIN"})):
		return []CompletionType{
			CompletionTypeColumn,
			CompletionTypeTable,
			CompletionTypeView,
			CompletionTypeFunction,
		}, noneParent, nil
	// case nodeWalker.PrevNodesIs(true, genKeywordMatcher([]string{"ON"})):
	// 	res = []CompletionType{
	// 		CompletionTypeColumn,
	// 		CompletionTypeTable,
	// 		CompletionTypeView,
	// 		CompletionTypeFunction,
	// 	}
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

func genKeywordMatcher(keywords []string) astutil.NodeMatcher {
	return astutil.NodeMatcher{
		ExpectKeyword: keywords,
	}
}

func filterCandidates(candidates []CompletionItem, lastWord string) []CompletionItem {
	filterd := []CompletionItem{}
	for _, candidate := range candidates {
		if strings.HasPrefix(strings.ToUpper(candidate.Label), strings.ToUpper(lastWord)) {
			filterd = append(filterd, candidate)
		}
	}
	return filterd
}

func (c *Completer) keywordCandidates() []CompletionItem {
	candidates := []CompletionItem{}
	for _, k := range keywords {
		candidate := CompletionItem{
			Label:  k,
			Kind:   KeywordCompletion,
			Detail: "Keyword",
		}
		candidates = append(candidates, candidate)
	}
	return candidates
}

var ColumnDetailTemplate = "Column"

func (c *Completer) columnCandidates(targetTables []*parser.TableInfo, pare *parent) []CompletionItem {
	candidates := []CompletionItem{}

	switch pare.Type {
	case ParentTypeNone:
		for _, info := range targetTables {
			if info.Name == "" {
				continue
			}
			if c.DBInfo == nil {
				continue
			}
			columns, ok := c.DBInfo.ColumnDescs(info.Name)
			if !ok {
				continue
			}
			for _, column := range columns {
				candidate := CompletionItem{
					Label:  column.Name,
					Kind:   FieldCompletion,
					Detail: ColumnDetailTemplate,
				}
				candidates = append(candidates, candidate)
			}
		}
	case ParentTypeSchema:
	case ParentTypeTable:
		for _, info := range targetTables {
			if info.Name != pare.Name && info.Alias != pare.Name {
				continue
			}
			if c.DBInfo == nil {
				continue
			}
			columns, ok := c.DBInfo.ColumnDescs(info.Name)
			if !ok {
				continue
			}
			for _, column := range columns {
				candidate := CompletionItem{
					Label:  column.Name,
					Kind:   FieldCompletion,
					Detail: ColumnDetailTemplate,
				}
				candidates = append(candidates, candidate)
			}
		}
	}
	return candidates
}

var TableDetailTemplate = "Table"

func (c *Completer) TableCandidates() []CompletionItem {
	candidates := []CompletionItem{}
	if c.DBInfo == nil {
		return candidates
	}
	tables := c.DBInfo.SortedTables()
	for _, tableName := range tables {
		candidate := CompletionItem{
			Label:  tableName,
			Kind:   FieldCompletion,
			Detail: TableDetailTemplate,
		}
		candidates = append(candidates, candidate)
	}
	return candidates
}

var AliasDetailTemplate = "Alias"

func (c *Completer) aliasCandidates(targetTables []*parser.TableInfo) []CompletionItem {
	candidates := []CompletionItem{}
	for _, info := range targetTables {
		if info.Alias == "" {
			continue
		}
		candidate := CompletionItem{
			Label:  info.Alias,
			Kind:   FieldCompletion,
			Detail: AliasDetailTemplate,
		}
		candidates = append(candidates, candidate)
	}
	return candidates
}

func (c *Completer) DatabaseCandidates() []CompletionItem {
	candidates := []CompletionItem{}
	for _, databaseName := range c.DBInfo.SortedDatabases() {
		candidate := CompletionItem{
			Label:  databaseName,
			Kind:   FieldCompletion,
			Detail: "Database",
		}
		candidates = append(candidates, candidate)

	}
	return candidates
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
