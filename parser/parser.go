package parser

import (
	"io"

	"github.com/lighttiger2505/sqls/ast"
	"github.com/lighttiger2505/sqls/dialect"
	"github.com/lighttiger2505/sqls/token"
	"github.com/pkg/errors"
)

type writeContext struct {
	node    ast.TokenList
	curNode ast.Node
	index   uint
}

func newWriteContext(list ast.TokenList) *writeContext {
	wc := &writeContext{
		node: list,
	}
	return wc
}

func newWriteContextWithIndex(list ast.TokenList, index uint) *writeContext {
	wc := &writeContext{
		node:  list,
		index: index,
	}
	return wc
}

func (wc *writeContext) nodesWithRange(startIndex, endIndex uint) []ast.Node {
	return wc.node.GetTokens()[startIndex:endIndex]
}

func (wc *writeContext) replaceIndex(add ast.Node, index uint) {
	wc.node.GetTokens()[index] = add
}

func (wc *writeContext) replace(add ast.Node, startIndex, endIndex uint) {
	oldList := wc.node.GetTokens()

	start := oldList[:startIndex]
	end := oldList[endIndex:]

	var out []ast.Node
	out = append(out, start...)
	out = append(out, add)
	out = append(out, end...)
	wc.node.SetTokens(out)

	offset := (endIndex - startIndex)
	wc.index = wc.index - uint(offset)
	wc.nextNode()
}

func (wc *writeContext) hasNext() bool {
	return wc.index < uint(len(wc.node.GetTokens()))
}

func (wc *writeContext) nextNode() bool {
	if !wc.hasNext() {
		return false
	}
	wc.curNode = wc.node.GetTokens()[wc.index]
	wc.index++
	return true
}

func (wc *writeContext) hasTokenList() bool {
	_, ok := wc.curNode.(ast.TokenList)
	return ok
}

func (wc *writeContext) getTokenList() (ast.TokenList, error) {
	if !wc.hasTokenList() {
		return nil, errors.Errorf("want TokenList got %T", wc.curNode)
	}
	children, _ := wc.curNode.(ast.TokenList)
	return children, nil
}

func (wc *writeContext) mustTokenList() ast.TokenList {
	children, _ := wc.getTokenList()
	return children
}

func (wc *writeContext) hasToken() bool {
	_, ok := wc.curNode.(ast.Token)
	return ok
}

func (wc *writeContext) getToken() (*ast.SQLToken, error) {
	if !wc.hasToken() {
		return nil, errors.Errorf("want Token got %T", wc.curNode)
	}
	token, _ := wc.curNode.(ast.Token)
	return token.GetToken(), nil
}

func (wc *writeContext) mustToken() *ast.SQLToken {
	token, _ := wc.getToken()
	return token
}

func (wc *writeContext) getPeekToken() (*ast.SQLToken, error) {
	if !wc.hasNext() {
		return nil, errors.Errorf("EOF")
	}
	tok, ok := wc.node.GetTokens()[wc.index].(ast.Token)
	if !ok {
		return nil, errors.Errorf("want Token got %T", wc.curNode)
	}
	return tok.GetToken(), nil
}

func (wc *writeContext) peekTokenMatchKind(expect token.Kind) bool {
	token, err := wc.getPeekToken()
	if err != nil {
		return false
	}
	return token.MatchKind(expect)
}

func (wc *writeContext) peekTokenMatchSQLKind(expect dialect.KeywordKind) bool {
	token, err := wc.getPeekToken()
	if err != nil {
		return false
	}
	return token.MatchSQLKind(expect)
}

func (wc *writeContext) peekTokenMatchSQLKeyword(expect string) bool {
	token, err := wc.getPeekToken()
	if err != nil {
		return false
	}
	return token.MatchSQLKeyword(expect)
}

func (wc *writeContext) peekTokenMatchSQLKeywords(expects []string) bool {
	token, err := wc.getPeekToken()
	if err != nil {
		return false
	}
	return token.MatchSQLKeywords(expects)
}

type (
	prefixParseFn func() ast.Node
	infixParseFn  func(ast.Node) ast.Node
)

type Parser struct {
	root ast.TokenList

	prefixParseFns map[token.Kind]prefixParseFn
	infixParseFns  map[token.Kind]infixParseFn
}

func NewParser(src io.Reader, d dialect.Dialect) (*Parser, error) {
	tokenizer := token.NewTokenizer(src, d)
	tokens, err := tokenizer.Tokenize()
	if err != nil {
		return nil, errors.Errorf("tokenize err failed: %w", err)
	}

	parsed := []ast.Node{}
	for _, tok := range tokens {
		parsed = append(parsed, ast.NewItem(tok))
	}

	parser := &Parser{
		root: &ast.Query{Toks: parsed},
	}

	return parser, nil
}

func (p *Parser) Parse() (ast.TokenList, error) {
	root := p.root
	root = parseStatement(newWriteContext(root))
	root = parseParenthesis(newWriteContext(root))
	root = parseWhere(newWriteContext(root))
	root = parsePeriod(newWriteContext(root))
	root = parseIdentifier(newWriteContext(root))
	root = parseAliased(newWriteContext(root))
	return root, nil
}

func parseStatement(wc *writeContext) ast.TokenList {
	var replaceNodes []ast.Node
	var startIndex uint
	for wc.nextNode() {
		if wc.hasTokenList() {
			list := wc.mustTokenList()
			replaceNodes = append(replaceNodes, parseStatement(newWriteContext(list)))
			continue
		}

		tok := wc.mustToken()
		if tok.MatchKind(token.Semicolon) {
			stmt := &ast.Statement{Toks: wc.nodesWithRange(startIndex, wc.index)}
			replaceNodes = append(replaceNodes, stmt)
			startIndex = wc.index
		}
	}
	if wc.index != startIndex {
		stmt := &ast.Statement{Toks: wc.nodesWithRange(startIndex, wc.index)}
		replaceNodes = append(replaceNodes, stmt)
	}
	wc.node.SetTokens(replaceNodes)
	return wc.node
}

// parseComments
// parseBrackets

func parseParenthesis(wc *writeContext) ast.TokenList {
	var replaceNodes []ast.Node

	for wc.nextNode() {
		if wc.hasTokenList() {
			list := wc.mustTokenList()
			replaceNodes = append(replaceNodes, parseParenthesis(newWriteContext(list)))
			continue
		}

		tok := wc.mustToken()
		if tok.MatchKind(token.LParen) {
			newWC := newWriteContextWithIndex(wc.node, wc.index)
			parenthesis := findParenthesisMatch(newWC, wc.curNode, wc.index)
			if parenthesis != nil {
				wc = newWC
				replaceNodes = append(replaceNodes, parenthesis)
			} else {
				replaceNodes = append(replaceNodes, wc.curNode)
			}
		} else {
			replaceNodes = append(replaceNodes, wc.curNode)
		}
	}
	wc.node.SetTokens(replaceNodes)
	return wc.node
}

func findParenthesisMatch(wc *writeContext, startTok ast.Node, startIndex uint) ast.Node {
	var nodes []ast.Node
	nodes = append(nodes, startTok)
	for wc.nextNode() {
		if wc.hasTokenList() {
			continue
		}

		tok := wc.mustToken()
		if tok.MatchKind(token.LParen) {
			group := findParenthesisMatch(wc, wc.curNode, wc.index)
			nodes = append(nodes, group)
		} else if tok.MatchKind(token.RParen) {
			nodes = append(nodes, wc.curNode)
			return &ast.Parenthesis{Toks: nodes}
		} else {
			nodes = append(nodes, wc.curNode)
		}
	}
	return nil
}

// parseCase
// parseIf
// parseFor
// parseBegin
// parseFunctions

var WhereOpenKeyword = "WHERE"
var WhereCloseKeywords = []string{
	"ORDER",
	"GROUP",
	"LIMIT",
	"UNION",
	"EXCEPT",
	"HAVING",
	"RETURNING",
	"INTO",
}

func parseWhere(wc *writeContext) ast.TokenList {
	var replaceNodes []ast.Node

	for wc.nextNode() {
		if wc.hasTokenList() {
			list := wc.mustTokenList()
			replaceNodes = append(replaceNodes, parseWhere(newWriteContext(list)))
			continue
		}

		tok := wc.mustToken()
		if tok.MatchSQLKeyword(WhereOpenKeyword) {
			where := findWhereMatch(wc, wc.curNode, wc.index)
			if where != nil {
				replaceNodes = append(replaceNodes, where)
			}
		} else {
			replaceNodes = append(replaceNodes, wc.curNode)
		}
	}
	wc.node.SetTokens(replaceNodes)
	return wc.node
}

func findWhereMatch(wc *writeContext, startTok ast.Node, startIndex uint) ast.Node {
	var nodes []ast.Node
	nodes = append(nodes, startTok)
	for wc.nextNode() {
		if wc.hasTokenList() {
			continue
		}

		tok := wc.mustToken()
		if tok.MatchSQLKeyword(WhereOpenKeyword) {
			group := findWhereMatch(wc, wc.curNode, wc.index)
			nodes = append(nodes, group)
		} else if wc.peekTokenMatchSQLKeywords(WhereCloseKeywords) {
			nodes = append(nodes, wc.curNode)
			return &ast.Where{Toks: nodes}
		} else {
			nodes = append(nodes, wc.curNode)
		}
		if wc.peekTokenMatchKind(token.RParen) {
			break
		}
	}
	return &ast.Where{Toks: nodes}
}

func parsePeriod(wc *writeContext) ast.TokenList {
	var replaceNodes []ast.Node
	for wc.nextNode() {
		if wc.hasTokenList() {
			list := wc.mustTokenList()
			replaceNodes = append(replaceNodes, parsePeriod(newWriteContext(list)))
			continue
		}

		tok := wc.mustToken()
		if wc.peekTokenMatchKind(token.Period) {
			memberIdentifer := &ast.MemberIdentifer{
				Parent: tok,
			}
			wc.nextNode()
			period := wc.mustToken()
			memberIdentifer.Period = period

			if wc.peekTokenMatchSQLKind(dialect.Unmatched) || wc.peekTokenMatchKind(token.Mult) {
				wc.nextNode()
				child := wc.mustToken()
				memberIdentifer.Child = child
			}
			replaceNodes = append(replaceNodes, memberIdentifer)
		} else {
			replaceNodes = append(replaceNodes, wc.curNode)
		}
	}
	wc.node.SetTokens(replaceNodes)
	return wc.node
}

// parseArrays

func parseIdentifier(wc *writeContext) ast.TokenList {
	var replaceNodes []ast.Node
	for wc.nextNode() {
		if wc.hasTokenList() {
			list := wc.mustTokenList()
			if _, ok := list.(*ast.MemberIdentifer); ok {
				replaceNodes = append(replaceNodes, wc.curNode)
				continue
			}
			replaceNodes = append(replaceNodes, parseIdentifier(newWriteContext(list)))
			continue
		}

		tok := wc.mustToken()
		if tok.MatchSQLKind(dialect.Unmatched) {
			identifer := &ast.Identifer{Tok: tok}
			replaceNodes = append(replaceNodes, identifer)
		} else {
			replaceNodes = append(replaceNodes, wc.curNode)
		}
	}
	wc.node.SetTokens(replaceNodes)
	return wc.node
}

// parseOrder
// parseTypecasts
// parseTzcasts
// parseTyped_literal
// parseOperator
// parseComparison
// parseAs

// ast.Identifer,
// ast.MemberIdentifer,
// ast.Parenthesis,

func parseAliased(wc *writeContext) ast.TokenList {
	var replaceNodes []ast.Node
	for wc.nextNode() {
		if wc.hasTokenList() {
			list := wc.mustTokenList()
			replaceNodes = append(replaceNodes, parseAliased(newWriteContext(list)))
			continue
		}

		if _, ok := wc.curNode.(*ast.Identifer); ok {
			newWC := newWriteContextWithIndex(wc.node, wc.index)
			aliased := findAliasMatch(newWC, wc.curNode, wc.index)
			if aliased != nil {
				wc = newWC
				replaceNodes = append(replaceNodes, aliased)
			} else {
				replaceNodes = append(replaceNodes, wc.curNode)
			}
		} else {
			replaceNodes = append(replaceNodes, wc.curNode)
		}
	}
	wc.node.SetTokens(replaceNodes)
	return wc.node
}

func findAliasMatch(wc *writeContext, startTok ast.Node, startIndex uint) ast.Node {
	var nodes []ast.Node
	nodes = append(nodes, startTok)
	for wc.nextNode() {
		if wc.hasTokenList() {
			continue
		}

		if _, ok := wc.curNode.(*ast.Identifer); ok {
			nodes = append(nodes, wc.curNode)
			return &ast.Aliased{Toks: nodes}
		}

		tok := wc.mustToken()
		if tok.MatchSQLKeyword("AS") || tok.MatchKind(token.Whitespace) {
			nodes = append(nodes, wc.curNode)
		} else {
			break
		}
	}
	return nil
}

// parseAssignment
// alignComments
// parseIdentifierList
// parseValues
