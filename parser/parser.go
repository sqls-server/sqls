package parser

import (
	"fmt"
	"io"

	"github.com/lighttiger2505/sqls/ast"
	"github.com/lighttiger2505/sqls/dialect"
	"github.com/lighttiger2505/sqls/token"
	"github.com/pkg/errors"
)

type nodeMatcher struct {
	nodeTypeMatcherFunc func(node interface{}) bool
	expectTokens        []token.Kind
	expectSQLType       []dialect.KeywordKind
	expectKeyword       []string
}

func (f *nodeMatcher) isMatchNodeType(node interface{}) bool {
	if f.nodeTypeMatcherFunc != nil {
		if f.nodeTypeMatcherFunc(node) {
			return true
		}
	}
	return false
}

func (f *nodeMatcher) isMatchTokens(tok *ast.SQLToken) bool {
	if f.expectTokens != nil {
		for _, expect := range f.expectTokens {
			if tok.MatchKind(expect) {
				return true
			}
		}
	}
	return false
}

func (f *nodeMatcher) isMatchSQLType(tok *ast.SQLToken) bool {
	if f.expectSQLType != nil {
		for _, expect := range f.expectSQLType {
			if tok.MatchSQLKind(expect) {
				return true
			}
		}
	}
	return false
}

func (f *nodeMatcher) isMatchKeyword(tok *ast.SQLToken) bool {
	if f.expectKeyword != nil {
		for _, expect := range f.expectKeyword {
			if tok.MatchSQLKeyword(expect) {
				return true
			}
		}
	}
	return false
}

func (f *nodeMatcher) isMatch(node ast.Node) bool {
	// For node object
	if f.isMatchNodeType(node) {
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
	if f.isMatchTokens(sqlTok) || f.isMatchSQLType(sqlTok) || f.isMatchKeyword(sqlTok) {
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

type nodeReader struct {
	node    ast.TokenList
	curNode ast.Node
	index   uint
}

func newNodeReader(list ast.TokenList) *nodeReader {
	return &nodeReader{
		node: list,
	}
}

func (nr *nodeReader) copyReader() *nodeReader {
	return &nodeReader{
		node:  nr.node,
		index: nr.index,
	}
}

func (nr *nodeReader) nodesWithRange(startIndex, endIndex uint) []ast.Node {
	return nr.node.GetTokens()[startIndex:endIndex]
}

func (nr *nodeReader) hasNext() bool {
	return nr.index < uint(len(nr.node.GetTokens()))
}

func (nr *nodeReader) nextNode(ignoreWhiteSpace bool) bool {
	if !nr.hasNext() {
		return false
	}
	nr.curNode = nr.node.GetTokens()[nr.index]
	nr.index++

	if ignoreWhiteSpace && isWhitespace(nr.curNode) {
		return nr.nextNode(ignoreWhiteSpace)
	}
	return true
}

func (nr *nodeReader) curNodeIs(fd nodeMatcher) bool {
	if nr.curNode != nil {
		if fd.isMatch(nr.curNode) {
			return true
		}
	}
	return false
}

func (nr *nodeReader) peekNode(ignoreWhiteSpace bool) (uint, ast.Node) {
	tmpReader := nr.copyReader()
	for tmpReader.hasNext() {
		index := tmpReader.index
		node := tmpReader.node.GetTokens()[index]

		if ignoreWhiteSpace {
			if !isWhitespace(node) {
				return index, node
			}
		} else {
			return index, node
		}
		tmpReader.nextNode(false)
	}
	return 0, nil
}

func (nr *nodeReader) peekNodeIs(ignoreWhiteSpace bool, fd nodeMatcher) bool {
	_, node := nr.peekNode(ignoreWhiteSpace)
	if node != nil {
		if fd.isMatch(node) {
			return true
		}
	}
	return false
}

func (nr *nodeReader) matchedPeekNode(ignoreWhiteSpace bool, fd nodeMatcher) (uint, ast.Node) {
	index, node := nr.peekNode(ignoreWhiteSpace)
	if node != nil {
		if fd.isMatch(node) {
			return index, node
		}
	}
	return 0, nil
}

func (nr *nodeReader) findNode(ignoreWhiteSpace bool, fd nodeMatcher) (*nodeReader, ast.Node) {
	tmpReader := nr.copyReader()
	for tmpReader.hasNext() {
		node := tmpReader.node.GetTokens()[tmpReader.index]

		// For node object
		if fd.isMatchNodeType(node) {
			return tmpReader, node
		}
		if tmpReader.hasTokenList() {
			continue
		}
		// For token object
		tok, _ := nr.curNode.(ast.Token)
		sqlTok := tok.GetToken()
		if fd.isMatchTokens(sqlTok) || fd.isMatchSQLType(sqlTok) || fd.isMatchKeyword(sqlTok) {
			return tmpReader, node
		}
		tmpReader.nextNode(ignoreWhiteSpace)
	}
	return nil, nil
}

func (nr *nodeReader) hasTokenList() bool {
	_, ok := nr.curNode.(ast.TokenList)
	return ok
}

func (nr *nodeReader) getTokenList() (ast.TokenList, error) {
	if !nr.hasTokenList() {
		return nil, errors.Errorf("want TokenList got %T", nr.curNode)
	}
	children, _ := nr.curNode.(ast.TokenList)
	return children, nil
}

func (nr *nodeReader) mustTokenList() ast.TokenList {
	children, _ := nr.getTokenList()
	return children
}

type (
	prefixParseFn func(reader *nodeReader) ast.Node
	infixParseFn  func(reader *nodeReader) ast.Node
)

func parsePrefixGroup(reader *nodeReader, matcher nodeMatcher, fn prefixParseFn) ast.TokenList {
	var replaceNodes []ast.Node
	for reader.nextNode(false) {
		if list, ok := reader.curNode.(ast.TokenList); ok {
			newReader := newNodeReader(list)
			replaceNodes = append(replaceNodes, parsePrefixGroup(newReader, matcher, fn))
			continue
		}

		if reader.curNodeIs(matcher) {
			replaceNodes = append(replaceNodes, fn(reader))
		} else {
			replaceNodes = append(replaceNodes, reader.curNode)
		}
	}
	reader.node.SetTokens(replaceNodes)
	return reader.node
}

func parseInfixGroup(reader *nodeReader, matcher nodeMatcher, ignoreWhiteSpace bool, fn infixParseFn) ast.TokenList {
	var replaceNodes []ast.Node
	for reader.nextNode(false) {

		if reader.peekNodeIs(ignoreWhiteSpace, matcher) {
			replaceNodes = append(replaceNodes, fn(reader))
		} else if list, ok := reader.curNode.(ast.TokenList); ok {
			newReader := newNodeReader(list)
			replaceNodes = append(replaceNodes, parseInfixGroup(newReader, matcher, ignoreWhiteSpace, fn))
		} else {
			replaceNodes = append(replaceNodes, reader.curNode)
		}
	}
	reader.node.SetTokens(replaceNodes)
	return reader.node
}

type Parser struct {
	root ast.TokenList
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
	root = parseStatement(newNodeReader(root))
	root = parsePrefixGroup(newNodeReader(root), parenthesisPrefixMatcher, parseParenthesis)
	root = parsePrefixGroup(newNodeReader(root), functionPrefixMatcher, parseFunctions)
	root = parsePrefixGroup(newNodeReader(root), wherePrefixMatcher, parseWhere)
	root = parseInfixGroup(newNodeReader(root), memberIdentifierInfixMatcher, false, parseMemberIdentifier)
	root = parsePrefixGroup(newNodeReader(root), identifierPrefixMatcher, parseIdentifier)
	root = parseInfixGroup(newNodeReader(root), operatorInfixMatcher, true, parseOperator)
	root = parseInfixGroup(newNodeReader(root), comparisonInfixMatcher, true, parseComparison)
	root = parseInfixGroup(newNodeReader(root), aliasInfixMatcher, true, parseAliased)
	return root, nil
}

var statementMatcher = nodeMatcher{
	expectTokens: []token.Kind{
		token.Semicolon,
	},
}

func parseStatement(reader *nodeReader) ast.TokenList {
	var replaceNodes []ast.Node
	var startIndex uint
	for reader.nextNode(false) {
		if reader.hasTokenList() {
			list := reader.mustTokenList()
			replaceNodes = append(replaceNodes, parseStatement(newNodeReader(list)))
			continue
		}

		tmpReader, node := reader.findNode(true, statementMatcher)
		if node != nil {
			stmt := &ast.Statement{Toks: reader.nodesWithRange(startIndex, tmpReader.index)}
			replaceNodes = append(replaceNodes, stmt)
			reader = tmpReader
			startIndex = reader.index
		}
	}
	if reader.index != startIndex {
		stmt := &ast.Statement{Toks: reader.nodesWithRange(startIndex, reader.index)}
		replaceNodes = append(replaceNodes, stmt)
	}
	reader.node.SetTokens(replaceNodes)
	return reader.node
}

// parseComments
// parseBrackets

var parenthesisPrefixMatcher = nodeMatcher{
	expectTokens: []token.Kind{
		token.LParen,
	},
}
var parenthesisCloseMatcher = nodeMatcher{
	expectTokens: []token.Kind{
		token.RParen,
	},
}

func parseParenthesis(reader *nodeReader) ast.Node {
	nodes := []ast.Node{reader.curNode}
	tmpReader := reader.copyReader()
	for tmpReader.nextNode(false) {
		if tmpReader.hasTokenList() {
			continue
		}

		if tmpReader.curNodeIs(parenthesisPrefixMatcher) {
			parenthesis := parseParenthesis(tmpReader)
			nodes = append(nodes, parenthesis)
		} else if tmpReader.curNodeIs(parenthesisCloseMatcher) {
			reader.index = tmpReader.index
			reader.curNode = tmpReader.curNode
			return &ast.Parenthesis{Toks: append(nodes, tmpReader.curNode)}
		} else {
			nodes = append(nodes, tmpReader.curNode)
		}
	}
	return reader.curNode
}

// parseCase
// parseIf
// parseFor
// parseBegin

var functionPrefixMatcher = nodeMatcher{
	expectSQLType: []dialect.KeywordKind{
		dialect.Matched,
		dialect.Unmatched,
	},
}
var functionArgsMatcher = nodeMatcher{
	nodeTypeMatcherFunc: func(node interface{}) bool {
		if _, ok := node.(*ast.Parenthesis); ok {
			return true
		}
		return false
	},
}

func parseFunctions(reader *nodeReader) ast.Node {
	funcName := reader.curNode
	if _, funcArgs := reader.matchedPeekNode(false, functionArgsMatcher); funcArgs != nil {
		function := &ast.Function{Toks: []ast.Node{funcName, funcArgs}}
		reader.nextNode(false)
		return function
	}
	return reader.curNode
}

var wherePrefixMatcher = nodeMatcher{
	expectKeyword: []string{
		"WHERE",
	},
}
var whereCloseMatcher = nodeMatcher{
	expectTokens: []token.Kind{
		token.RParen,
	},
	expectKeyword: []string{
		"ORDER",
		"GROUP",
		"LIMIT",
		"UNION",
		"EXCEPT",
		"HAVING",
		"RETURNING",
		"INTO",
	},
}

func parseWhere(reader *nodeReader) ast.Node {
	nodes := []ast.Node{reader.curNode}
	for reader.nextNode(false) {
		if reader.hasTokenList() {
			continue
		}

		if reader.curNodeIs(wherePrefixMatcher) {
			nodes = append(nodes, parseWhere(reader))
		} else if reader.peekNodeIs(false, whereCloseMatcher) {
			nodes = append(nodes, reader.curNode)
			return &ast.Where{Toks: nodes}
		} else {
			nodes = append(nodes, reader.curNode)
		}
	}
	return &ast.Where{Toks: nodes}
}

var memberIdentifierInfixMatcher = nodeMatcher{
	expectTokens: []token.Kind{
		token.Period,
	},
}
var memberIdentifierTargetMatcher = nodeMatcher{
	expectTokens: []token.Kind{
		token.Mult,
	},
	expectSQLType: []dialect.KeywordKind{
		dialect.Unmatched,
	},
}

func parseMemberIdentifier(reader *nodeReader) ast.Node {
	if !reader.curNodeIs(memberIdentifierTargetMatcher) {
		return reader.curNode
	}
	startIndex := reader.index - 1
	memberIdentifier := &ast.MemberIdentifer{Toks: reader.nodesWithRange(startIndex, reader.index+1)}
	reader.nextNode(false)

	endIndex, right := reader.matchedPeekNode(true, memberIdentifierTargetMatcher)
	if right == nil {
		return memberIdentifier
	}
	reader.nextNode(false)

	memberIdentifier = &ast.MemberIdentifer{Toks: reader.nodesWithRange(startIndex, endIndex+1)}
	return memberIdentifier
}

// parseArrays

var identifierPrefixMatcher = nodeMatcher{
	expectSQLType: []dialect.KeywordKind{
		dialect.Unmatched,
	},
}

func parseIdentifier(reader *nodeReader) ast.Node {
	token, _ := reader.curNode.(ast.Token)
	return &ast.Identifer{Tok: token.GetToken()}
}

// parseOrder
// parseTypecasts
// parseTzcasts
// parseTyped_literal

var operatorInfixMatcher = nodeMatcher{
	expectTokens: []token.Kind{
		token.Plus,
		token.Minus,
		token.Mult,
		token.Div,
		token.Mod,
	},
}
var operatorTargetMatcher = nodeMatcher{
	nodeTypeMatcherFunc: func(node interface{}) bool {
		if _, ok := node.(*ast.Identifer); ok {
			return true
		}
		if _, ok := node.(*ast.MemberIdentifer); ok {
			return true
		}
		if _, ok := node.(*ast.Operator); ok {
			return true
		}
		if _, ok := node.(*ast.Parenthesis); ok {
			return true
		}
		if _, ok := node.(*ast.Function); ok {
			return true
		}
		return false
	},
	expectTokens: []token.Kind{
		token.Number,
		token.Char,
		token.SingleQuotedString,
		token.NationalStringLiteral,
	},
}

func parseOperator(reader *nodeReader) ast.Node {
	if !reader.curNodeIs(operatorTargetMatcher) {
		return reader.curNode
	}
	startIndex := reader.index - 1
	tmpReader := reader.copyReader()
	tmpReader.nextNode(true)

	endIndex, right := tmpReader.matchedPeekNode(true, operatorTargetMatcher)
	if right == nil {
		return reader.curNode
	}
	tmpReader.nextNode(true)
	reader.index = tmpReader.index
	reader.curNode = tmpReader.curNode

	return &ast.Operator{Toks: reader.nodesWithRange(startIndex, endIndex+1)}
}

var comparisonInfixMatcher = nodeMatcher{
	expectTokens: []token.Kind{
		token.Eq,
		token.Neq,
		token.Lt,
		token.Gt,
		token.LtEq,
		token.GtEq,
	},
}
var comparisonTargetMatcher = nodeMatcher{
	nodeTypeMatcherFunc: func(node interface{}) bool {
		if _, ok := node.(*ast.Parenthesis); ok {
			return true
		}
		if _, ok := node.(*ast.Identifer); ok {
			return true
		}
		if _, ok := node.(*ast.MemberIdentifer); ok {
			return true
		}
		if _, ok := node.(*ast.Operator); ok {
			return true
		}
		if _, ok := node.(*ast.Function); ok {
			return true
		}
		return false
	},
	expectTokens: []token.Kind{
		token.Number,
		token.Char,
		token.SingleQuotedString,
		token.NationalStringLiteral,
	},
}

func parseComparison(reader *nodeReader) ast.Node {
	if !reader.curNodeIs(comparisonTargetMatcher) {
		return reader.curNode
	}
	startIndex := reader.index - 1
	tmpReader := reader.copyReader()
	tmpReader.nextNode(true)

	endIndex, right := tmpReader.matchedPeekNode(true, comparisonTargetMatcher)
	if right == nil {
		return reader.curNode
	}
	tmpReader.nextNode(true)
	reader.index = tmpReader.index
	reader.curNode = tmpReader.curNode

	return &ast.Comparison{Toks: reader.nodesWithRange(startIndex, endIndex+1)}
}

// ast.Identifer,
// ast.MemberIdentifer,
// ast.Parenthesis,

var aliasInfixMatcher = nodeMatcher{
	expectKeyword: []string{
		"AS",
	},
}
var aliasTargetMatcher = nodeMatcher{
	nodeTypeMatcherFunc: func(node interface{}) bool {
		if _, ok := node.(*ast.Identifer); ok {
			return true
		}
		if _, ok := node.(*ast.MemberIdentifer); ok {
			return true
		}
		if _, ok := node.(*ast.Parenthesis); ok {
			return true
		}
		return false
	},
}

func parseAliased(reader *nodeReader) ast.Node {
	if !reader.curNodeIs(aliasTargetMatcher) {
		return reader.curNode
	}
	startIndex := reader.index - 1
	tmpReader := reader.copyReader()
	tmpReader.nextNode(true)

	endIndex, aliasedName := tmpReader.matchedPeekNode(true, aliasTargetMatcher)
	if aliasedName == nil {
		return reader.curNode
	}
	tmpReader.nextNode(true)
	reader.index = tmpReader.index
	reader.curNode = tmpReader.curNode

	return &ast.Aliased{Toks: reader.nodesWithRange(startIndex, endIndex+1)}
}

// parseAssignment
// alignComments
// parseIdentifierList
// parseValues
