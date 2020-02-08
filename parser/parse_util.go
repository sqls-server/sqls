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

func PathEnclosingInterval(root ast.Node, start, end token.Pos) (path []ast.Node, exact bool) {
	fmt.Printf("EnclosingInterval %d %d\n", start, end) // debugging

	// Precondition: node.[Pos..End) and adjoining whitespace contain [start, end).
	var visit func(node ast.Node) bool
	visit = func(node ast.Node) bool {
		path = append(path, node)

		nodePos := node.Pos()
		nodeEnd := node.End()

		fmt.Printf("visit(%T, %d, %d)\n", node, nodePos, nodeEnd) // debugging

		// Intersect [start, end) with interval of node.
		if 0 > token.ComparePos(start, nodePos) {
			start = nodePos
		}
		if 0 < token.ComparePos(end, nodeEnd) {
			end = nodeEnd
		}

		// Find sole child that contains [start, end).
		children := childrenOf(node)
		l := len(children)
		for i, child := range children {
			// [childPos, childEnd) is unaugmented interval of child.
			childPos := child.Pos()
			childEnd := child.End()

			// [augPos, augEnd) is whitespace-augmented interval of child.
			augPos := childPos
			augEnd := childEnd
			if i > 0 {
				augPos = children[i-1].End() // start of preceding whitespace
			}
			if i < l-1 {
				nextChildPos := children[i+1].Pos()
				// Does [start, end) lie between child and next child?
				if 0 <= token.ComparePos(start, augEnd) && 0 >= token.ComparePos(end, nextChildPos) {
					start = nodePos
				}
				// if start >= augEnd && end <= nextChildPos {
				// 	return false // inexact match
				// }
				augEnd = nextChildPos // end of following whitespace
			}

			fmt.Printf("\tchild %d: [%d..%d)\tcontains interval [%d..%d)?\n",
				i, augPos, augEnd, start, end) // debugging

			// Does augmented child strictly contain [start, end)?
			// equals augPos <= start && end <= augEnd {
			if 0 >= token.ComparePos(augPos, start) && 0 >= token.ComparePos(end, augEnd) {
				_, isToken := child.(ast.Node)
				return isToken || visit(child)
			}

			// Does [start, end) overlap multiple children?
			// i.e. left-augmented child contains start
			// but LR-augmented child does not contain end.

			// equals [start < childEnd && end > augEnd]
			if 0 > token.ComparePos(start, childEnd) && 0 < token.ComparePos(end, augEnd) {
				break
			}
		}

		// No single child contained [start, end),
		// so node is the result.  Is it exact?

		// (It's tempting to put this condition before the
		// child loop, but it gives the wrong result in the
		// case where a node (e.g. ExprStmt) and its sole
		// child have equal intervals.)
		if start == nodePos && end == nodeEnd {
			return true // exact match
		}

		return false // inexact: overlaps multiple children
	}

	// start > end
	if 0 < token.ComparePos(start, end) {
		start, end = end, start
	}

	// start < root.End() && end > root.Pos()
	if 0 > token.ComparePos(start, root.End()) && 0 < token.ComparePos(end, root.Pos()) {
		if start == end {
			end.Col = start.Col + 1 // empty interval => interval of size 1
		}
		exact = visit(root)

		// Reverse the path:
		for i, l := 0, len(path); i < l/2; i++ {
			path[i], path[l-1-i] = path[l-1-i], path[i]
		}
	} else {
		// Selection lies within whitespace preceding the
		// first (or following the last) declaration in the file.
		// The result nonetheless always includes the ast.File.
		path = append(path, root)
	}

	return
}

// childrenOf returns the direct non-nil children of ast.Node n.
// It may include fake ast.Node implementations for bare tokens.
// it is not safe to call (e.g.) ast.Walk on such nodes.
//
func childrenOf(n ast.Node) []ast.Node {
	var children []ast.Node
	return children
}

type NodeWalker struct {
	Paths   []*astutil.NodeReader
	CurPath *astutil.NodeReader
	Index   uint
}

func astPaths(reader *astutil.NodeReader, pos token.Pos) []*astutil.NodeReader {
	paths := []*astutil.NodeReader{}
	paths = append(paths, reader)

	for reader.NextNode(false) {
		if reader.CurNodeEncloseIs(pos) {
			// fmt.Println(fmt.Printf(
			// 	"CurNode Enclose, type: %T, value: %q, start: %+v, end: %+v, pos: %+v",
			// 	reader.CurNode,
			// 	reader.CurNode,
			// 	reader.CurNode.Pos(),
			// 	reader.CurNode.End(),
			// 	pos,
			// )) // debugging
			if list, ok := reader.CurNode.(ast.TokenList); ok {
				newReader := astutil.NewNodeReader(list)
				paths = append(paths, astPaths(newReader, pos)...)
			} else {
				return paths
			}
		}
		// fmt.Println(fmt.Printf(
		// 	"CurNode Not Enclose, type: %T, value: %q, start: %+v, end: %+v, pos: %+v",
		// 	reader.CurNode,
		// 	reader.CurNode,
		// 	reader.CurNode.Pos(),
		// 	reader.CurNode.End(),
		// 	pos,
		// )) // debugging
	}
	return paths
}

func NewNodeWalker(root ast.TokenList, pos token.Pos) *NodeWalker {
	paths := astPaths(astutil.NewNodeReader(root), pos)
	index := uint(len(paths) - 1)
	return &NodeWalker{
		Paths:   paths,
		CurPath: paths[index],
		Index:   index,
	}
}

func (nw *NodeWalker) PrevNodesIs(ignoreWitespace bool, matcher astutil.NodeMatcher) bool {
	for _, reader := range nw.Paths {
		if reader.PrevNodeIs(ignoreWitespace, matcher) {
			return true
		}
	}
	return false
}
