package parseutil

import (
	"github.com/sqls-server/sqls/ast"
	"github.com/sqls-server/sqls/ast/astutil"
	"github.com/sqls-server/sqls/token"
)

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
			}
			return paths
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

func (nw *NodeWalker) CurNodeDepth(matcher astutil.NodeMatcher) (int, bool) {
	for i, reader := range nw.Paths {
		if reader.CurNodeIs(matcher) {
			return i, true
		}
	}
	return 0, false
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

func (nw *NodeWalker) CurNodeBottomMatched(matcher astutil.NodeMatcher) ast.Node {
	matches := nw.CurNodeMatches(matcher)
	if len(matches) == 0 {
		return nil
	}
	return matches[len(matches)-1]
}

func (nw *NodeWalker) CurNodes() []ast.Node {
	results := []ast.Node{}
	for _, reader := range nw.Paths {
		results = append(results, reader.CurNode)
	}
	return results
}

func (nw *NodeWalker) PrevNodes(ignoreWitespace bool) []ast.Node {
	results := []ast.Node{}
	for _, reader := range nw.Paths {
		_, node := reader.PrevNode(ignoreWitespace)
		results = append(results, node)
	}
	return results
}

func (nw *NodeWalker) PrevNodesIs(ignoreWitespace bool, matcher astutil.NodeMatcher) bool {
	for _, reader := range nw.Paths {
		if reader.PrevNodeIs(ignoreWitespace, matcher) {
			return true
		}
	}
	return false
}

func (nw *NodeWalker) PrevNodesIsWithDepth(ignoreWitespace bool, matcher astutil.NodeMatcher, depth int) bool {
	reader := nw.Paths[depth]
	return reader.PrevNodeIs(ignoreWitespace, matcher)
}
