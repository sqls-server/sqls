package parser

import (
	"fmt"

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

func ExtractTable(stmt ast.TokenList) []*TableInfo {
	list := stmt.GetTokens()[0].(ast.TokenList)
	fromJoinExpr := filterTokenList(astutil.NewNodeReader(list), fromJoinMatcher)
	identifiers := filterTokenList(astutil.NewNodeReader(fromJoinExpr), identifierMatcher)

	res := []*TableInfo{}
	for _, ident := range identifiers.GetTokens() {
		infos, err := parseTableInfo(ident)
		if err != nil {
			// FIXME error tracking
			return res
		}
		res = append(res, infos...)
	}
	return res
}

var fromJoinMatcher = astutil.NodeMatcher{
	NodeTypeMatcherFunc: func(node interface{}) bool {
		if _, ok := node.(*ast.FromClause); ok {
			return true
		}
		if _, ok := node.(*ast.JoinClause); ok {
			return true
		}
		return false
	},
}

var identifierMatcher = astutil.NodeMatcher{
	NodeTypeMatcherFunc: func(node interface{}) bool {
		if _, ok := node.(*ast.Identifer); ok {
			return true
		}
		if _, ok := node.(*ast.IdentiferList); ok {
			return true
		}
		if _, ok := node.(*ast.MemberIdentifer); ok {
			return true
		}
		if _, ok := node.(*ast.Aliased); ok {
			return true
		}
		return false
	},
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
		res = append(res, identifierListToTableInfo(v)...)
	case *ast.MemberIdentifer:
		if v.Parent != nil {
			ti := &TableInfo{
				DatabaseSchema: v.Parent.String(),
				Name:           v.Child.String(),
			}
			res = append(res, ti)
		}
	case *ast.Aliased:
		res = append(res, aliasedToTableInfo(v))
	default:
		return nil, xerrors.Errorf("unknown node type %T", v)
	}
	return res, nil
}

func identifierListToTableInfo(il *ast.IdentiferList) []*TableInfo {
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
			// FIXME add error tracking
			panic(fmt.Sprintf("unknown node type %T", v))
		}
	}
	return tis
}

func aliasedToTableInfo(aliased *ast.Aliased) *TableInfo {
	ti := &TableInfo{}
	switch v := aliased.RealName.(type) {
	case *ast.Identifer:
		ti.Name = v.String()
	case *ast.MemberIdentifer:
		ti.DatabaseSchema = v.Parent.String()
		ti.Name = v.Child.String()
	default:
		// FIXME add error tracking
		panic(fmt.Sprintf("unknown node type, want Identifer or MemberIdentifier, got %T", v))
	}

	switch v := aliased.AliasedName.(type) {
	case *ast.Identifer:
		ti.Alias = v.String()
	default:
		// FIXME add error tracking
		panic(fmt.Sprintf("unknown node type, want Identifer, got %T", v))
	}
	return ti
}

type NodeWalker struct {
	Paths   []*astutil.NodeReader
	CurPath *astutil.NodeReader
	Index   int
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

func (nw *NodeWalker) CurNodeMatched(matcher astutil.NodeMatcher) ast.Node {
	for _, reader := range nw.Paths {
		if reader.CurNodeIs(matcher) {
			return reader.CurNode
		}
	}
	return nil
}

func (nw *NodeWalker) PrevNodesIs(ignoreWitespace bool, matcher astutil.NodeMatcher) bool {
	for _, reader := range nw.Paths {
		if reader.PrevNodeIs(ignoreWitespace, matcher) {
			return true
		}
	}
	return false
}
