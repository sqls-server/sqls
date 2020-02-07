package parser

import (
	"fmt"
	"io"

	"github.com/lighttiger2505/sqls/ast"
	"github.com/lighttiger2505/sqls/ast/astutil"
	"github.com/lighttiger2505/sqls/dialect"
	"github.com/lighttiger2505/sqls/token"
	"github.com/pkg/errors"
)

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

func (nr *nodeReader) curNodeIs(nm astutil.NodeMatcher) bool {
	if nr.curNode != nil {
		if nm.IsMatch(nr.curNode) {
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

func (nr *nodeReader) peekNodeIs(ignoreWhiteSpace bool, nm astutil.NodeMatcher) bool {
	_, node := nr.peekNode(ignoreWhiteSpace)
	if node != nil {
		if nm.IsMatch(node) {
			return true
		}
	}
	return false
}

func (nr *nodeReader) matchedPeekNode(ignoreWhiteSpace bool, nm astutil.NodeMatcher) (uint, ast.Node) {
	index, node := nr.peekNode(ignoreWhiteSpace)
	if node != nil {
		if nm.IsMatch(node) {
			return index, node
		}
	}
	return 0, nil
}

func (nr *nodeReader) findNode(ignoreWhiteSpace bool, nm astutil.NodeMatcher) (*nodeReader, ast.Node) {
	tmpReader := nr.copyReader()
	for tmpReader.hasNext() {
		node := tmpReader.node.GetTokens()[tmpReader.index]

		// For node object
		if nm.IsMatchNodeType(node) {
			return tmpReader, node
		}
		if tmpReader.hasTokenList() {
			continue
		}
		// For token object
		tok, _ := nr.curNode.(ast.Token)
		sqlTok := tok.GetToken()
		if nm.IsMatchTokens(sqlTok) || nm.IsMatchSQLType(sqlTok) || nm.IsMatchKeyword(sqlTok) {
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

func parsePrefixGroup(reader *nodeReader, matcher astutil.NodeMatcher, fn prefixParseFn) ast.TokenList {
	var replaceNodes []ast.Node
	for reader.nextNode(false) {
		if reader.curNodeIs(matcher) {
			replaceNodes = append(replaceNodes, fn(reader))
		} else if list, ok := reader.curNode.(ast.TokenList); ok {
			newReader := newNodeReader(list)
			replaceNodes = append(replaceNodes, parsePrefixGroup(newReader, matcher, fn))
		} else {
			replaceNodes = append(replaceNodes, reader.curNode)
		}
	}
	reader.node.SetTokens(replaceNodes)
	return reader.node
}

func parseInfixGroup(reader *nodeReader, matcher astutil.NodeMatcher, ignoreWhiteSpace bool, fn infixParseFn) ast.TokenList {
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
	root = parsePrefixGroup(newNodeReader(root), FromPrefixMatcher, parseFrom)
	root = parsePrefixGroup(newNodeReader(root), JoinPrefixMatcher, parseJoin)
	root = parsePrefixGroup(newNodeReader(root), wherePrefixMatcher, parseWhere)
	root = parseInfixGroup(newNodeReader(root), memberIdentifierInfixMatcher, false, parseMemberIdentifier)
	root = parsePrefixGroup(newNodeReader(root), identifierPrefixMatcher, parseIdentifier)
	root = parseInfixGroup(newNodeReader(root), operatorInfixMatcher, true, parseOperator)
	root = parseInfixGroup(newNodeReader(root), comparisonInfixMatcher, true, parseComparison)
	root = parseInfixGroup(newNodeReader(root), aliasInfixMatcher, true, parseAliased)
	root = parseInfixGroup(newNodeReader(root), identifierListInfixMatcher, true, parseIdentifierList)
	return root, nil
}

var statementMatcher = astutil.NodeMatcher{
	ExpectTokens: []token.Kind{
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

var parenthesisPrefixMatcher = astutil.NodeMatcher{
	ExpectTokens: []token.Kind{
		token.LParen,
	},
}
var parenthesisCloseMatcher = astutil.NodeMatcher{
	ExpectTokens: []token.Kind{
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

var functionPrefixMatcher = astutil.NodeMatcher{
	ExpectSQLType: []dialect.KeywordKind{
		dialect.Matched,
		dialect.Unmatched,
	},
}
var functionArgsMatcher = astutil.NodeMatcher{
	NodeTypeMatcherFunc: func(node interface{}) bool {
		if _, ok := node.(*ast.Parenthesis); ok {
			return true
		}
		return false
	},
}

func parseFunctions(reader *nodeReader) ast.Node {
	funcName := reader.curNode
	if _, funcArgs := reader.matchedPeekNode(false, functionArgsMatcher); funcArgs != nil {
		function := &ast.FunctionLiteral{Toks: []ast.Node{funcName, funcArgs}}
		reader.nextNode(false)
		return function
	}
	return reader.curNode
}

var wherePrefixMatcher = astutil.NodeMatcher{
	ExpectKeyword: []string{
		"WHERE",
	},
}
var whereCloseMatcher = astutil.NodeMatcher{
	ExpectTokens: []token.Kind{
		token.RParen,
	},
	ExpectKeyword: []string{
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
	whereExpr := reader.curNode
	nodes := []ast.Node{whereExpr}

	for reader.nextNode(false) {
		nodes = append(nodes, reader.curNode)
		if reader.peekNodeIs(false, whereCloseMatcher) {
			fmt.Println(reader.peekNode(false))
			return &ast.WhereClause{Toks: nodes}
		}
	}
	return &ast.WhereClause{Toks: nodes}
}

var JoinPrefixMatcher = astutil.NodeMatcher{
	ExpectKeyword: []string{
		"JOIN",
	},
}
var JoinCloseMatcher = astutil.NodeMatcher{
	ExpectTokens: []token.Kind{
		token.RParen,
	},
	ExpectKeyword: []string{
		"ON",
		"WHERE",
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

func parseJoin(reader *nodeReader) ast.Node {
	fromExpr := reader.curNode
	nodes := []ast.Node{fromExpr}

	for reader.nextNode(false) {
		nodes = append(nodes, reader.curNode)
		if reader.peekNodeIs(false, JoinCloseMatcher) {
			fmt.Println(reader.peekNode(false))
			return &ast.JoinClause{Toks: nodes}
		}
	}
	return &ast.JoinClause{Toks: nodes}
}

var FromPrefixMatcher = astutil.NodeMatcher{
	ExpectKeyword: []string{
		"FROM",
	},
}
var FromCloseMatcher = astutil.NodeMatcher{
	ExpectTokens: []token.Kind{
		token.RParen,
	},
	ExpectKeyword: []string{
		// for join prefix
		"LEFT",
		"RIGHT",
		"INNER",
		"OUTER",
		"CROSS",
		"JOIN",
		// for other expression
		"WHERE",
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

func parseFrom(reader *nodeReader) ast.Node {
	fromExpr := reader.curNode
	nodes := []ast.Node{fromExpr}

	for reader.nextNode(false) {
		nodes = append(nodes, reader.curNode)
		if reader.peekNodeIs(false, FromCloseMatcher) {
			fmt.Println(reader.peekNode(false))
			return &ast.FromClause{Toks: nodes}
		}
	}
	return &ast.FromClause{Toks: nodes}
}

var memberIdentifierInfixMatcher = astutil.NodeMatcher{
	ExpectTokens: []token.Kind{
		token.Period,
	},
}
var memberIdentifierTargetMatcher = astutil.NodeMatcher{
	ExpectTokens: []token.Kind{
		token.Mult,
	},
	ExpectSQLType: []dialect.KeywordKind{
		dialect.Unmatched,
	},
}

func parseMemberIdentifier(reader *nodeReader) ast.Node {
	if !reader.curNodeIs(memberIdentifierTargetMatcher) {
		return reader.curNode
	}
	parent := reader.curNode
	startIndex := reader.index - 1
	memberIdentifier := &ast.MemberIdentifer{Toks: reader.nodesWithRange(startIndex, reader.index+1)}
	reader.nextNode(false)

	endIndex, child := reader.matchedPeekNode(true, memberIdentifierTargetMatcher)
	if child == nil {
		return memberIdentifier
	}
	reader.nextNode(false)

	memberIdentifier = &ast.MemberIdentifer{
		Toks:   reader.nodesWithRange(startIndex, endIndex+1),
		Parent: parent,
		Child:  child,
	}
	return memberIdentifier
}

// parseArrays

var identifierPrefixMatcher = astutil.NodeMatcher{
	ExpectSQLType: []dialect.KeywordKind{
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

var operatorInfixMatcher = astutil.NodeMatcher{
	ExpectTokens: []token.Kind{
		token.Plus,
		token.Minus,
		token.Mult,
		token.Div,
		token.Mod,
	},
}
var operatorTargetMatcher = astutil.NodeMatcher{
	NodeTypeMatcherFunc: func(node interface{}) bool {
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
		if _, ok := node.(*ast.FunctionLiteral); ok {
			return true
		}
		return false
	},
	ExpectTokens: []token.Kind{
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

var comparisonInfixMatcher = astutil.NodeMatcher{
	ExpectTokens: []token.Kind{
		token.Eq,
		token.Neq,
		token.Lt,
		token.Gt,
		token.LtEq,
		token.GtEq,
	},
}
var comparisonTargetMatcher = astutil.NodeMatcher{
	NodeTypeMatcherFunc: func(node interface{}) bool {
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
		if _, ok := node.(*ast.FunctionLiteral); ok {
			return true
		}
		return false
	},
	ExpectTokens: []token.Kind{
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

var aliasInfixMatcher = astutil.NodeMatcher{
	ExpectKeyword: []string{
		"AS",
	},
}

var aliasTargetMatcher = astutil.NodeMatcher{
	NodeTypeMatcherFunc: func(node interface{}) bool {
		if _, ok := node.(*ast.Parenthesis); ok {
			return true
		}
		if _, ok := node.(*ast.FunctionLiteral); ok {
			return true
		}
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
	realName := reader.curNode
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

	return &ast.Aliased{
		Toks:        reader.nodesWithRange(startIndex, endIndex+1),
		RealName:    realName,
		AliasedName: aliasedName,
	}
}

// parseAssignment
// alignComments

var identifierListInfixMatcher = astutil.NodeMatcher{
	ExpectTokens: []token.Kind{
		token.Comma,
	},
}
var identifierListTargetMatcher = astutil.NodeMatcher{
	NodeTypeMatcherFunc: func(node interface{}) bool {
		if _, ok := node.(*ast.FunctionLiteral); ok {
			return true
		}
		if _, ok := node.(*ast.Identifer); ok {
			return true
		}
		if _, ok := node.(*ast.MemberIdentifer); ok {
			return true
		}
		if _, ok := node.(*ast.Aliased); ok {
			return true
		}
		if _, ok := node.(*ast.Comparison); ok {
			return true
		}
		if _, ok := node.(*ast.Operator); ok {
			return true
		}
		return false
	},
}

func parseIdentifierList(reader *nodeReader) ast.Node {
	if !reader.curNodeIs(identifierListTargetMatcher) {
		return reader.curNode
	}
	startIndex := reader.index - 1
	tmpReader := reader.copyReader()
	tmpReader.nextNode(true)

	var endIndex uint
	var count uint
	for {
		tmpIndex, ident := tmpReader.matchedPeekNode(true, identifierListTargetMatcher)
		if ident == nil {
			if count > 0 {
				break
			}
			return tmpReader.curNode
		}
		count++
		tmpReader.nextNode(true)
		endIndex = tmpIndex

		_, nextIdent := tmpReader.matchedPeekNode(true, identifierListInfixMatcher)
		if nextIdent == nil {
			break
		}
		tmpReader.nextNode(true)
	}

	reader.index = tmpReader.index
	reader.curNode = tmpReader.curNode
	return &ast.IdentiferList{Toks: reader.nodesWithRange(startIndex, endIndex+1)}
}

// parseValues
