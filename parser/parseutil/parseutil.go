package parseutil

import (
	"fmt"

	"github.com/sqls-server/sqls/ast"
	"github.com/sqls-server/sqls/ast/astutil"
	"github.com/sqls-server/sqls/token"
)

type TableInfo struct {
	DatabaseSchema  string
	Name            string
	Alias           string
	SubQueryColumns []*SubQueryColumn
}

func (ti *TableInfo) isMatchTableName(name string) bool {
	if ti.Name == name {
		return true
	}
	if ti.Alias == name {
		return true
	}
	return false
}

func (ti *TableInfo) hasSubQuery() bool {
	return len(ti.SubQueryColumns) > 0
}

type SubQueryInfo struct {
	Name  string
	Views []*SubQueryView
}

type SubQueryView struct {
	SubQueryColumns []*SubQueryColumn
}

type SubQueryColumn struct {
	ParentTable *TableInfo
	ParentName  string
	ColumnName  string
	AliasName   string
}

func (sc *SubQueryColumn) DisplayName() string {
	name := sc.ColumnName
	if sc.AliasName != "" {
		name = sc.AliasName
	}
	return name
}

func extractFocusedStatement(parsed ast.TokenList, pos token.Pos) (ast.TokenList, error) {
	nodeWalker := NewNodeWalker(parsed, pos)
	matcher := astutil.NodeMatcher{NodeTypes: []ast.NodeType{ast.TypeStatement}}
	if !nodeWalker.CurNodeIs(matcher) {
		return nil, fmt.Errorf("not found statement, Node: %q, Position: (%d, %d)", parsed.String(), pos.Line, pos.Col)
	}
	stmt := nodeWalker.CurNodeTopMatched(matcher).(ast.TokenList)
	return stmt, nil
}

func encloseIsSubQuery(stmt ast.TokenList, pos token.Pos) bool {
	nodeWalker := NewNodeWalker(stmt, pos)
	matcher := astutil.NodeMatcher{NodeTypes: []ast.NodeType{ast.TypeParenthesis}}
	if !nodeWalker.CurNodeIs(matcher) {
		return false
	}
	parenthesis := nodeWalker.CurNodeBottomMatched(matcher)
	tokenList, ok := parenthesis.(ast.TokenList)
	if !ok {
		return false
	}
	return isSubQuery(tokenList)
}

func isSubQuery(tokenList ast.TokenList) bool {
	reader := astutil.NewNodeReader(tokenList)
	if !reader.NextNode(false) {
		return false
	}
	if !reader.NextNode(false) {
		return false
	}
	if !reader.CurNodeIs(astutil.NodeMatcher{ExpectKeyword: []string{"SELECT"}}) {
		return false
	}
	return true
}

func isSubQueryByNode(node ast.Node) bool {
	alias, ok := node.(*ast.Aliased)
	if !ok {
		return false
	}
	list, ok := alias.RealName.(ast.TokenList)
	if !ok {
		return false
	}
	return isSubQuery(list)
}

func extractFocusedSubQuery(stmt ast.TokenList, pos token.Pos) ast.TokenList {
	nodeWalker := NewNodeWalker(stmt, pos)
	matcher := astutil.NodeMatcher{NodeTypes: []ast.NodeType{ast.TypeParenthesis}}
	if !nodeWalker.CurNodeIs(matcher) {
		return nil
	}
	parenthesis := nodeWalker.CurNodeBottomMatched(matcher)
	return parenthesis.(ast.TokenList)
}

func ExtractSubQueryViews(parsed ast.TokenList, pos token.Pos) ([]*SubQueryInfo, error) {
	stmt, err := extractFocusedStatement(parsed, pos)
	if err != nil {
		return nil, err
	}

	reader := astutil.NewNodeReader(parsed)
	matcher := astutil.NodeMatcher{NodeTypes: []ast.NodeType{ast.TypeAliased}}
	aliases := reader.FindRecursive(matcher)

	var subQueries []*ast.Aliased
	var before *ast.Aliased
	for _, alias := range aliases {
		if token.ComparePos(alias.Pos(), pos) < 0 {
			continue
		}
		if before != nil && token.ComparePos(alias.End(), before.End()) < 0 {
			continue
		}

		alias, ok := alias.(*ast.Aliased)
		if !ok {
			continue
		}
		list, ok := alias.RealName.(ast.TokenList)
		if !ok {
			continue
		}
		if isSubQuery(list) {
			subQueries = append(subQueries, alias)
			before = alias
		}
	}
	if len(subQueries) == 0 {
		return nil, nil
	}

	results := []*SubQueryInfo{}
	for _, subQuery := range subQueries {
		parenthesis, ok := subQuery.RealName.(*ast.Parenthesis)
		if !ok {
			return nil, fmt.Errorf("is not sub query, query: %q, type: %T", stmt, stmt)
		}

		subqueryCols, _, err := extractSubQueryColumns(parenthesis.Inner())
		if err != nil {
			return nil, err
		}

		cols := make([]string, len(subqueryCols))
		for i, subqueryCol := range subqueryCols {
			cols[i] = subqueryCol.ColumnName
		}

		info := &SubQueryInfo{
			Name: subQuery.AliasedName.String(),
			Views: []*SubQueryView{
				{
					SubQueryColumns: subqueryCols,
				},
			},
		}
		results = append(results, info)
	}
	return results, nil
}

func ExtractTable(parsed ast.TokenList, pos token.Pos) ([]*TableInfo, error) {
	return extractTables(parsed, pos, false)
}

func ExtractPrevTables(parsed ast.TokenList, pos token.Pos) ([]*TableInfo, error) {
	return extractTables(parsed, pos, true)
}

func ExtractLastTable(parsed ast.TokenList, pos token.Pos) (*TableInfo, error) {
	nodes := ExtractTableFactor(parsed)
	if len(nodes) == 0 {
		return nil, nil
	}

	var all []*TableInfo
	for _, ident := range nodes {
		p := ident.Pos()
		if token.ComparePos(p, pos) > 0 {
			continue
		}

		if isFollowedByOn(parsed, p) {
			continue
		}

		if isSubQueryByNode(ident) {
			continue
		}
		infos, err := parseTableInfo(ident)
		if err != nil {
			return nil, err
		}
		all = append(all, infos...)
	}
	l := len(all)
	var res *TableInfo
	if l != 0 {
		res = all[l-1]
	}
	return res, nil
}

func extractTables(parsed ast.TokenList, pos token.Pos, stopOnPos bool) ([]*TableInfo, error) {
	stmt, err := extractFocusedStatement(parsed, pos)
	if err != nil {
		return nil, err
	}
	list := stmt
	if encloseIsSubQuery(stmt, pos) {
		list = extractFocusedSubQuery(stmt, pos)
	}
	var stopPos *token.Pos
	if stopOnPos {
		stopPos = &pos
	}
	tables, err := extractTableIdentifier(list, false, stopPos)
	if err != nil {
		return nil, err
	}

	tableMap := map[string]*TableInfo{}
	cleanTables := []*TableInfo{}

	for _, table := range tables {
		tableKey := table.DatabaseSchema + "\t" + table.Name
		if _, ok := tableMap[tableKey]; !ok {
			tableMap[tableKey] = table
			cleanTables = append(cleanTables, table)
		}
	}
	return cleanTables, nil
}

func isFollowedByOn(parsed ast.TokenList, pos token.Pos) bool {
	nw := NewNodeWalker(parsed, pos)
	for _, n := range nw.Paths {
		if n.PeekNodeIs(true,
			astutil.NodeMatcher{
				NodeTypes: []ast.NodeType{ast.TypeAliased}}) {
			if !n.NextNode(true) {
				continue
			}
		}
		if n.PeekNodeIs(true, genKeywordMatcher([]string{"ON"})) {
			if !n.NextNode(true) {
				continue
			}
			if n.PeekNodeIs(true, astutil.NodeMatcher{
				NodeTypes: []ast.NodeType{ast.TypeComparison}}) {
				return true
			}
		}
	}
	return false
}

var identifierMatcher = astutil.NodeMatcher{
	NodeTypes: []ast.NodeType{
		ast.TypeIdentifier,
		ast.TypeIdentifierList,
		ast.TypeMemberIdentifier,
		ast.TypeAliased,
	},
}

func extractSubQueryColumns(selectStmt ast.TokenList) ([]*SubQueryColumn, []*TableInfo, error) {
	tables, err := extractAllTableIdentifiers(selectStmt, true)
	if err != nil {
		return nil, nil, err
	}

	// extract select identifiers
	identsObj := selectStmt.GetTokens()[2]
	cols, err := parseSubQueryColumns(identsObj, tables)
	if err != nil {
		return nil, nil, err
	}

	// check from clause is sub query
	fromIdentifier := ExtractTableReferences(selectStmt)
	if len(fromIdentifier) == 0 {
		return cols, tables, nil
	}
	alias, ok := fromIdentifier[0].(*ast.Aliased)
	if !ok {
		return cols, tables, nil
	}
	parenthesis, ok := alias.RealName.(*ast.Parenthesis)
	if !ok {
		return cols, tables, nil
	}
	if !isSubQuery(parenthesis) {
		return cols, tables, nil
	}

	// merge select identiner of inner sub query
	innerIdents, innerTables, err := extractSubQueryColumns(parenthesis.Inner())
	if err != nil {
		return nil, nil, err
	}
	realIdents := []*SubQueryColumn{}
	for _, ident := range cols {
		if ident.ColumnName == "*" {
			return innerIdents, innerTables, nil
		}
		for _, innerIdent := range innerIdents {
			if ident.ColumnName == innerIdent.ColumnName {
				ident.ParentTable.SubQueryColumns = innerIdents
				realIdents = append(realIdents, ident)
			}
		}
	}
	return realIdents, tables, nil
}

func extractTableIdentifier(list ast.TokenList, isSubQuery bool, stopPos *token.Pos) ([]*TableInfo, error) {
	nodes := []ast.Node{}
	nodes = append(nodes, ExtractTableReferences(list)...)
	nodes = append(nodes, ExtractTableReference(list)...)
	nodes = append(nodes, ExtractTableFactor(list)...)
	res := []*TableInfo{}
	for _, ident := range nodes {
		if !isSubQuery && isSubQueryByNode(ident) {
			continue
		}

		if stopPos != nil && token.ComparePos(ident.Pos(), *stopPos) > 0 {
			continue
		}

		infos, err := parseTableInfo(ident)
		if err != nil {
			return nil, err
		}
		res = append(res, infos...)
	}
	return res, nil
}

func extractAllTableIdentifiers(list ast.TokenList, isSubQuery bool) ([]*TableInfo, error) {
	return extractTableIdentifier(list, isSubQuery, nil)
}

func filterTokenList(reader *astutil.NodeReader, matcher astutil.NodeMatcher) ast.TokenList {
	var res []ast.Node
	for reader.NextNode(false) {
		if reader.CurNodeIs(matcher) {
			res = append(res, reader.CurNode)
		} else if list, ok := reader.CurNode.(ast.TokenList); ok {
			newReader := astutil.NewNodeReader(list)
			res = append(res, filterTokenList(newReader, matcher).GetTokens()...)
		}
	}
	return &ast.Statement{Toks: res}
}

func parseTableInfo(idents ast.Node) ([]*TableInfo, error) {
	res := []*TableInfo{}
	switch v := idents.(type) {
	case *ast.Identifier:
		ti := &TableInfo{Name: v.NoQuoteString()}
		res = append(res, ti)
	case *ast.IdentifierList:
		tis, err := identifierListToTableInfo(v)
		if err != nil {
			return nil, err
		}
		res = append(res, tis...)
	case *ast.MemberIdentifier:
		if v.Parent != nil {
			ti := &TableInfo{
				DatabaseSchema: v.Parent.String(),
				Name:           v.GetChild().String(),
			}
			res = append(res, ti)
		}
	case *ast.Aliased:
		tis, err := aliasedToTableInfo(v)
		if err != nil {
			return nil, err
		}
		res = append(res, tis)
	default:
		return nil, fmt.Errorf("unknown node type %T", v)
	}
	return res, nil
}

func identifierListToTableInfo(il *ast.IdentifierList) ([]*TableInfo, error) {
	tis := []*TableInfo{}
	for _, ident := range il.GetIdentifiers() {
		switch v := ident.(type) {
		case *ast.Identifier:
			ti := &TableInfo{
				Name: v.NoQuoteString(),
			}
			tis = append(tis, ti)
		case *ast.MemberIdentifier:
			ti := &TableInfo{
				DatabaseSchema: v.Parent.String(),
				Name:           v.GetChild().String(),
			}
			tis = append(tis, ti)
		case *ast.Aliased:
			// pass
		default:
			return nil, fmt.Errorf("failed parse table info, unknown node type %T, value %q in %q", ident, ident, il)
		}
	}
	return tis, nil
}

func aliasedToTableInfo(aliased *ast.Aliased) (*TableInfo, error) {
	ti := &TableInfo{}
	// fetch table schema and name
	switch v := aliased.RealName.(type) {
	case *ast.Identifier:
		ti.Name = v.NoQuoteString()
	case *ast.MemberIdentifier:
		ti.DatabaseSchema = v.Parent.String()
		ti.Name = v.GetChild().String()
	case *ast.Parenthesis:
		tables, err := extractAllTableIdentifiers(v.Inner(), true)
		if err != nil {
			panic(err)
		}
		ti.DatabaseSchema = tables[0].DatabaseSchema
		ti.Name = tables[0].Name
	default:
		return nil, fmt.Errorf(
			"failed parse real name of alias, unknown node type %T, value %q",
			aliased.RealName,
			aliased.RealName,
		)
	}

	// fetch table aliased name
	switch v := aliased.AliasedName.(type) {
	case *ast.Identifier:
		ti.Alias = v.NoQuoteString()
	default:
		return nil, fmt.Errorf(
			"failed parse aliased name of alias, unknown node type %T, value %q",
			aliased.AliasedName,
			aliased.AliasedName,
		)
	}
	return ti, nil
}

func parseSubQueryColumns(idents ast.Node, tables []*TableInfo) ([]*SubQueryColumn, error) {
	subqueryCols := []*SubQueryColumn{}
	switch v := idents.(type) {
	case *ast.Identifier:
		ident := v.NoQuoteString()
		if ident == "*" {
			for _, table := range tables {
				subqueryCol := &SubQueryColumn{
					ColumnName:  ident,
					ParentTable: table,
				}
				subqueryCols = append(subqueryCols, subqueryCol)
			}
		} else {
			subqueryCol := &SubQueryColumn{ColumnName: ident}
			if len(tables) == 1 {
				subqueryCol.ParentTable = tables[0]
			}
			subqueryCols = append(subqueryCols, subqueryCol)
		}
	case *ast.IdentifierList:
		for _, ident := range v.GetIdentifiers() {
			resSubqueryCols, err := parseSubQueryColumns(ident, tables)
			if err != nil {
				return nil, err
			}
			subqueryCols = append(subqueryCols, resSubqueryCols...)
		}
	case *ast.MemberIdentifier:
		subqueryCols = append(
			subqueryCols,
			&SubQueryColumn{
				ParentName: v.GetParentIdent().NoQuoteString(),
				ColumnName: v.GetChildIdent().NoQuoteString(),
			},
		)
	case *ast.Aliased:
		subqueryCol, err := aliasedToSubQueryColumn(v)
		if err != nil {
			return nil, err
		}
		subqueryCols = append(subqueryCols, subqueryCol)
	// TODO Add case of function:
	// case *ast.Function:
	default:
		return nil, fmt.Errorf("failed parse sub query columns, unknown node type %T, value %q", idents, idents)
	}

	for _, subqueryCol := range subqueryCols {
		for _, table := range tables {
			if table.isMatchTableName(subqueryCol.ParentName) {
				subqueryCol.ParentTable = table
			}
		}
	}
	return subqueryCols, nil
}

func aliasedToSubQueryColumn(aliased *ast.Aliased) (*SubQueryColumn, error) {
	// fetch table schema and name
	aliasedName := aliased.GetAliasedNameIdent().NoQuoteString()
	switch v := aliased.RealName.(type) {
	case *ast.Identifier:
		subqueryCol := &SubQueryColumn{
			ColumnName: v.NoQuoteString(),
			AliasName:  aliasedName,
		}
		return subqueryCol, nil
	case *ast.MemberIdentifier:
		subqueryCol := &SubQueryColumn{
			ParentName: v.GetParentIdent().NoQuoteString(),
			ColumnName: v.GetChildIdent().NoQuoteString(),
			AliasName:  aliasedName,
		}
		return subqueryCol, nil
	default:
		return nil, fmt.Errorf(
			"failed trans alias to column, unknown node type %T, value %q",
			aliased.AliasedName,
			aliased.AliasedName,
		)
	}
}
