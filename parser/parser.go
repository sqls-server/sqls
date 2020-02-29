package parser

import (
	"io"

	"github.com/lighttiger2505/sqls/ast"
	"github.com/lighttiger2505/sqls/ast/astutil"
	"github.com/lighttiger2505/sqls/dialect"
	"github.com/lighttiger2505/sqls/token"
	"github.com/pkg/errors"
)

type (
	prefixParseFn func(reader *astutil.NodeReader) ast.Node
	infixParseFn  func(reader *astutil.NodeReader) ast.Node
)

func parsePrefixGroup(reader *astutil.NodeReader, matcher astutil.NodeMatcher, fn prefixParseFn) ast.TokenList {
	var replaceNodes []ast.Node
	for reader.NextNode(false) {
		if reader.CurNodeIs(matcher) {
			replaceNodes = append(replaceNodes, fn(reader))
		} else if list, ok := reader.CurNode.(ast.TokenList); ok {
			newReader := astutil.NewNodeReader(list)
			replaceNodes = append(replaceNodes, parsePrefixGroup(newReader, matcher, fn))
		} else {
			replaceNodes = append(replaceNodes, reader.CurNode)
		}
	}
	reader.Node.SetTokens(replaceNodes)
	return reader.Node
}

func parseInfixGroup(reader *astutil.NodeReader, matcher astutil.NodeMatcher, ignoreWhiteSpace bool, fn infixParseFn) ast.TokenList {
	var replaceNodes []ast.Node
	for reader.NextNode(false) {
		if reader.PeekNodeIs(ignoreWhiteSpace, matcher) {
			replaceNodes = append(replaceNodes, fn(reader))
		} else if list, ok := reader.CurNode.(ast.TokenList); ok {
			newReader := astutil.NewNodeReader(list)
			replaceNodes = append(replaceNodes, parseInfixGroup(newReader, matcher, ignoreWhiteSpace, fn))
		} else {
			replaceNodes = append(replaceNodes, reader.CurNode)
		}
	}
	reader.Node.SetTokens(replaceNodes)
	return reader.Node
}

type Parser struct {
	root ast.TokenList
}

func NewParser(src io.Reader, d dialect.Dialect) (*Parser, error) {
	tokenizer := token.NewTokenizer(src, d)
	tokens, err := tokenizer.Tokenize()
	if err != nil {
		return nil, errors.Errorf("tokenize err failed: %+v", err)
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
	root = parseStatement(astutil.NewNodeReader(root))

	root = parsePrefixGroup(astutil.NewNodeReader(root), parenthesisPrefixMatcher, parseParenthesis)
	root = parsePrefixGroup(astutil.NewNodeReader(root), functionPrefixMatcher, parseFunctions)
	root = parsePrefixGroup(astutil.NewNodeReader(root), FromPrefixMatcher, parseFrom)
	root = parsePrefixGroup(astutil.NewNodeReader(root), JoinPrefixMatcher, parseJoin)
	root = parsePrefixGroup(astutil.NewNodeReader(root), wherePrefixMatcher, parseWhere)
	root = parsePrefixGroup(astutil.NewNodeReader(root), identifierPrefixMatcher, parseIdentifier)

	root = parseInfixGroup(astutil.NewNodeReader(root), memberIdentifierInfixMatcher, false, parseMemberIdentifier)
	root = parseInfixGroup(astutil.NewNodeReader(root), multiKeywordInfixMatcher, true, parseMultiKeyword)
	root = parseInfixGroup(astutil.NewNodeReader(root), operatorInfixMatcher, true, parseOperator)
	root = parseInfixGroup(astutil.NewNodeReader(root), comparisonInfixMatcher, true, parseComparison)
	root = parseInfixGroup(astutil.NewNodeReader(root), aliasInfixMatcher, true, parseAliased)
	root = parseInfixGroup(astutil.NewNodeReader(root), identifierListInfixMatcher, true, parseIdentifierList)
	return root, nil
}

var statementMatcher = astutil.NodeMatcher{
	ExpectTokens: []token.Kind{
		token.Semicolon,
	},
}

func parseStatement(reader *astutil.NodeReader) ast.TokenList {
	var replaceNodes []ast.Node
	var startIndex int
	for reader.NextNode(false) {
		if list, ok := reader.CurNode.(ast.TokenList); ok {
			replaceNodes = append(replaceNodes, parseStatement(astutil.NewNodeReader(list)))
			continue
		}

		tmpReader, node := reader.FindNode(true, statementMatcher)
		if node != nil {
			stmt := &ast.Statement{Toks: reader.NodesWithRange(startIndex, tmpReader.Index)}
			replaceNodes = append(replaceNodes, stmt)
			reader = tmpReader
			startIndex = reader.Index
		}
	}
	if reader.Index != startIndex {
		stmt := &ast.Statement{Toks: reader.NodesWithRange(startIndex, reader.Index)}
		replaceNodes = append(replaceNodes, stmt)
	}
	reader.Node.SetTokens(replaceNodes)
	return reader.Node
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

func parseParenthesis(reader *astutil.NodeReader) ast.Node {
	nodes := []ast.Node{reader.CurNode}
	tmpReader := reader.CopyReader()
	for tmpReader.NextNode(false) {
		if _, ok := reader.CurNode.(ast.TokenList); ok {
			continue
		}

		if tmpReader.CurNodeIs(parenthesisPrefixMatcher) {
			parenthesis := parseParenthesis(tmpReader)
			nodes = append(nodes, parenthesis)
		} else if tmpReader.CurNodeIs(parenthesisCloseMatcher) {
			reader.Index = tmpReader.Index
			reader.CurNode = tmpReader.CurNode
			return &ast.Parenthesis{Toks: append(nodes, tmpReader.CurNode)}
		} else {
			nodes = append(nodes, tmpReader.CurNode)
		}
	}
	return reader.CurNode
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

func parseFunctions(reader *astutil.NodeReader) ast.Node {
	funcName := reader.CurNode
	if reader.PeekNodeIs(false, functionArgsMatcher) {
		_, funcArgs := reader.PeekNode(false)
		function := &ast.FunctionLiteral{Toks: []ast.Node{funcName, funcArgs}}
		reader.NextNode(false)
		return function
	}
	return reader.CurNode
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

func parseWhere(reader *astutil.NodeReader) ast.Node {
	whereExpr := reader.CurNode
	nodes := []ast.Node{whereExpr}

	for reader.NextNode(false) {
		nodes = append(nodes, reader.CurNode)
		if reader.PeekNodeIs(false, whereCloseMatcher) {
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

func parseJoin(reader *astutil.NodeReader) ast.Node {
	fromExpr := reader.CurNode
	nodes := []ast.Node{fromExpr}

	for reader.NextNode(false) {
		nodes = append(nodes, reader.CurNode)
		if reader.PeekNodeIs(false, JoinCloseMatcher) {
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
var FromRecursionMatcher = astutil.NodeMatcher{
	NodeTypeMatcherFunc: func(node interface{}) bool {
		if _, ok := node.(*ast.Parenthesis); ok {
			return true
		}
		return false
	},
}

func parseFrom(reader *astutil.NodeReader) ast.Node {
	nodes := []ast.Node{reader.CurNode}
	tmpReader := reader.CopyReader()
	for tmpReader.NextNode(false) {
		if _, ok := reader.CurNode.(ast.TokenList); ok {
			continue
		}

		if tmpReader.CurNodeIs(FromRecursionMatcher) {
			// For sub query
			if list, ok := tmpReader.CurNode.(ast.TokenList); ok {
				parenthesis := parsePrefixGroup(astutil.NewNodeReader(list), FromPrefixMatcher, parseFrom)
				nodes = append(nodes, parenthesis)
			} else {
				nodes = append(nodes, tmpReader.CurNode)
			}
		} else if tmpReader.CurNodeIs(FromPrefixMatcher) {
			from := parseFrom(tmpReader)
			nodes = append(nodes, from)
		} else if tmpReader.PeekNodeIs(false, FromCloseMatcher) {
			reader.Index = tmpReader.Index
			reader.CurNode = tmpReader.CurNode
			return &ast.FromClause{Toks: append(nodes, tmpReader.CurNode)}
		} else {
			nodes = append(nodes, tmpReader.CurNode)
		}
	}
	reader.Index = tmpReader.Index
	reader.CurNode = tmpReader.CurNode
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

func parseMemberIdentifier(reader *astutil.NodeReader) ast.Node {
	if !reader.CurNodeIs(memberIdentifierTargetMatcher) {
		return reader.CurNode
	}
	parent := reader.CurNode
	startIndex := reader.Index - 1
	memberIdentifier := &ast.MemberIdentifer{
		Toks:   reader.NodesWithRange(startIndex, reader.Index+1),
		Parent: parent,
	}

	reader.NextNode(false)
	if !reader.PeekNodeIs(true, memberIdentifierTargetMatcher) {
		return memberIdentifier
	}
	endIndex, child := reader.PeekNode(true)
	memberIdentifier = &ast.MemberIdentifer{
		Toks:   reader.NodesWithRange(startIndex, endIndex+1),
		Parent: parent,
		Child:  child,
	}

	reader.NextNode(false)
	return memberIdentifier
}

var multiKeywordInfixMatcher = astutil.NodeMatcher{
	ExpectKeyword: []string{
		"BY",
	},
}
var multiKeywordTargetMatcher = astutil.NodeMatcher{
	ExpectKeyword: []string{
		"ORDER",
		"GROUP",
	},
}

func parseMultiKeyword(reader *astutil.NodeReader) ast.Node {
	if !reader.CurNodeIs(multiKeywordTargetMatcher) {
		return reader.CurNode
	}
	startIndex := reader.Index - 1
	reader.NextNode(true)
	memberIdentifier := &ast.MultiKeyword{
		Toks: reader.NodesWithRange(startIndex, reader.Index),
	}
	return memberIdentifier
}

// parseArrays

var identifierPrefixMatcher = astutil.NodeMatcher{
	ExpectSQLType: []dialect.KeywordKind{
		dialect.Unmatched,
	},
}

func parseIdentifier(reader *astutil.NodeReader) ast.Node {
	token, _ := reader.CurNode.(ast.Token)
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
var operatorRecursionMatcher = astutil.NodeMatcher{
	NodeTypeMatcherFunc: func(node interface{}) bool {
		if _, ok := node.(*ast.Parenthesis); ok {
			return true
		}
		return false
	},
}

func parseOperator(reader *astutil.NodeReader) ast.Node {
	operator := &ast.Operator{Left: reader.CurNode}
	if !reader.CurNodeIs(operatorTargetMatcher) {
		return reader.CurNode
	}
	if reader.CurNodeIs(operatorRecursionMatcher) {
		if list, ok := reader.CurNode.(ast.TokenList); ok {
			// For sub query
			parenthesis := parseInfixGroup(astutil.NewNodeReader(list), operatorInfixMatcher, true, parseOperator)
			reader.Replace(parenthesis, reader.Index-1)
		}
	}
	startIndex := reader.Index - 1
	tmpReader := reader.CopyReader()
	tmpReader.NextNode(true)
	operator.Operator = tmpReader.CurNode

	if !tmpReader.PeekNodeIs(true, operatorTargetMatcher) {
		return reader.CurNode
	}
	endIndex, right := tmpReader.PeekNode(true)
	if tmpReader.PeekNodeIs(true, operatorRecursionMatcher) {
		if list, ok := right.(ast.TokenList); ok {
			// For sub query
			parenthesis := parseInfixGroup(astutil.NewNodeReader(list), operatorInfixMatcher, true, parseOperator)
			reader.Replace(parenthesis, endIndex)
		}
	}
	operator.Right = right

	tmpReader.NextNode(true)
	reader.Index = tmpReader.Index
	reader.CurNode = tmpReader.CurNode

	operator.Toks = reader.NodesWithRange(startIndex, endIndex+1)
	return operator
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
var comparisonRecursionMatcher = astutil.NodeMatcher{
	NodeTypeMatcherFunc: func(node interface{}) bool {
		if _, ok := node.(*ast.Parenthesis); ok {
			return true
		}
		return false
	},
}

func parseComparison(reader *astutil.NodeReader) ast.Node {
	comparison := &ast.Comparison{Left: reader.CurNode}
	if !reader.CurNodeIs(comparisonTargetMatcher) {
		return reader.CurNode
	}
	if reader.CurNodeIs(comparisonRecursionMatcher) {
		if list, ok := reader.CurNode.(ast.TokenList); ok {
			// For sub query
			parenthesis := parseInfixGroup(astutil.NewNodeReader(list), comparisonInfixMatcher, true, parseComparison)
			reader.Replace(parenthesis, reader.Index-1)
		}
	}
	startIndex := reader.Index - 1
	tmpReader := reader.CopyReader()
	tmpReader.NextNode(true)
	comparison.Comparison = tmpReader.CurNode

	if !tmpReader.PeekNodeIs(true, comparisonTargetMatcher) {
		return reader.CurNode
	}
	endIndex, right := tmpReader.PeekNode(true)
	if tmpReader.PeekNodeIs(true, operatorRecursionMatcher) {
		if list, ok := right.(ast.TokenList); ok {
			// For sub query
			parenthesis := parseInfixGroup(astutil.NewNodeReader(list), comparisonInfixMatcher, true, parseComparison)
			reader.Replace(parenthesis, endIndex)
		}
	}
	comparison.Right = right

	tmpReader.NextNode(true)
	reader.Index = tmpReader.Index
	reader.CurNode = tmpReader.CurNode

	comparison.Toks = reader.NodesWithRange(startIndex, endIndex+1)
	return comparison
}

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
		return false
	},
}

var aliasRecursionMatcher = astutil.NodeMatcher{
	NodeTypeMatcherFunc: func(node interface{}) bool {
		if _, ok := node.(*ast.Parenthesis); ok {
			return true
		}
		return false
	},
}

func parseAliased(reader *astutil.NodeReader) ast.Node {
	if !reader.CurNodeIs(aliasTargetMatcher) {
		return reader.CurNode
	}
	if reader.CurNodeIs(aliasRecursionMatcher) {
		if list, ok := reader.CurNode.(ast.TokenList); ok {
			// For sub query
			parenthesis := parseInfixGroup(astutil.NewNodeReader(list), aliasInfixMatcher, true, parseAliased)
			reader.Replace(parenthesis, reader.Index-1)
		}
	}

	realName := reader.CurNode
	startIndex := reader.Index - 1
	tmpReader := reader.CopyReader()
	tmpReader.NextNode(true)

	if !tmpReader.PeekNodeIs(true, aliasTargetMatcher) {
		return reader.CurNode
	}
	endIndex, aliasedName := tmpReader.PeekNode(true)

	tmpReader.NextNode(true)
	reader.Index = tmpReader.Index
	reader.CurNode = tmpReader.CurNode
	return &ast.Aliased{
		Toks:        reader.NodesWithRange(startIndex, endIndex+1),
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

func parseIdentifierList(reader *astutil.NodeReader) ast.Node {
	if !reader.CurNodeIs(identifierListTargetMatcher) {
		return reader.CurNode
	}
	startIndex := reader.Index - 1
	tmpReader := reader.CopyReader()
	tmpReader.NextNode(true)

	var endIndex, peekIndex int
	for {
		if !tmpReader.PeekNodeIs(true, identifierListTargetMatcher) {
			// Include white space after the comma
			peekIndex, peekNode := tmpReader.PeekNode(true)
			if peekNode != nil {
				endIndex = peekIndex - 1
				tmpReader.Index = endIndex + 1
			}
			break
		}

		peekIndex, _ = tmpReader.PeekNode(true)
		endIndex = peekIndex

		tmpReader.NextNode(true)
		if !tmpReader.PeekNodeIs(true, identifierListInfixMatcher) {
			break
		}
		peekIndex, _ = tmpReader.PeekNode(true)
		endIndex = peekIndex

		tmpReader.NextNode(true)
	}

	reader.Index = tmpReader.Index
	reader.CurNode = tmpReader.CurNode
	return &ast.IdentiferList{Toks: reader.NodesWithRange(startIndex, endIndex+1)}
}

// parseValues
