package completer

import (
	"bufio"
	"bytes"
	"fmt"
	"regexp"
	"strings"

	"github.com/sqls-server/sqls/ast"
	"github.com/sqls-server/sqls/ast/astutil"
	"github.com/sqls-server/sqls/dialect"
	"github.com/sqls-server/sqls/internal/database"
	"github.com/sqls-server/sqls/internal/lsp"
	"github.com/sqls-server/sqls/parser"
	"github.com/sqls-server/sqls/parser/parseutil"
	"github.com/sqls-server/sqls/token"
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
	CompletionTypeJoin
	CompletionTypeJoinOn
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
	case CompletionTypeSubQuery:
		return "SubQuery"
	case CompletionTypeSubQueryColumn:
		return "SubQueryColumn"
	case CompletionTypeJoin:
		return "Join clause"
	case CompletionTypeJoinOn:
		return "Join On condition"
	default:
		return ""
	}
}

type Completer struct {
	DBCache *database.DBCache
	Driver  dialect.DatabaseDriver
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

func (c *Completer) Complete(text string, params lsp.CompletionParams, lowercaseKeywords bool) ([]lsp.CompletionItem, error) {
	parsed, err := parser.Parse(text)
	if err != nil {
		return nil, err
	}

	pos := token.Pos{
		Line: params.Position.Line,
		Col:  params.Position.Character,
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
	definedSubQueries, err := parseutil.ExtractSubQueryViews(parsed, pos)
	if err != nil {
		return nil, err
	}

	lastWord := getLastWord(text, params.Position.Line+1, params.Position.Character)
	withBackQuote := strings.HasPrefix(lastWord, "`")

	var items []lsp.CompletionItem

	if c.DBCache != nil {
		if completionTypeIs(ctx.types, CompletionTypeColumn) {
			candidates := c.columnCandidates(definedTables, ctx.parent)
			if withBackQuote {
				candidates = toQuotedCandidates(candidates)
			}
			items = append(items, candidates...)
		}
		if completionTypeIs(ctx.types, CompletionTypeReferencedTable) {
			candidates := c.ReferencedTableCandidates(definedTables)
			if withBackQuote {
				candidates = toQuotedCandidates(candidates)
			}
			items = append(items, candidates...)
		}
		if completionTypeIs(ctx.types, CompletionTypeTable) {
			excl := definedTables
			if completionTypeIs(ctx.types, CompletionTypeJoin) {
				excl = nil
			}
			candidates := c.TableCandidates(ctx.parent, excl)
			if withBackQuote {
				candidates = toQuotedCandidates(candidates)
			}
			items = append(items, candidates...)
		}
		if completionTypeIs(ctx.types, CompletionTypeSchema) {
			candidates := c.SchemaCandidates()
			if withBackQuote {
				candidates = toQuotedCandidates(candidates)
			}
			items = append(items, candidates...)
		}
		if completionTypeIs(ctx.types, CompletionTypeSubQuery) {
			candidates := c.SubQueryCandidates(definedSubQueries)
			if withBackQuote {
				candidates = toQuotedCandidates(candidates)
			}
			items = append(items, candidates...)
		}
		if completionTypeIs(ctx.types, CompletionTypeSubQueryColumn) {
			candidates := c.SubQueryColumnCandidates(definedSubQueries)
			if withBackQuote {
				candidates = toQuotedCandidates(candidates)
			}
			items = append(items, candidates...)
		}
		joinOn := completionTypeIs(ctx.types, CompletionTypeJoinOn)
		if completionTypeIs(ctx.types, CompletionTypeJoin) || joinOn {
			table, err := parseutil.ExtractLastTable(parsed, pos)
			if err != nil {
				return nil, err
			}
			tables, err := parseutil.ExtractPrevTables(parsed, pos)
			if err != nil {
				return nil, err
			}
			candidates := c.joinCandidates(table, tables, definedTables, joinOn, lowercaseKeywords)
			if withBackQuote {
				candidates = toQuotedCandidates(candidates) // what to do here?
			}
			items = append(candidates, items...)
		}
	}

	if completionTypeIs(ctx.types, CompletionTypeKeyword) {
		drivers := dialect.DataBaseKeywords(c.Driver)
		items = append(items, c.keywordCandidates(lowercaseKeywords, drivers)...)
	}
	if completionTypeIs(ctx.types, CompletionTypeFunction) {
		drivers := dialect.DataBaseFunctions(c.Driver)
		items = append(items, c.functionCandidates(lowercaseKeywords, drivers)...)
	}

	items = filterCandidates(items, lastWord)
	populateSortText(items)

	return items, nil
}

// Override the sort text for each completion item.
func populateSortText(items []lsp.CompletionItem) {
	for i := range items {
		items[i].SortText = getSortTextPrefix(items[i].Kind) + items[i].Label
	}
}

// Some completion kinds are more relevant than others.
// This prefix defines the alphabetic priority of each kind.
func getSortTextPrefix(kind lsp.CompletionItemKind) string {
	switch kind {
	case lsp.SnippetCompletion:
		return "00"
	case lsp.FieldCompletion:
		return "0"
	case lsp.ClassCompletion:
		return "1"
	case lsp.ModuleCompletion:
		return "2"
	case lsp.FunctionCompletion:
		return "10"
	case
		lsp.ColorCompletion,
		lsp.ConstantCompletion,
		lsp.ConstructorCompletion,
		lsp.EnumCompletion,
		lsp.EnumMemberCompletion,
		lsp.EventCompletion,
		lsp.FileCompletion,
		lsp.FolderCompletion,
		lsp.InterfaceCompletion,
		lsp.KeywordCompletion,
		lsp.MethodCompletion,
		lsp.OperatorCompletion,
		lsp.PropertyCompletion,
		lsp.ReferenceCompletion,
		lsp.StructCompletion,
		lsp.TextCompletion,
		lsp.TypeParameterCompletion,
		lsp.UnitCompletion,
		lsp.ValueCompletion,
		lsp.VariableCompletion:
		return "9999"
	default:
		return "9999"
	}
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
		NodeTypes: []ast.NodeType{ast.TypeMemberIdentifier},
	}

	syntaxPos := parseutil.CheckSyntaxPosition(nw)
	var t []completionType
	p := noneParent
	switch {
	case syntaxPos == parseutil.ColName:
		if nw.CurNodeIs(memberIdentifierMatcher) {
			// has parent
			mi := nw.CurNodeTopMatched(memberIdentifierMatcher).(*ast.MemberIdentifier)
			t = []completionType{
				CompletionTypeColumn,
				CompletionTypeSubQueryColumn,
				CompletionTypeView,
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
			}
			p = noneParent
		}
	case syntaxPos == parseutil.AliasName:
		// pass
	case syntaxPos == parseutil.SelectExpr || syntaxPos == parseutil.CaseValue:
		if nw.CurNodeIs(memberIdentifierMatcher) {
			// has parent
			mi := nw.CurNodeTopMatched(memberIdentifierMatcher).(*ast.MemberIdentifier)
			t = []completionType{
				CompletionTypeColumn,
				CompletionTypeView,
				CompletionTypeSubQueryColumn,
			}
			p = &completionParent{
				Type: ParentTypeTable,
				Name: mi.ParentTok.NoQuoteString(),
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
			}
		}
	case syntaxPos == parseutil.TableReference:
		if nw.CurNodeIs(memberIdentifierMatcher) {
			// has parent
			mi := nw.CurNodeTopMatched(memberIdentifierMatcher).(*ast.MemberIdentifier)
			t = []completionType{
				CompletionTypeTable,
				CompletionTypeView,
				CompletionTypeSubQueryColumn,
			}
			p = &completionParent{
				Type: ParentTypeSchema,
				Name: mi.ParentTok.NoQuoteString(),
			}
		} else {
			t = []completionType{
				CompletionTypeTable,
				CompletionTypeReferencedTable,
				CompletionTypeSchema,
				CompletionTypeView,
				CompletionTypeSubQuery,
			}
		}
	case syntaxPos == parseutil.WhereCondition:
		if nw.CurNodeIs(memberIdentifierMatcher) {
			// has parent
			mi := nw.CurNodeTopMatched(memberIdentifierMatcher).(*ast.MemberIdentifier)
			t = []completionType{
				CompletionTypeColumn,
				CompletionTypeView,
				CompletionTypeSubQueryColumn,
			}
			p = &completionParent{
				Type: ParentTypeTable,
				Name: mi.ParentTok.NoQuoteString(),
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
			}
		}
	case syntaxPos == parseutil.JoinClause:
		t = []completionType{
			CompletionTypeJoin,
			CompletionTypeTable,
			CompletionTypeReferencedTable,
			CompletionTypeSchema,
			CompletionTypeView,
			CompletionTypeSubQuery,
		}
	case syntaxPos == parseutil.JoinOn:
		t = []completionType{
			CompletionTypeJoinOn,
			CompletionTypeColumn,
			CompletionTypeReferencedTable,
			CompletionTypeSubQueryColumn,
			CompletionTypeSubQuery,
		}
	case syntaxPos == parseutil.InsertColumn:
		t = []completionType{
			CompletionTypeColumn,
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
	filtered := []lsp.CompletionItem{}
	for _, candidate := range candidates {
		if strings.HasPrefix(strings.ToUpper(candidate.Label), strings.ToUpper(lastWord)) {
			filtered = append(filtered, candidate)
		}
	}
	return filtered
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

	reg := regexp.MustCompile("[\\w`]+$")
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
		fmt.Fprintln(writer, scanner.Text())
		i++
	}
	return writer.String()
}

func toQuotedCandidates(candidates []lsp.CompletionItem) []lsp.CompletionItem {
	quotedCandidates := make([]lsp.CompletionItem, len(candidates))
	for i, candidate := range candidates {
		candidate.Label = fmt.Sprintf("`%s`", candidate.Label)
		quotedCandidates[i] = candidate
	}
	return quotedCandidates
}
