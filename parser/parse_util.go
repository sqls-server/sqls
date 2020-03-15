package parser

import (
	"github.com/lighttiger2505/sqls/ast"
	"github.com/lighttiger2505/sqls/ast/astutil"
	"github.com/lighttiger2505/sqls/token"
	"golang.org/x/xerrors"
)

type TableInfo struct {
	DatabaseSchema string
	Name           string
	Alias          string
}

type SubQueryInfo struct {
	Name  string
	Views []*SubQueryView
}

type SubQueryView struct {
	Table   *TableInfo
	Columns []string
}

func extractFocusedStatement(parsed ast.TokenList, pos token.Pos) (ast.TokenList, error) {
	nodeWalker := NewNodeWalker(parsed, pos)
	matcher := astutil.NodeMatcher{NodeTypes: []ast.NodeType{ast.TypeStatement}}
	if !nodeWalker.CurNodeIs(matcher) {
		return nil, xerrors.Errorf("Not found statement, Node: %q, Position: (%d, %d)", parsed.String(), pos.Line, pos.Col)
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

func extractFocusedSubQuery(stmt ast.TokenList, pos token.Pos) ast.TokenList {
	nodeWalker := NewNodeWalker(stmt, pos)
	matcher := astutil.NodeMatcher{NodeTypes: []ast.NodeType{ast.TypeParenthesis}}
	if !nodeWalker.CurNodeIs(matcher) {
		return nil
	}
	parenthesis := nodeWalker.CurNodeButtomMatched(matcher)
	return parenthesis.(ast.TokenList)
}

func ExtractSubQueryView(parsed ast.TokenList, pos token.Pos) (*SubQueryInfo, error) {
	stmt, err := extractFocusedStatement(parsed, pos)
	if err != nil {
		return nil, err
	}

	var firstSubQuery *ast.Aliased
	reader := astutil.NewNodeReader(parsed)
	matcher := astutil.NodeMatcher{NodeTypes: []ast.NodeType{ast.TypeAliased}}
	aliases := reader.FindRecursive(matcher)
	for _, node := range aliases {
		if token.ComparePos(node.Pos(), pos) <= 0 {
			continue
		}
		alias, ok := node.(*ast.Aliased)
		if !ok {
			continue
		}
		list, ok := alias.RealName.(ast.TokenList)
		if !ok {
			continue
		}
		if isSubQuery(list) {
			firstSubQuery = alias
			break
		}
	}
	if firstSubQuery == nil {
		return &SubQueryInfo{}, nil
	}

	parenthesis, ok := firstSubQuery.RealName.(*ast.Parenthesis)
	if !ok {
		return nil, xerrors.Errorf("is not sub query, query: %q, type: %T", stmt, stmt)
	}

	idents, err := extractSelectIdentifier(parenthesis.Inner())
	if err != nil {
		return nil, err
	}

	tables, err := extractTableIdentifier(parenthesis.Inner())
	if err != nil {
		return nil, err
	}

	return &SubQueryInfo{
		Name: firstSubQuery.AliasedName.String(),
		Views: []*SubQueryView{
			&SubQueryView{
				Table:   tables[0],
				Columns: idents,
			},
		},
	}, nil
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
	return extractTableIdentifier(list)
}

var fromJoinMatcher = astutil.NodeMatcher{
	NodeTypes: []ast.NodeType{
		ast.TypeFromClause,
		ast.TypeJoinClause,
	},
}

var identifierMatcher = astutil.NodeMatcher{
	NodeTypes: []ast.NodeType{
		ast.TypeIdentifer,
		ast.TypeIdentiferList,
		ast.TypeMemberIdentifer,
		ast.TypeAliased,
	},
}

var fromMatcher = astutil.NodeMatcher{
	NodeTypes: []ast.NodeType{
		ast.TypeFromClause,
	},
}

func extractSelectIdentifier(selectStmt ast.TokenList) ([]string, error) {
	// extract select identifiers
	idents := []string{}
	identsObj := selectStmt.GetTokens()[2]
	switch v := identsObj.(type) {
	case ast.TokenList:
		identifiers := filterTokenList(astutil.NewNodeReader(v), identifierMatcher)
		for _, ident := range identifiers.GetTokens() {
			res, err := parseSubQueryColumns(ident)
			if err != nil {
				return nil, err
			}
			idents = append(idents, res...)
		}
	case *ast.Identifer:
		res, err := parseSubQueryColumns(v)
		if err != nil {
			return nil, err
		}
		idents = append(idents, res...)
	default:
		return nil, xerrors.Errorf("failed read the TokenList of select, query: %q, type: %T", identsObj, identsObj)
	}

	// check from clause is sub query
	from := filterTokenList(astutil.NewNodeReader(selectStmt), fromMatcher).GetTokens()[0]
	list, ok := from.(ast.TokenList)
	if !ok {
		return idents, nil
	}
	fromIdents := list.GetTokens()[2]
	alias, ok := fromIdents.(*ast.Aliased)
	if !ok {
		return idents, nil
	}
	parenthesis, ok := alias.RealName.(*ast.Parenthesis)
	if !ok {
		return idents, nil
	}
	if !isSubQuery(parenthesis) {
		return idents, nil
	}

	// merge select identiner of inner sub query
	innerIdents, err := extractSelectIdentifier(parenthesis.Inner())
	if err != nil {
		return nil, err
	}
	realIdents := []string{}
	for _, ident := range idents {
		if ident == "*" {
			return innerIdents, nil
		}
		for _, innerIdent := range innerIdents {
			if ident == innerIdent {
				realIdents = append(realIdents, ident)
			}
		}
	}
	return realIdents, nil
}

func extractTableIdentifier(list ast.TokenList) ([]*TableInfo, error) {
	fromJoinExpr := filterTokenList(astutil.NewNodeReader(list), fromJoinMatcher)
	identifiers := filterTokenList(astutil.NewNodeReader(fromJoinExpr), identifierMatcher)
	res := []*TableInfo{}
	for _, ident := range identifiers.GetTokens() {
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

func filterTokens(toks []ast.Node, matcher astutil.NodeMatcher) []ast.Node {
	res := []ast.Node{}
	for _, tok := range toks {
		if matcher.IsMatch(tok) {
			res = append(res, tok)
		}
	}
	return res
}

func parseTableInfo(idents ast.Node) ([]*TableInfo, error) {
	res := []*TableInfo{}
	switch v := idents.(type) {
	case *ast.Identifer:
		ti := &TableInfo{Name: v.String()}
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
				Name:           v.Child.String(),
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
		return nil, xerrors.Errorf("unknown node type %T", v)
	}
	return res, nil
}

func identifierListToTableInfo(il *ast.IdentiferList) ([]*TableInfo, error) {
	tis := []*TableInfo{}
	idents := filterTokens(il.GetTokens(), identifierMatcher)
	for _, ident := range idents {
		switch v := ident.(type) {
		case *ast.Identifer:
			ti := &TableInfo{
				Name: v.String(),
			}
			tis = append(tis, ti)
		case *ast.MemberIdentifer:
			ti := &TableInfo{
				DatabaseSchema: v.Parent.String(),
				Name:           v.Child.String(),
			}
			tis = append(tis, ti)
		default:
			return nil, xerrors.Errorf("failed parse table info, unknown node type %T, value %q in %q", ident, ident, il)
		}
	}
	return tis, nil
}

func aliasedToTableInfo(aliased *ast.Aliased) (*TableInfo, error) {
	ti := &TableInfo{}
	// fetch table schema and name
	switch v := aliased.RealName.(type) {
	case *ast.Identifer:
		ti.Name = v.String()
	case *ast.MemberIdentifer:
		ti.DatabaseSchema = v.Parent.String()
		ti.Name = v.Child.String()
	case *ast.Parenthesis:
		tables, err := extractTableIdentifier(v.Inner())
		if err != nil {
			panic(err)
		}
		ti.DatabaseSchema = tables[0].DatabaseSchema
		ti.Name = tables[0].Name
	default:
		return nil, xerrors.Errorf(
			"failed parse real name of alias, unknown node type %T, value %q",
			aliased.RealName,
			aliased.RealName,
		)
	}

	// fetch table aliased name
	switch v := aliased.AliasedName.(type) {
	case *ast.Identifer:
		ti.Alias = v.String()
	default:
		return nil, xerrors.Errorf(
			"failed parse aliased name of alias, unknown node type %T, value %q",
			aliased.AliasedName,
			aliased.AliasedName,
		)
	}
	return ti, nil
}

func parseSubQueryColumns(idents ast.Node) ([]string, error) {
	columns := []string{}
	switch v := idents.(type) {
	case *ast.Identifer:
		columns = append(columns, v.String())
	case *ast.IdentiferList:
		results, err := identifierListToSubQueryColumns(v)
		if err != nil {
			return nil, err
		}
		columns = append(columns, results...)
	case *ast.MemberIdentifer:
		columns = append(columns, v.Child.String())
	case *ast.Aliased:
		result, err := aliasedToSubQueryColumn(v)
		if err != nil {
			return nil, err
		}
		columns = append(columns, result)
	default:
		return nil, xerrors.Errorf("failed parse sub query columns, unknown node type %T, value %q", idents, idents)
	}
	return columns, nil
}

func identifierListToSubQueryColumns(il *ast.IdentiferList) ([]string, error) {
	columns := []string{}
	idents := filterTokens(il.GetTokens(), identifierMatcher)
	for _, ident := range idents {
		switch v := ident.(type) {
		case *ast.Identifer:
			columns = append(columns, v.String())
		case *ast.MemberIdentifer:
			columns = append(columns, v.Child.String())
		default:
			return nil, xerrors.Errorf(
				"failed trans identifier list to column, unknown node type %T, value %q",
				ident,
				ident,
			)
		}
	}
	return columns, nil
}

func aliasedToSubQueryColumn(aliased *ast.Aliased) (string, error) {
	// fetch table schema and name
	switch v := aliased.AliasedName.(type) {
	case *ast.Identifer:
		return v.String(), nil
	default:
		return "", xerrors.Errorf(
			"failed trans alias to column, unknown node type %T, value %q",
			aliased.AliasedName,
			aliased.AliasedName,
		)
	}
}

type NodeWalker struct {
	Paths []*astutil.NodeReader
	Index int
}

func astPaths(reader *astutil.NodeReader, pos token.Pos) []*astutil.NodeReader {
	paths := []*astutil.NodeReader{}
	for reader.NextNode(false) {
		if reader.CurNodeEncloseIs(pos) {
			paths = append(paths, reader)
			if list, ok := reader.CurNode.(ast.TokenList); ok {
				newReader := astutil.NewNodeReader(list)
				return append(paths, astPaths(newReader, pos)...)
			} else {
				return paths
			}
		}
	}
	return paths
}

func NewNodeWalker(root ast.TokenList, pos token.Pos) *NodeWalker {
	return &NodeWalker{
		Paths: astPaths(astutil.NewNodeReader(root), pos),
	}
}

func (nw *NodeWalker) CurNodeIs(matcher astutil.NodeMatcher) bool {
	for _, reader := range nw.Paths {
		if reader.CurNodeIs(matcher) {
			return true
		}
	}
	return false
}

func (nw *NodeWalker) CurNodeMatches(matcher astutil.NodeMatcher) []ast.Node {
	matches := []ast.Node{}
	for _, reader := range nw.Paths {
		if reader.CurNodeIs(matcher) {
			matches = append(matches, reader.CurNode)
		}
	}
	return matches
}

func (nw *NodeWalker) CurNodeTopMatched(matcher astutil.NodeMatcher) ast.Node {
	matches := nw.CurNodeMatches(matcher)
	if len(matches) == 0 {
		return nil
	}
	return matches[0]
}

func (nw *NodeWalker) CurNodeButtomMatched(matcher astutil.NodeMatcher) ast.Node {
	matches := nw.CurNodeMatches(matcher)
	if len(matches) == 0 {
		return nil
	}
	return matches[len(matches)-1]
}

func (nw *NodeWalker) PrevNodesIs(ignoreWitespace bool, matcher astutil.NodeMatcher) bool {
	for _, reader := range nw.Paths {
		if reader.PrevNodeIs(ignoreWitespace, matcher) {
			return true
		}
	}
	return false
}
