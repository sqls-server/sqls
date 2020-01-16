package parser

import (
	"io"

	"github.com/lighttiger2505/sqls/ast"
	"github.com/lighttiger2505/sqls/dialect"
	"github.com/lighttiger2505/sqls/token"
	"github.com/pkg/errors"
)

type writeContext struct {
	node     ast.TokenList
	curNode  ast.Node
	peekNode ast.Node
	index    uint
}

func newWriteContext(list ast.TokenList) *writeContext {
	wc := &writeContext{
		node: list,
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
	root = parseIdentifier(newWriteContext(root))
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
// parseParenthesis
// parseCase
// parseIf
// parseFor
// parseBegin
// parseFunctions
// parseWhere
// parsePeriod
// parseArrays

func parseIdentifier(wc *writeContext) ast.TokenList {
	var replaceNodes []ast.Node
	for wc.nextNode() {
		if wc.hasTokenList() {
			list := wc.mustTokenList()
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
// parseAliased
// parseAssignment
// alignComments
// parseIdentifierList
// parseValues
