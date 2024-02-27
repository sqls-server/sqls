package astutil

import (
	"fmt"
	"strings"

	"github.com/sqls-server/sqls/ast"
	"github.com/sqls-server/sqls/dialect"
	"github.com/sqls-server/sqls/token"
)

type NodeMatcher struct {
	NodeTypes     []ast.NodeType
	ExpectTokens  []token.Kind
	ExpectSQLType []dialect.KeywordKind
	ExpectKeyword []string
}

func (nm *NodeMatcher) IsMatchNodeTypes(node ast.Node) bool {
	if nm.NodeTypes != nil {
		for _, expect := range nm.NodeTypes {
			if expect == node.Type() {
				return true
			}
		}
	}
	return false
}

func (nm *NodeMatcher) IsMatchTokens(tok *ast.SQLToken) bool {
	if nm.ExpectTokens != nil {
		for _, expect := range nm.ExpectTokens {
			if tok.MatchKind(expect) {
				return true
			}
		}
	}
	return false
}

func (nm *NodeMatcher) IsMatchSQLType(tok *ast.SQLToken) bool {
	if nm.ExpectSQLType != nil {
		for _, expect := range nm.ExpectSQLType {
			if tok.MatchSQLKind(expect) {
				return true
			}
		}
	}
	return false
}

func (nm *NodeMatcher) IsMatchKeyword(node ast.Node) bool {
	if nm.ExpectKeyword != nil {
		for _, expect := range nm.ExpectKeyword {
			if strings.EqualFold(expect, node.String()) {
				return true
			}
		}
	}
	return false
}

func (nm *NodeMatcher) IsMatch(node ast.Node) bool {
	// For node object
	if nm.IsMatchNodeTypes(node) {
		return true
	}
	if nm.IsMatchKeyword(node) {
		return true
	}
	if _, ok := node.(ast.TokenList); ok {
		return false
	}
	// For token object
	tok, ok := node.(ast.Token)
	if !ok {
		panic(fmt.Sprintf("invalid type. not has Token, got=(type: %T, value: %#v)", node, node.String()))
	}
	sqlTok := tok.GetToken()
	if nm.IsMatchTokens(sqlTok) || nm.IsMatchSQLType(sqlTok) {
		return true
	}
	return false
}

func isWhitespace(node ast.Node) bool {
	tok, ok := node.(ast.Token)
	if !ok {
		return false
	}
	if tok.GetToken().MatchKind(token.Whitespace) {
		return true
	}
	return false
}

type NodeReader struct {
	Node    ast.TokenList
	CurNode ast.Node
	Index   int
}

func NewNodeReader(list ast.TokenList) *NodeReader {
	return &NodeReader{
		Node: list,
	}
}

func (nr *NodeReader) CopyReader() *NodeReader {
	return &NodeReader{
		Node:  nr.Node,
		Index: nr.Index,
	}
}

func (nr *NodeReader) Replace(add ast.Node, index int) {
	list := nr.Node.GetTokens()
	list = append(list[:index], list[index:]...)
	list[index] = add
	nr.Node.SetTokens(list)
}

func (nr *NodeReader) NodesWithRange(startIndex, endIndex int) []ast.Node {
	return nr.Node.GetTokens()[startIndex:endIndex]
}

func (nr *NodeReader) hasNext() bool {
	return nr.Index < len(nr.Node.GetTokens())
}

func (nr *NodeReader) hasPrev() bool {
	return 0 < nr.Index
}

func (nr *NodeReader) NextNode(ignoreWhiteSpace bool) bool {
	if !nr.hasNext() {
		return false
	}
	nr.CurNode = nr.Node.GetTokens()[nr.Index]
	nr.Index++

	if ignoreWhiteSpace && isWhitespace(nr.CurNode) {
		return nr.NextNode(ignoreWhiteSpace)
	}
	return true
}

func (nr *NodeReader) prev(ignoreWhiteSpace bool) bool {
	if !nr.hasPrev() {
		return false
	}
	nr.Index--
	nr.CurNode = nr.Node.GetTokens()[nr.Index]

	if ignoreWhiteSpace && isWhitespace(nr.CurNode) {
		return nr.prev(ignoreWhiteSpace)
	}
	return true
}

func (nr *NodeReader) CurNodeIs(nm NodeMatcher) bool {
	if nr.CurNode != nil {
		if nm.IsMatch(nr.CurNode) {
			return true
		}
	}
	return false
}

func IsEnclose(node ast.Node, pos token.Pos) bool {
	if 0 <= token.ComparePos(pos, node.Pos()) && 0 >= token.ComparePos(pos, node.End()) {
		return true
	}
	return false
}

func (nr *NodeReader) CurNodeEncloseIs(pos token.Pos) bool {
	if nr.CurNode != nil {
		return IsEnclose(nr.CurNode, pos)
	}
	return false
}

func (nr *NodeReader) PeekNodeEncloseIs(pos token.Pos) bool {
	_, peekNode := nr.PeekNode(false)
	if peekNode != nil {
		return IsEnclose(peekNode, pos)
	}
	return false
}

func (nr *NodeReader) PeekNode(ignoreWhiteSpace bool) (int, ast.Node) {
	tmpReader := nr.CopyReader()
	for tmpReader.hasNext() {
		index := tmpReader.Index
		node := tmpReader.Node.GetTokens()[index]

		if ignoreWhiteSpace {
			if !isWhitespace(node) {
				return index, node
			}
		} else {
			return index, node
		}
		tmpReader.NextNode(false)
	}
	return 0, nil
}

func (nr *NodeReader) PeekNodeIs(ignoreWhiteSpace bool, nm NodeMatcher) bool {
	_, node := nr.PeekNode(ignoreWhiteSpace)
	if node != nil {
		if nm.IsMatch(node) {
			return true
		}
	}
	return false
}

func (nr *NodeReader) TailNode() (int, ast.Node) {
	var (
		index int
		node  ast.Node
	)

	tmpReader := nr.CopyReader()
	for {
		index = tmpReader.Index
		node = tmpReader.CurNode
		if !tmpReader.hasNext() {
			break
		}
		tmpReader.NextNode(false)
	}
	return index, node
}

func (nr *NodeReader) FindNode(ignoreWhiteSpace bool, nm NodeMatcher) (*NodeReader, ast.Node) {
	tmpReader := nr.CopyReader()
	for tmpReader.hasNext() {
		node := tmpReader.Node.GetTokens()[tmpReader.Index]

		// For node object
		if nm.IsMatchNodeTypes(node) {
			return tmpReader, node
		}
		if _, ok := tmpReader.CurNode.(ast.TokenList); ok {
			continue
		}
		// For token object
		tok, _ := nr.CurNode.(ast.Token)
		sqlTok := tok.GetToken()
		if nm.IsMatchTokens(sqlTok) || nm.IsMatchSQLType(sqlTok) || nm.IsMatchKeyword(sqlTok) {
			return tmpReader, node
		}
		tmpReader.NextNode(ignoreWhiteSpace)
	}
	return nil, nil
}

func (nr *NodeReader) FindRecursive(matcher NodeMatcher) []ast.Node {
	matches := []ast.Node{}
	for nr.NextNode(false) {
		if nr.CurNodeIs(matcher) {
			matches = append(matches, nr.CurNode)
		}
		if list, ok := nr.CurNode.(ast.TokenList); ok {
			newReader := NewNodeReader(list)
			matches = append(matches, newReader.FindRecursive(matcher)...)
		}
	}
	return matches
}

func (nr *NodeReader) PrevNode(ignoreWhiteSpace bool) (int, ast.Node) {
	if !nr.hasPrev() {
		return 0, nil
	}
	tmpReader := nr.CopyReader()
	tmpReader.prev(false)

	for tmpReader.prev(ignoreWhiteSpace) {
		index := tmpReader.Index
		node := tmpReader.CurNode

		if ignoreWhiteSpace {
			if !isWhitespace(node) {
				return index, node
			}
		} else {
			return index, node
		}

		if !tmpReader.hasPrev() {
			break
		}
	}
	return 0, nil
}

func (nr *NodeReader) PrevNodeIs(ignoreWhiteSpace bool, nm NodeMatcher) bool {
	_, node := nr.PrevNode(ignoreWhiteSpace)
	if node != nil {
		if nm.IsMatch(node) {
			return true
		}
	}
	return false
}
