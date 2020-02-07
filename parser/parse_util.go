package parser

import (
	"fmt"

	"github.com/lighttiger2505/sqls/ast"
	"github.com/lighttiger2505/sqls/ast/astutil"
	"golang.org/x/xerrors"
)

type TableInfo struct {
	DatabaseSchema string
	Name           string
	Alias          string
}

func ExtractTable(stmt ast.TokenList) []*TableInfo {
	list := stmt.GetTokens()[0].(ast.TokenList)
	fromJoinExpr := filterTokenList(newNodeReader(list), fromJoinMatcher)
	identifiers := filterTokenList(newNodeReader(fromJoinExpr), identifierMatcher)

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

func filterTokenList(reader *nodeReader, matcher astutil.NodeMatcher) ast.TokenList {
	var res []ast.Node
	for reader.nextNode(false) {
		if reader.curNodeIs(matcher) {
			res = append(res, reader.curNode)
		} else if list, ok := reader.curNode.(ast.TokenList); ok {
			newReader := newNodeReader(list)
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
		ti := &TableInfo{
			DatabaseSchema: v.Parent.String(),
			Name:           v.Child.String(),
		}
		res = append(res, ti)
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

// func PathEnclosingInterval() {
// }
