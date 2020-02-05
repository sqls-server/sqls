package parser

import (
	"fmt"

	"github.com/lighttiger2505/sqls/ast"
)

type TableInfo struct {
	DatabaseSchema string
	Name           string
	Alias          string
}

func ExtractTable(stmt ast.TokenList) []*TableInfo {
	list := stmt.GetTokens()[0].(ast.TokenList)
	fromJoinExpr := filterByMatcher(newNodeReader(list), fromJoinMatcher)
	fmt.Println(fromJoinExpr)
	identifiers := filterByMatcher(newNodeReader(fromJoinExpr), identifierMatcher)
	fmt.Println(identifiers)

	res := []*TableInfo{}
	for _, ident := range identifiers.GetTokens() {
		res = append(res, parseTableInfo(ident))
	}
	return res
}

var fromJoinMatcher = nodeMatcher{
	nodeTypeMatcherFunc: func(node interface{}) bool {
		if _, ok := node.(*ast.From); ok {
			return true
		}
		if _, ok := node.(*ast.Join); ok {
			return true
		}
		return false
	},
}

var identifierMatcher = nodeMatcher{
	nodeTypeMatcherFunc: func(node interface{}) bool {
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

func filterByMatcher(reader *nodeReader, matcher nodeMatcher) ast.TokenList {
	var res []ast.Node
	for reader.nextNode(false) {
		if reader.curNodeIs(matcher) {
			res = append(res, reader.curNode)
		} else if list, ok := reader.curNode.(ast.TokenList); ok {
			newReader := newNodeReader(list)
			res = append(res, filterByMatcher(newReader, matcher).GetTokens()...)
		}
	}
	return &ast.Statement{Toks: res}
}

func parseTableInfo(ident ast.Node) *TableInfo {
	res := &TableInfo{}
	switch v := ident.(type) {
	case *ast.Identifer:
		res.Name = v.String()
	case *ast.IdentiferList:
		res.Name = v.String()
	case *ast.MemberIdentifer:
		res.Name = v.String()
	default:
		panic(fmt.Sprintf("unknown node type %T", v))
	}
	return res
}

// func PathEnclosingInterval() {
// }
