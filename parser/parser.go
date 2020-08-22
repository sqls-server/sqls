package parser

import (
	"bytes"
	"io"
	"strings"

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

func Parse(text string) (ast.TokenList, error) {
	src := bytes.NewBuffer([]byte(text))
	p, err := NewParser(src, &dialect.GenericSQLDialect{})
	if err != nil {
		return nil, err
	}
	parsed, err := p.Parse()
	if err != nil {
		return nil, err
	}
	return parsed, nil
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
		if tok.Kind == token.Comment {
			continue
		}
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
	root = parsePrefixGroup(astutil.NewNodeReader(root), identifierPrefixMatcher, parseIdentifier)
	root = parsePrefixGroup(astutil.NewNodeReader(root), switchCaseOpenMatcher, parseCase)

	root = parseInfixGroup(astutil.NewNodeReader(root), memberIdentifierInfixMatcher, false, parseMemberIdentifier)
	root = parsePrefixGroup(astutil.NewNodeReader(root), multiKeywordPrefixMatcher, parseMultiKeyword)
	root = parseInfixGroup(astutil.NewNodeReader(root), operatorInfixMatcher, true, parseOperator)
	root = parseInfixGroup(astutil.NewNodeReader(root), comparisonInfixMatcher, true, parseComparison)
	root = parsePrefixGroup(astutil.NewNodeReader(root), aliasLeftMatcher, parseAliasedWithoutAs)
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
	NodeTypes: []ast.NodeType{ast.TypeParenthesis},
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

var multiKeywordMap = map[string]string{
	"ORDER":  "BY",
	"GROUP":  "BY",
	"INSERT": "INTO",
	"DELETE": "FROM",
}
var multiKeywordPrefixMatcher = astutil.NodeMatcher{
	ExpectKeyword: []string{
		"ORDER",
		"GROUP",
		"INSERT",
		"DELETE",
	},
}

func parseMultiKeyword(reader *astutil.NodeReader) ast.Node {
	curKeyword := strings.ToUpper(reader.CurNode.String())
	peekKeyword := multiKeywordMap[curKeyword]
	if !reader.PeekNodeIs(true, astutil.NodeMatcher{ExpectKeyword: []string{peekKeyword}}) {
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
	ExpectTokens: []token.Kind{
		token.Mult,
	},
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
	NodeTypes: []ast.NodeType{
		ast.TypeIdentifer,
		ast.TypeMemberIdentifer,
		ast.TypeOperator,
		ast.TypeParenthesis,
		ast.TypeFunctionLiteral,
	},
	ExpectTokens: []token.Kind{
		token.Number,
		token.Char,
		token.SingleQuotedString,
		token.NationalStringLiteral,
	},
}
var operatorRecursionMatcher = astutil.NodeMatcher{
	NodeTypes: []ast.NodeType{
		ast.TypeParenthesis,
	},
}

func parseOperator(reader *astutil.NodeReader) ast.Node {
	if !reader.CurNodeIs(operatorTargetMatcher) {
		return reader.CurNode
	}
	if reader.CurNodeIs(operatorRecursionMatcher) {
		if list, ok := reader.CurNode.(ast.TokenList); ok {
			// FIXME: more simplity
			// For sub query
			parenthesis := parseInfixGroup(astutil.NewNodeReader(list), operatorInfixMatcher, true, parseOperator)
			reader.Replace(parenthesis, reader.Index-1)
		}
	}
	left := reader.CurNode
	startIndex := reader.Index - 1
	reader.NextNode(true)
	operator := &ast.Operator{
		Toks:     reader.NodesWithRange(startIndex, reader.Index),
		Left:     left,
		Operator: reader.CurNode,
	}

	if !reader.PeekNodeIs(true, operatorTargetMatcher) {
		// Include white space after the comma
		var endIndex int
		peekIndex, peekNode := reader.PeekNode(true)
		if peekNode != nil {
			endIndex = peekIndex - 1
			reader.Index = endIndex + 1
		} else {
			tailIndex, tailNode := reader.TailNode()
			endIndex = tailIndex - 1
			reader.Index = tailIndex
			reader.CurNode = tailNode
		}
		operator.Toks = reader.NodesWithRange(startIndex, endIndex+1)
		return operator
	}
	endIndex, right := reader.PeekNode(true)
	if reader.PeekNodeIs(true, operatorRecursionMatcher) {
		if list, ok := right.(ast.TokenList); ok {
			// FIXME: more simplity
			// For sub query
			parenthesis := parseInfixGroup(astutil.NewNodeReader(list), operatorInfixMatcher, true, parseOperator)
			reader.Replace(parenthesis, endIndex)
		}
	}
	operator.Right = right

	reader.NextNode(true)
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
	NodeTypes: []ast.NodeType{
		ast.TypeParenthesis,
		ast.TypeIdentifer,
		ast.TypeMemberIdentifer,
		ast.TypeOperator,
		ast.TypeFunctionLiteral,
	},
	ExpectTokens: []token.Kind{
		token.Number,
		token.Char,
		token.SingleQuotedString,
		token.NationalStringLiteral,
	},
}
var comparisonRecursionMatcher = astutil.NodeMatcher{
	NodeTypes: []ast.NodeType{
		ast.TypeParenthesis,
	},
}

func parseComparison(reader *astutil.NodeReader) ast.Node {
	if !reader.CurNodeIs(comparisonTargetMatcher) {
		return reader.CurNode
	}
	if reader.CurNodeIs(comparisonRecursionMatcher) {
		if list, ok := reader.CurNode.(ast.TokenList); ok {
			// FIXME: more simplity
			// For sub query
			parenthesis := parseInfixGroup(astutil.NewNodeReader(list), comparisonInfixMatcher, true, parseComparison)
			reader.Replace(parenthesis, reader.Index-1)
		}
	}
	left := reader.CurNode
	startIndex := reader.Index - 1
	reader.NextNode(true)
	comparison := &ast.Comparison{
		Toks:       reader.NodesWithRange(startIndex, reader.Index),
		Left:       left,
		Comparison: reader.CurNode,
	}

	if !reader.PeekNodeIs(true, comparisonTargetMatcher) {
		// Include white space after the comma
		var endIndex int
		peekIndex, peekNode := reader.PeekNode(true)
		if peekNode != nil {
			endIndex = peekIndex - 1
			reader.Index = endIndex + 1
		} else {
			tailIndex, tailNode := reader.TailNode()
			endIndex = tailIndex - 1
			reader.Index = tailIndex
			reader.CurNode = tailNode
		}
		comparison.Toks = reader.NodesWithRange(startIndex, endIndex+1)
		return comparison
	}
	endIndex, right := reader.PeekNode(true)
	if reader.PeekNodeIs(true, operatorRecursionMatcher) {
		if list, ok := right.(ast.TokenList); ok {
			// FIXME: more simplity
			// For sub query
			parenthesis := parseInfixGroup(astutil.NewNodeReader(list), comparisonInfixMatcher, true, parseComparison)
			reader.Replace(parenthesis, endIndex)
		}
	}
	comparison.Right = right

	reader.NextNode(true)
	comparison.Toks = reader.NodesWithRange(startIndex, endIndex+1)
	return comparison
}

var aliasInfixMatcher = astutil.NodeMatcher{
	ExpectKeyword: []string{
		"AS",
	},
}

var aliasLeftMatcher = astutil.NodeMatcher{
	NodeTypes: []ast.NodeType{
		ast.TypeParenthesis,
		ast.TypeFunctionLiteral,
		ast.TypeIdentifer,
		ast.TypeMemberIdentifer,
		ast.TypeSwitchCase,
	},
}

var aliasRightMatcher = astutil.NodeMatcher{
	NodeTypes: []ast.NodeType{
		ast.TypeIdentifer,
	},
}

var aliasRecursionMatcher = astutil.NodeMatcher{
	NodeTypes: []ast.NodeType{
		ast.TypeParenthesis,
	},
}

func parseAliasedWithoutAs(reader *astutil.NodeReader) ast.Node {
	if reader.CurNodeIs(aliasRecursionMatcher) {
		if list, ok := reader.CurNode.(ast.TokenList); ok {
			// FIXME: more simplity
			// For sub query
			parenthesis := parsePrefixGroup(astutil.NewNodeReader(list), aliasLeftMatcher, parseAliasedWithoutAs)
			reader.Replace(parenthesis, reader.Index-1)
		}
	}

	if !reader.PeekNodeIs(true, aliasRightMatcher) {
		return reader.CurNode
	}

	startIndex := reader.Index - 1
	realName := reader.CurNode
	endIndex, aliasedName := reader.PeekNode(true)
	reader.NextNode(true)

	return &ast.Aliased{
		Toks:        reader.NodesWithRange(startIndex, endIndex+1),
		RealName:    realName,
		AliasedName: aliasedName,
	}
}

func parseAliased(reader *astutil.NodeReader) ast.Node {
	if !reader.CurNodeIs(aliasLeftMatcher) {
		return reader.CurNode
	}
	if reader.CurNodeIs(aliasRecursionMatcher) {
		if list, ok := reader.CurNode.(ast.TokenList); ok {
			// FIXME: more simplity
			// For sub query
			parenthesis := parseInfixGroup(astutil.NewNodeReader(list), aliasInfixMatcher, true, parseAliased)
			reader.Replace(parenthesis, reader.Index-1)
		}
	}

	realName := reader.CurNode
	startIndex := reader.Index - 1
	tmpReader := reader.CopyReader()
	tmpReader.NextNode(true)

	if !tmpReader.PeekNodeIs(true, aliasRightMatcher) {
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
	NodeTypes: []ast.NodeType{
		ast.TypeFunctionLiteral,
		ast.TypeIdentifer,
		ast.TypeMemberIdentifer,
		ast.TypeAliased,
		ast.TypeComparison,
		ast.TypeOperator,
		ast.TypeSwitchCase,
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
			} else {
				tailIndex, tailNode := tmpReader.TailNode()
				endIndex = tailIndex - 1
				tmpReader.Index = tailIndex
				tmpReader.CurNode = tailNode
			}
			break
		}

		peekIndex, _ = tmpReader.PeekNode(true)
		endIndex = peekIndex

		tmpReader.NextNode(true)
		if !tmpReader.PeekNodeIs(true, identifierListInfixMatcher) {
			break
		}
		tmpReader.NextNode(true)
	}

	reader.Index = tmpReader.Index
	reader.CurNode = tmpReader.CurNode
	return &ast.IdentiferList{Toks: reader.NodesWithRange(startIndex, endIndex+1)}
}

var switchCaseOpenMatcher = astutil.NodeMatcher{
	ExpectKeyword: []string{
		"CASE",
	},
}
var switchCaseCloseMatcher = astutil.NodeMatcher{
	ExpectKeyword: []string{
		"END",
	},
}

func parseCase(reader *astutil.NodeReader) ast.Node {
	nodes := []ast.Node{reader.CurNode}

	tmpReader := reader.CopyReader()
	for tmpReader.NextNode(false) {
		if _, ok := reader.CurNode.(ast.TokenList); ok {
			continue
		}

		if tmpReader.CurNodeIs(switchCaseCloseMatcher) {
			reader.Index = tmpReader.Index
			reader.CurNode = tmpReader.CurNode
			return &ast.SwitchCase{Toks: append(nodes, tmpReader.CurNode)}
		} else {
			nodes = append(nodes, tmpReader.CurNode)
		}
	}
	return reader.Node
}
