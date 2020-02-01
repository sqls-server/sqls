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

type nodeWalkContext struct {
	node    ast.TokenList
	curNode ast.Node
	index   uint
}

func newNodeWalkContext(list ast.TokenList) *nodeWalkContext {
	return &nodeWalkContext{
		node: list,
	}
}

func (ctx *nodeWalkContext) copyContext() *nodeWalkContext {
	return &nodeWalkContext{
		node:  ctx.node,
		index: ctx.index,
	}
}

func (ctx *nodeWalkContext) nodesWithRange(startIndex, endIndex uint) []ast.Node {
	return ctx.node.GetTokens()[startIndex:endIndex]
}

func (ctx *nodeWalkContext) hasNext() bool {
	return ctx.index < uint(len(ctx.node.GetTokens()))
}

func (ctx *nodeWalkContext) nextNode(ignoreWhiteSpace bool) bool {
	if !ctx.hasNext() {
		return false
	}
	ctx.curNode = ctx.node.GetTokens()[ctx.index]
	ctx.index++

	if ignoreWhiteSpace && isWhitespace(ctx.curNode) {
		return ctx.nextNode(ignoreWhiteSpace)
	}
	return true
}

func (ctx *nodeWalkContext) curNodeIs(fd nodeMatcher) (uint, ast.Node) {
	index := ctx.index - 1
	node := ctx.curNode
	if node != nil {
		if fd.isMatch(node) {
			return index, node
		}
	}
	return 0, nil
}

func (ctx *nodeWalkContext) peekNode(ignoreWhiteSpace bool) (uint, ast.Node) {
	newCtx := ctx.copyContext()
	for newCtx.hasNext() {
		index := newCtx.index
		node := newCtx.node.GetTokens()[index]

		if ignoreWhiteSpace {
			if !isWhitespace(node) {
				return index, node
			}
		} else {
			return index, node
		}
		newCtx.nextNode(false)
	}
	return 0, nil
}

func (ctx *nodeWalkContext) peekNodeIs(ignoreWhiteSpace bool, fd nodeMatcher) (uint, ast.Node) {
	index, node := ctx.peekNode(ignoreWhiteSpace)
	if node != nil {
		if fd.isMatch(node) {
			return index, node
		}
	}
	return 0, nil
}

func (ctx *nodeWalkContext) findNode(ignoreWhiteSpace bool, fd nodeMatcher) (*nodeWalkContext, ast.Node) {
	newCtx := ctx.copyContext()
	for newCtx.hasNext() {
		node := newCtx.node.GetTokens()[newCtx.index]

		// For node object
		if fd.isMatchNodeType(node) {
			return newCtx, node
		}
		if newCtx.hasTokenList() {
			continue
		}
		// For token object
		tok, _ := ctx.curNode.(ast.Token)
		sqlTok := tok.GetToken()
		if fd.isMatchTokens(sqlTok) || fd.isMatchSQLType(sqlTok) || fd.isMatchKeyword(sqlTok) {
			return newCtx, node
		}
		newCtx.nextNode(ignoreWhiteSpace)
	}
	return nil, nil
}

func (ctx *nodeWalkContext) hasTokenList() bool {
	_, ok := ctx.curNode.(ast.TokenList)
	return ok
}

func (ctx *nodeWalkContext) getTokenList() (ast.TokenList, error) {
	if !ctx.hasTokenList() {
		return nil, errors.Errorf("want TokenList got %T", ctx.curNode)
	}
	children, _ := ctx.curNode.(ast.TokenList)
	return children, nil
}

func (ctx *nodeWalkContext) mustTokenList() ast.TokenList {
	children, _ := ctx.getTokenList()
	return children
}

type prefixParseFn func(ctx *nodeWalkContext) ast.Node

func parsePrefixGroup(ctx *nodeWalkContext, matcher nodeMatcher, fn prefixParseFn) ast.TokenList {
	var replaceNodes []ast.Node
	for ctx.nextNode(false) {
		if list, ok := ctx.curNode.(ast.TokenList); ok {
			newCtx := newNodeWalkContext(list)
			replaceNodes = append(replaceNodes, parsePrefixGroup(newCtx, matcher, fn))
			continue
		}

		if _, node := ctx.curNodeIs(matcher); node != nil {
			replaceNodes = append(replaceNodes, fn(ctx))
		} else {
			replaceNodes = append(replaceNodes, ctx.curNode)
		}
	}
	ctx.node.SetTokens(replaceNodes)
	return ctx.node
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
	root = parseStatement(newNodeWalkContext(root))
	root = parsePrefixGroup(newNodeWalkContext(root), parenthesisPrefixMatcher, parseParenthesis)
	root = parsePrefixGroup(newNodeWalkContext(root), functionPrefixMatcher, parseFunctions)
	root = parseWhere(newNodeWalkContext(root))
	root = parseMemberIdentifier(newNodeWalkContext(root))
	root = parsePrefixGroup(newNodeWalkContext(root), identifierPrefixMatcher, parseIdentifier)
	root = parseOperator(newNodeWalkContext(root))
	root = parseAliased(newNodeWalkContext(root))
	return root, nil
}

var statementMatcher = nodeMatcher{
	expectTokens: []token.Kind{
		token.Semicolon,
	},
}

func parseStatement(ctx *nodeWalkContext) ast.TokenList {
	var replaceNodes []ast.Node
	var startIndex uint
	for ctx.nextNode(false) {
		if ctx.hasTokenList() {
			list := ctx.mustTokenList()
			replaceNodes = append(replaceNodes, parseStatement(newNodeWalkContext(list)))
			continue
		}

		newCtx, node := ctx.findNode(true, statementMatcher)
		if node != nil {
			stmt := &ast.Statement{Toks: ctx.nodesWithRange(startIndex, newCtx.index)}
			replaceNodes = append(replaceNodes, stmt)
			ctx = newCtx
			startIndex = ctx.index
		}
	}
	if ctx.index != startIndex {
		stmt := &ast.Statement{Toks: ctx.nodesWithRange(startIndex, ctx.index)}
		replaceNodes = append(replaceNodes, stmt)
	}
	ctx.node.SetTokens(replaceNodes)
	return ctx.node
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

func parseParenthesis(ctx *nodeWalkContext) ast.Node {
	nodes := []ast.Node{ctx.curNode}
	tmpCtx := ctx.copyContext()
	for tmpCtx.nextNode(false) {
		if tmpCtx.hasTokenList() {
			continue
		}

		if _, node := tmpCtx.curNodeIs(parenthesisPrefixMatcher); node != nil {
			parenthesis := parseParenthesis(tmpCtx)
			nodes = append(nodes, parenthesis)
		} else if _, node := tmpCtx.curNodeIs(parenthesisCloseMatcher); node != nil {
			ctx.index = tmpCtx.index
			ctx.curNode = tmpCtx.curNode
			return &ast.Parenthesis{Toks: append(nodes, node)}
		} else {
			nodes = append(nodes, tmpCtx.curNode)
		}
	}
	return ctx.curNode
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

func parseFunctions(ctx *nodeWalkContext) ast.Node {
	funcName := ctx.curNode
	if _, funcArgs := ctx.peekNodeIs(false, functionArgsMatcher); funcArgs != nil {
		function := &ast.Function{Toks: []ast.Node{funcName, funcArgs}}
		ctx.nextNode(false)
		return function
	}
	return ctx.curNode
}

var whereOpenMatcher = nodeMatcher{
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

func parseWhere(ctx *nodeWalkContext) ast.TokenList {
	var replaceNodes []ast.Node

	for ctx.nextNode(false) {
		if ctx.hasTokenList() {
			list := ctx.mustTokenList()
			replaceNodes = append(replaceNodes, parseWhere(newNodeWalkContext(list)))
			continue
		}

		if index, whereOpener := ctx.curNodeIs(whereOpenMatcher); whereOpener != nil {
			where := findWhereMatch(ctx, whereOpener, index)
			if where != nil {
				replaceNodes = append(replaceNodes, where)
			}
		} else {
			replaceNodes = append(replaceNodes, ctx.curNode)
		}
	}
	ctx.node.SetTokens(replaceNodes)
	return ctx.node
}

func findWhereMatch(ctx *nodeWalkContext, startTok ast.Node, startIndex uint) ast.Node {
	var nodes []ast.Node
	nodes = append(nodes, startTok)
	for ctx.nextNode(false) {
		if ctx.hasTokenList() {
			continue
		}

		if index, whereOpener := ctx.curNodeIs(whereOpenMatcher); whereOpener != nil {
			nodes = append(nodes, findWhereMatch(ctx, whereOpener, index))
		} else if _, node := ctx.peekNodeIs(false, whereCloseMatcher); node != nil {
			nodes = append(nodes, ctx.curNode)
			return &ast.Where{Toks: nodes}
		} else {
			nodes = append(nodes, ctx.curNode)
		}
	}
	return &ast.Where{Toks: nodes}
}

var periodMatcher = nodeMatcher{
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

func parseMemberIdentifier(ctx *nodeWalkContext) ast.TokenList {
	var replaceNodes []ast.Node
	for ctx.nextNode(false) {
		if ctx.hasTokenList() {
			list := ctx.mustTokenList()
			replaceNodes = append(replaceNodes, parseMemberIdentifier(newNodeWalkContext(list)))
			continue
		}

		_, period := ctx.peekNodeIs(false, periodMatcher)
		if period != nil {
			startIndex, left := ctx.curNodeIs(memberIdentifierTargetMatcher)
			if left == nil {
				replaceNodes = append(replaceNodes, ctx.curNode)
				continue
			}
			mi := &ast.MemberIdentifer{Toks: ctx.nodesWithRange(startIndex, ctx.index+1)}
			ctx.nextNode(false)

			endIndex, right := ctx.peekNodeIs(true, memberIdentifierTargetMatcher)
			if right == nil {
				replaceNodes = append(replaceNodes, mi)
				continue
			}
			ctx.nextNode(false)

			mi = &ast.MemberIdentifer{Toks: ctx.nodesWithRange(startIndex, endIndex+1)}
			replaceNodes = append(replaceNodes, mi)
		} else {
			replaceNodes = append(replaceNodes, ctx.curNode)
		}
	}
	ctx.node.SetTokens(replaceNodes)
	return ctx.node
}

// parseArrays

var identifierPrefixMatcher = nodeMatcher{
	expectSQLType: []dialect.KeywordKind{
		dialect.Unmatched,
	},
}

func parseIdentifier(ctx *nodeWalkContext) ast.Node {
	token, _ := ctx.curNode.(ast.Token)
	return &ast.Identifer{Tok: token.GetToken()}
}

// parseOrder
// parseTypecasts
// parseTzcasts
// parseTyped_literal

// operatorTypes
// ast.Parenthesis
// ast.Function
// ast.Identifier
var comparisons = []token.Kind{
	token.Eq,
	token.Neq,
	token.Lt,
	token.Gt,
	token.LtEq,
	token.GtEq,
}

type operatorTypeMatcher struct{}

func (om *operatorTypeMatcher) match(node interface{}) bool {
	if _, ok := node.(*ast.Identifer); ok {
		return true
	}
	return false
}

var operatorMatcher = nodeMatcher{
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
		return false
	},
	expectTokens: []token.Kind{
		token.Number,
		token.Char,
		token.SingleQuotedString,
		token.NationalStringLiteral,
	},
}

func parseOperator(ctx *nodeWalkContext) ast.TokenList {
	var replaceNodes []ast.Node

	for ctx.nextNode(false) {
		if ctx.hasTokenList() {
			list := ctx.mustTokenList()
			replaceNodes = append(replaceNodes, parseOperator(newNodeWalkContext(list)))
			continue
		}

		_, ope := ctx.peekNodeIs(true, operatorMatcher)
		if ope != nil {
			startIndex, left := ctx.curNodeIs(operatorTargetMatcher)
			if left == nil {
				replaceNodes = append(replaceNodes, ctx.curNode)
				continue
			}
			tmpCtx := ctx.copyContext()
			tmpCtx.nextNode(true)

			endIndex, right := tmpCtx.peekNodeIs(true, operatorTargetMatcher)
			if right == nil {
				replaceNodes = append(replaceNodes, ctx.curNode)
				continue
			}
			tmpCtx.nextNode(true)
			tmpCtx.nextNode(true)
			ctx = tmpCtx

			operator := &ast.Operator{Toks: ctx.nodesWithRange(startIndex, endIndex+1)}
			replaceNodes = append(replaceNodes, operator)
		} else {
			replaceNodes = append(replaceNodes, ctx.curNode)
		}
	}
	ctx.node.SetTokens(replaceNodes)
	return ctx.node
}

func parseComparison(ctx *nodeWalkContext) ast.TokenList {
	// sql.Parenthesis
	// sql.Function
	// sql.Identifier

	// T_NUMERICAL = (T.Number, T.Number.Integer, T.Number.Float)
	// T_STRING = (T.String, T.String.Single, T.String.Symbol)
	// T_NAME = (T.Name, T.Name.Placeholder)
	return nil
}

// ast.Identifer,
// ast.MemberIdentifer,
// ast.Parenthesis,

var aliasKeywordMatcher = nodeMatcher{
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

func parseAliased(ctx *nodeWalkContext) ast.TokenList {
	var replaceNodes []ast.Node
	for ctx.nextNode(false) {
		if ctx.hasTokenList() {
			list := ctx.mustTokenList()
			replaceNodes = append(replaceNodes, parseAliased(newNodeWalkContext(list)))
			continue
		}

		if _, as := ctx.peekNodeIs(true, aliasKeywordMatcher); as != nil {
			startIndex, realName := ctx.curNodeIs(aliasTargetMatcher)
			if realName == nil {
				replaceNodes = append(replaceNodes, ctx.curNode)
				continue
			}
			tmpCtx := ctx.copyContext()
			tmpCtx.nextNode(true)

			endIndex, aliasedName := tmpCtx.peekNodeIs(true, aliasTargetMatcher)
			if aliasedName == nil {
				replaceNodes = append(replaceNodes, ctx.curNode)
				continue
			}
			tmpCtx.nextNode(true)
			ctx = tmpCtx

			aliased := &ast.Aliased{Toks: ctx.nodesWithRange(startIndex, endIndex+1)}
			replaceNodes = append(replaceNodes, aliased)
		} else {
			replaceNodes = append(replaceNodes, ctx.curNode)
		}
	}
	ctx.node.SetTokens(replaceNodes)
	return ctx.node
}

// parseAssignment
// alignComments
// parseIdentifierList
// parseValues
