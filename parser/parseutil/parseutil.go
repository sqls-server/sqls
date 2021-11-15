package parseutil

import (
	"fmt"

	"github.com/lighttiger2505/sqls/ast"
	"github.com/lighttiger2505/sqls/ast/astutil"
	"github.com/lighttiger2505/sqls/token"
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
	if ti.SubQueryColumns != nil && len(ti.SubQueryColumns) > 0 {
		return true
	}
	return false
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
		return nil, fmt.Errorf("Not found statement, Node: %q, Position: (%d, %d)", parsed.String(), pos.Line, pos.Col)
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
	parenthesis := nodeWalker.CurNodeButtomMatched(matcher)
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
	parenthesis := nodeWalker.CurNodeButtomMatched(matcher)
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
	stmt, err := extractFocusedStatement(parsed, pos)
	if err != nil {
		return nil, err
	}
	list := stmt
	if encloseIsSubQuery(stmt, pos) {
		list = extractFocusedSubQuery(stmt, pos)
	}
	tables, err := extractTableIdentifier(list, false)
	if err != nil {
		return nil, err
	}

	tableMap := map[string]*TableInfo{}
	for _, table := range tables {
		tableMap[table.DatabaseSchema+"\t"+table.Name] = table
	}
	cleanTables := []*TableInfo{}
	for _, table := range tableMap {
		cleanTables = append(cleanTables, table)
	}

	return cleanTables, nil
}

var identifierMatcher = astutil.NodeMatcher{
	NodeTypes: []ast.NodeType{
		ast.TypeIdentifer,
		ast.TypeIdentiferList,
		ast.TypeMemberIdentifer,
		ast.TypeAliased,
	},
}

func extractSubQueryColumns(selectStmt ast.TokenList) ([]*SubQueryColumn, []*TableInfo, error) {
	tables, err := extractTableIdentifier(selectStmt, true)
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

func extractTableIdentifier(list ast.TokenList, isSubQuery bool) ([]*TableInfo, error) {
	nodes := []ast.Node{}
	nodes = append(nodes, ExtractTableReferences(list)...)
	nodes = append(nodes, ExtractTableReference(list)...)
	nodes = append(nodes, ExtractTableFactor(list)...)
	res := []*TableInfo{}
	for _, ident := range nodes {
		if !isSubQuery && isSubQueryByNode(ident) {
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
	case *ast.Identifer:
		ti := &TableInfo{Name: v.NoQuateString()}
		res = append(res, ti)
	case *ast.IdentiferList:
		tis, err := identifierListToTableInfo(v)
		if err != nil {
			return nil, err
		}
		res = append(res, tis...)
	case *ast.MemberIdentifer:
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

func identifierListToTableInfo(il *ast.IdentiferList) ([]*TableInfo, error) {
	tis := []*TableInfo{}
	for _, ident := range il.GetIdentifers() {
		switch v := ident.(type) {
		case *ast.Identifer:
			ti := &TableInfo{
				Name: v.NoQuateString(),
			}
			tis = append(tis, ti)
		case *ast.MemberIdentifer:
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
	case *ast.Identifer:
		ti.Name = v.NoQuateString()
	case *ast.MemberIdentifer:
		ti.DatabaseSchema = v.Parent.String()
		ti.Name = v.GetChild().String()
	case *ast.Parenthesis:
		tables, err := extractTableIdentifier(v.Inner(), true)
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
	case *ast.Identifer:
		ti.Alias = v.NoQuateString()
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
	case *ast.Identifer:
		ident := v.NoQuateString()
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
	case *ast.IdentiferList:
		for _, ident := range v.GetIdentifers() {
			resSubqueryCols, err := parseSubQueryColumns(ident, tables)
			if err != nil {
				return nil, err
			}
			subqueryCols = append(subqueryCols, resSubqueryCols...)
		}
	case *ast.MemberIdentifer:
		subqueryCols = append(
			subqueryCols,
			&SubQueryColumn{
				ParentName: v.GetParentIdent().NoQuateString(),
				ColumnName: v.GetChildIdent().NoQuateString(),
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
	aliasedName := aliased.GetAliasedNameIdent().NoQuateString()
	switch v := aliased.RealName.(type) {
	case *ast.Identifer:
		subqueryCol := &SubQueryColumn{
			ColumnName: v.NoQuateString(),
			AliasName:  aliasedName,
		}
		return subqueryCol, nil
	case *ast.MemberIdentifer:
		subqueryCol := &SubQueryColumn{
			ParentName: v.GetParentIdent().NoQuateString(),
			ColumnName: v.GetChildIdent().NoQuateString(),
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
