package parser

import (
	"io"

	"github.com/lighttiger2505/sqls/ast"
	"github.com/lighttiger2505/sqls/dialect"
	"github.com/lighttiger2505/sqls/token"
	"github.com/pkg/errors"
)

type nodeTypeMatcher interface {
	match(node interface{}) bool
}

type finder struct {
	matcher       nodeTypeMatcher
	expectTokens  []token.Kind
	expectSQLType []dialect.KeywordKind
	expectKeyword []string
}

func (f *finder) isMatchNodeType(node interface{}) bool {
	if f.matcher != nil {
		if f.matcher.match(node) {
			return true
		}
	}
	return false
}

func (f *finder) isMatchTokens(tok *ast.SQLToken) bool {
	if f.expectTokens != nil {
		for _, expect := range f.expectTokens {
			if tok.MatchKind(expect) {
				return true
			}
		}
	}
	return false
}

func (f *finder) isMatchSQLType(tok *ast.SQLToken) bool {
	if f.expectSQLType != nil {
		for _, expect := range f.expectSQLType {
			if tok.MatchSQLKind(expect) {
				return true
			}
		}
	}
	return false
}

func (f *finder) isMatchKeyword(tok *ast.SQLToken) bool {
	if f.expectKeyword != nil {
		for _, expect := range f.expectKeyword {
			if tok.MatchSQLKeyword(expect) {
				return true
			}
		}
	}
	return false
}

type nodeWalkContext struct {
	node    ast.TokenList
	curNode ast.Node
	index   uint
}

func newNodeWalkContext(list ast.TokenList) *nodeWalkContext {
	ctx := &nodeWalkContext{
		node: list,
	}
	return ctx
}

func newNodeWalkContextWithIndex(list ast.TokenList, index uint) *nodeWalkContext {
	ctx := &nodeWalkContext{
		node:  list,
		index: index,
	}
	return ctx
}

func (ctx *nodeWalkContext) nodesWithRange(startIndex, endIndex uint) []ast.Node {
	return ctx.node.GetTokens()[startIndex:endIndex]
}

func (ctx *nodeWalkContext) replaceIndex(add ast.Node, index uint) {
	ctx.node.GetTokens()[index] = add
}

func (ctx *nodeWalkContext) replace(add ast.Node, startIndex, endIndex uint) {
	oldList := ctx.node.GetTokens()

	start := oldList[:startIndex]
	end := oldList[endIndex:]

	var out []ast.Node
	out = append(out, start...)
	out = append(out, add)
	out = append(out, end...)
	ctx.node.SetTokens(out)

	offset := (endIndex - startIndex)
	ctx.index = ctx.index - uint(offset)
	ctx.nextNode()
}

func (ctx *nodeWalkContext) hasNext() bool {
	return ctx.index < uint(len(ctx.node.GetTokens()))
}

func (ctx *nodeWalkContext) nextNode() bool {
	if !ctx.hasNext() {
		return false
	}
	ctx.curNode = ctx.node.GetTokens()[ctx.index]
	ctx.index++
	return true
}

func (ctx *nodeWalkContext) curNodeIs(fd finder) (uint, ast.Node) {
	index := ctx.index
	node := ctx.curNode
	if node != nil {
		return 0, nil
	}
	// For node object
	if fd.isMatchNodeType(node) {
		return index, node
	}
	// For token object
	tok, _ := ctx.curNode.(ast.Token)
	sqlTok := tok.GetToken()
	if fd.isMatchTokens(sqlTok) || fd.isMatchSQLType(sqlTok) || fd.isMatchSQLType(sqlTok) {
		return index, node
	}
	return 0, nil
}

func (ctx *nodeWalkContext) peekNode(ignoreWhiteSpace bool) (uint, ast.Node) {
	newCtx := newNodeWalkContextWithIndex(ctx.node, ctx.index)
	for newCtx.hasNext() {
		newCtx.nextNode()
		node := newCtx.node.GetTokens()[newCtx.index]

		if newCtx.hasTokenList() {
			return newCtx.index, node
		}

		tok, _ := newCtx.curNode.(ast.Token)
		if ignoreWhiteSpace && !tok.GetToken().MatchKind(token.Whitespace) {
			return newCtx.index, node
		}
	}
	return 0, nil
}

func (ctx *nodeWalkContext) peekNodeIs(ignoreWhiteSpace bool, fd finder) (uint, ast.Node) {
	index, node := ctx.peekNode(ignoreWhiteSpace)
	if node != nil {
		return 0, nil
	}
	// For node object
	if fd.isMatchNodeType(node) {
		return index, node
	}
	// For token object
	tok, _ := ctx.curNode.(ast.Token)
	sqlTok := tok.GetToken()
	if fd.isMatchTokens(sqlTok) || fd.isMatchSQLType(sqlTok) || fd.isMatchSQLType(sqlTok) {
		return index, node
	}
	return 0, nil
}

func (ctx *nodeWalkContext) findNode(fd finder) (uint, ast.Node) {
	newCtx := newNodeWalkContextWithIndex(ctx.node, ctx.index)
	for newCtx.hasNext() {
		newCtx.nextNode()
		node := newCtx.node.GetTokens()[newCtx.index]

		// For node object
		if fd.isMatchNodeType(node) {
			return newCtx.index, node
		}
		if newCtx.hasTokenList() {
			continue
		}
		// For token object
		tok, _ := ctx.curNode.(ast.Token)
		sqlTok := tok.GetToken()
		if fd.isMatchTokens(sqlTok) || fd.isMatchSQLType(sqlTok) || fd.isMatchSQLType(sqlTok) {
			return newCtx.index, node
		}
	}
	return 0, nil
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

func (ctx *nodeWalkContext) hasToken() bool {
	_, ok := ctx.curNode.(ast.Token)
	return ok
}

func (ctx *nodeWalkContext) getToken() (*ast.SQLToken, error) {
	if !ctx.hasToken() {
		return nil, errors.Errorf("want Token got %T", ctx.curNode)
	}
	token, _ := ctx.curNode.(ast.Token)
	return token.GetToken(), nil
}

func (ctx *nodeWalkContext) mustToken() *ast.SQLToken {
	token, _ := ctx.getToken()
	return token
}

func (ctx *nodeWalkContext) getPeekNode() ast.Node {
	if !ctx.hasNext() {
		return nil
	}
	return ctx.node.GetTokens()[ctx.index]
}

func (ctx *nodeWalkContext) getPeekToken() (*ast.SQLToken, error) {
	if !ctx.hasNext() {
		return nil, errors.Errorf("EOF")
	}
	tok, ok := ctx.node.GetTokens()[ctx.index].(ast.Token)
	if !ok {
		return nil, errors.Errorf("want Token got %T", ctx.curNode)
	}
	return tok.GetToken(), nil
}

func (ctx *nodeWalkContext) peekTokenMatchKind(expect token.Kind) bool {
	token, err := ctx.getPeekToken()
	if err != nil {
		return false
	}
	return token.MatchKind(expect)
}

func (ctx *nodeWalkContext) peekTokenMatchSQLKind(expect dialect.KeywordKind) bool {
	token, err := ctx.getPeekToken()
	if err != nil {
		return false
	}
	return token.MatchSQLKind(expect)
}

func (ctx *nodeWalkContext) peekTokenMatchSQLKeyword(expect string) bool {
	token, err := ctx.getPeekToken()
	if err != nil {
		return false
	}
	return token.MatchSQLKeyword(expect)
}

func (ctx *nodeWalkContext) peekTokenMatchSQLKeywords(expects []string) bool {
	token, err := ctx.getPeekToken()
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
	root = parseStatement(newNodeWalkContext(root))
	root = parseParenthesis(newNodeWalkContext(root))
	root = parseFunctions(newNodeWalkContext(root))
	root = parseWhere(newNodeWalkContext(root))
	root = parsePeriod(newNodeWalkContext(root))
	root = parseIdentifier(newNodeWalkContext(root))
	root = parseOperator(newNodeWalkContext(root))
	root = parseAliased(newNodeWalkContext(root))
	return root, nil
}

func parseStatement(ctx *nodeWalkContext) ast.TokenList {
	var replaceNodes []ast.Node
	var startIndex uint
	for ctx.nextNode() {
		if ctx.hasTokenList() {
			list := ctx.mustTokenList()
			replaceNodes = append(replaceNodes, parseStatement(newNodeWalkContext(list)))
			continue
		}

		tok := ctx.mustToken()
		if tok.MatchKind(token.Semicolon) {
			stmt := &ast.Statement{Toks: ctx.nodesWithRange(startIndex, ctx.index)}
			replaceNodes = append(replaceNodes, stmt)
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

func parseParenthesis(ctx *nodeWalkContext) ast.TokenList {
	var replaceNodes []ast.Node

	for ctx.nextNode() {
		if ctx.hasTokenList() {
			list := ctx.mustTokenList()
			replaceNodes = append(replaceNodes, parseParenthesis(newNodeWalkContext(list)))
			continue
		}

		tok := ctx.mustToken()
		if tok.MatchKind(token.LParen) {
			newctx := newNodeWalkContextWithIndex(ctx.node, ctx.index)
			parenthesis := findParenthesisMatch(newctx, ctx.curNode, ctx.index)
			if parenthesis != nil {
				ctx = newctx
				replaceNodes = append(replaceNodes, parenthesis)
			} else {
				replaceNodes = append(replaceNodes, ctx.curNode)
			}
		} else {
			replaceNodes = append(replaceNodes, ctx.curNode)
		}
	}
	ctx.node.SetTokens(replaceNodes)
	return ctx.node
}

func findParenthesisMatch(ctx *nodeWalkContext, startTok ast.Node, startIndex uint) ast.Node {
	var nodes []ast.Node
	nodes = append(nodes, startTok)
	for ctx.nextNode() {
		if ctx.hasTokenList() {
			continue
		}

		tok := ctx.mustToken()
		if tok.MatchKind(token.LParen) {
			group := findParenthesisMatch(ctx, ctx.curNode, ctx.index)
			nodes = append(nodes, group)
		} else if tok.MatchKind(token.RParen) {
			nodes = append(nodes, ctx.curNode)
			return &ast.Parenthesis{Toks: nodes}
		} else {
			nodes = append(nodes, ctx.curNode)
		}
	}
	return nil
}

// parseCase
// parseIf
// parseFor
// parseBegin

func parseFunctions(ctx *nodeWalkContext) ast.TokenList {
	var replaceNodes []ast.Node

	for ctx.nextNode() {
		if ctx.hasTokenList() {
			list := ctx.mustTokenList()
			replaceNodes = append(replaceNodes, parseFunctions(newNodeWalkContext(list)))
			continue
		}

		tok := ctx.mustToken()
		if tok.MatchSQLKind(dialect.Matched) || tok.MatchSQLKind(dialect.Unmatched) {
			peekNode := ctx.getPeekNode()
			if _, ok := peekNode.(*ast.Parenthesis); ok {
				funcName := ctx.curNode
				ctx.nextNode()
				args := ctx.curNode
				funcNode := &ast.Function{Toks: []ast.Node{funcName, args}}
				replaceNodes = append(replaceNodes, funcNode)
				continue
			}
		}
		replaceNodes = append(replaceNodes, ctx.curNode)
	}
	ctx.node.SetTokens(replaceNodes)
	return ctx.node
}

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

func parseWhere(ctx *nodeWalkContext) ast.TokenList {
	var replaceNodes []ast.Node

	for ctx.nextNode() {
		if ctx.hasTokenList() {
			list := ctx.mustTokenList()
			replaceNodes = append(replaceNodes, parseWhere(newNodeWalkContext(list)))
			continue
		}

		tok := ctx.mustToken()
		if tok.MatchSQLKeyword(WhereOpenKeyword) {
			where := findWhereMatch(ctx, ctx.curNode, ctx.index)
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
	for ctx.nextNode() {
		if ctx.hasTokenList() {
			continue
		}

		tok := ctx.mustToken()
		if tok.MatchSQLKeyword(WhereOpenKeyword) {
			group := findWhereMatch(ctx, ctx.curNode, ctx.index)
			nodes = append(nodes, group)
		} else if ctx.peekTokenMatchSQLKeywords(WhereCloseKeywords) {
			nodes = append(nodes, ctx.curNode)
			return &ast.Where{Toks: nodes}
		} else {
			nodes = append(nodes, ctx.curNode)
		}
		if ctx.peekTokenMatchKind(token.RParen) {
			break
		}
	}
	return &ast.Where{Toks: nodes}
}

func parsePeriod(ctx *nodeWalkContext) ast.TokenList {
	var replaceNodes []ast.Node
	for ctx.nextNode() {
		if ctx.hasTokenList() {
			list := ctx.mustTokenList()
			replaceNodes = append(replaceNodes, parsePeriod(newNodeWalkContext(list)))
			continue
		}

		tok := ctx.mustToken()
		if ctx.peekTokenMatchKind(token.Period) {
			memberIdentifer := &ast.MemberIdentifer{
				Parent: tok,
			}
			ctx.nextNode()
			period := ctx.mustToken()
			memberIdentifer.Period = period

			if ctx.peekTokenMatchSQLKind(dialect.Unmatched) || ctx.peekTokenMatchKind(token.Mult) {
				ctx.nextNode()
				child := ctx.mustToken()
				memberIdentifer.Child = child
			}
			replaceNodes = append(replaceNodes, memberIdentifer)
		} else {
			replaceNodes = append(replaceNodes, ctx.curNode)
		}
	}
	ctx.node.SetTokens(replaceNodes)
	return ctx.node
}

// parseArrays

func parseIdentifier(ctx *nodeWalkContext) ast.TokenList {
	var replaceNodes []ast.Node
	for ctx.nextNode() {
		if ctx.hasTokenList() {
			list := ctx.mustTokenList()
			if _, ok := list.(*ast.MemberIdentifer); ok {
				replaceNodes = append(replaceNodes, ctx.curNode)
				continue
			}
			replaceNodes = append(replaceNodes, parseIdentifier(newNodeWalkContext(list)))
			continue
		}

		tok := ctx.mustToken()
		if tok.MatchSQLKind(dialect.Unmatched) {
			identifer := &ast.Identifer{Tok: tok}
			replaceNodes = append(replaceNodes, identifer)
		} else {
			replaceNodes = append(replaceNodes, ctx.curNode)
		}
	}
	ctx.node.SetTokens(replaceNodes)
	return ctx.node
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

type operatorMatcher struct{}

func (om *operatorMatcher) match(node interface{}) bool {
	if _, ok := node.(*ast.Identifer); ok {
		return true
	}
	return false
}

var operatorFinder = finder{
	expectTokens: []token.Kind{
		token.Plus,
		token.Minus,
		token.Mult,
		token.Div,
		token.Mod,
	},
}
var operatorTargetFinder = finder{
	matcher: &operatorMatcher{},
	expectTokens: []token.Kind{
		token.Number,
		token.Char,
		token.SingleQuotedString,
		token.NationalStringLiteral,
	},
}

func parseOperator(ctx *nodeWalkContext) ast.TokenList {
	var replaceNodes []ast.Node

	for ctx.nextNode() {
		if ctx.hasTokenList() {
			list := ctx.mustTokenList()
			replaceNodes = append(replaceNodes, parseOperator(newNodeWalkContext(list)))
			continue
		}

		tok, err := ctx.getToken()
		if err != nil {
			// FIXME workaround
			continue
		}

		if !isMatchKindOfOpeTarget(tok) && !isMatchOperatorNodeType(ctx.curNode) {
			replaceNodes = append(replaceNodes, ctx.curNode)
			continue
		}
		ptok, _ := ctx.getPeekToken()
		if ptok != nil {
			if !isMatchKindOfOperator(ptok) {
				replaceNodes = append(replaceNodes, ctx.curNode)
				continue
			}
			left := ctx.curNode
			op := ctx.getPeekNode()
			newCtx := newNodeWalkContextWithIndex(ctx.node, ctx.index)

			newCtx.nextNode()
			nextPTok, _ := newCtx.getPeekToken()
			if !isMatchKindOfOpeTarget(nextPTok) && !isMatchOperatorNodeType(newCtx.getPeekNode()) {
				replaceNodes = append(replaceNodes, ctx.curNode)
				continue
			}
			right := newCtx.getPeekNode()
			newCtx.nextNode()
			newCtx.nextNode()
			ctx = newCtx

			operator := &ast.Operator{}
			operator.SetTokens([]ast.Node{left, op, right})
			replaceNodes = append(replaceNodes, operator)
		} else {
			replaceNodes = append(replaceNodes, ctx.curNode)
		}

	}
	ctx.node.SetTokens(replaceNodes)
	return ctx.node
}

var operatorKinds = []token.Kind{
	token.Number,
	token.Char,
	token.SingleQuotedString,
	token.NationalStringLiteral,
}

func isMatchKindOfOpeTarget(tok *ast.SQLToken) bool {
	for _, op := range operatorKinds {
		if tok.MatchKind(op) {
			return true
		}
	}
	return false
}

var operators = []token.Kind{
	token.Plus,
	token.Minus,
	token.Mult,
	token.Div,
	token.Mod,
}

func isMatchKindOfOperator(tok *ast.SQLToken) bool {
	for _, op := range operators {
		if tok.MatchKind(op) {
			return true
		}
	}
	return false
}

func isMatchOperatorNodeType(node interface{}) bool {
	if _, ok := node.(*ast.Identifer); ok {
		return true
	}
	return false
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

func parseAliased(ctx *nodeWalkContext) ast.TokenList {
	var replaceNodes []ast.Node
	for ctx.nextNode() {
		if ctx.hasTokenList() {
			list := ctx.mustTokenList()
			replaceNodes = append(replaceNodes, parseAliased(newNodeWalkContext(list)))
			continue
		}

		if _, ok := ctx.curNode.(*ast.Identifer); ok {
			newWC := newNodeWalkContextWithIndex(ctx.node, ctx.index)
			aliased := findAliasMatch(newWC, ctx.curNode, ctx.index)
			if aliased != nil {
				ctx = newWC
				replaceNodes = append(replaceNodes, aliased)
			} else {
				replaceNodes = append(replaceNodes, ctx.curNode)
			}
		} else {
			replaceNodes = append(replaceNodes, ctx.curNode)
		}
	}
	ctx.node.SetTokens(replaceNodes)
	return ctx.node
}

func findAliasMatch(ctx *nodeWalkContext, startTok ast.Node, startIndex uint) ast.Node {
	var nodes []ast.Node
	nodes = append(nodes, startTok)
	for ctx.nextNode() {
		if ctx.hasTokenList() {
			continue
		}

		if _, ok := ctx.curNode.(*ast.Identifer); ok {
			nodes = append(nodes, ctx.curNode)
			return &ast.Aliased{Toks: nodes}
		}

		tok := ctx.mustToken()
		if tok.MatchSQLKeyword("AS") || tok.MatchKind(token.Whitespace) {
			nodes = append(nodes, ctx.curNode)
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
