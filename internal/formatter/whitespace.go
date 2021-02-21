package formatter

import (
	"github.com/lighttiger2505/sqls/ast"
	"github.com/lighttiger2505/sqls/ast/astutil"
	"github.com/lighttiger2505/sqls/token"
)

type (
	prefixFormatFn func(reader *astutil.NodeReader) ast.Node
	infixFormatFn  func(reader *astutil.NodeReader) ast.Node
)

var indentMatcher = astutil.NodeMatcher{
	NodeTypes: []ast.NodeType{
		ast.TypeIndent,
	},
}

var wsMatcher = astutil.NodeMatcher{
	ExpectTokens: []token.Kind{
		token.Whitespace,
	},
}

var lineBreakMatcher = astutil.NodeMatcher{
	ExpectKeyword: []string{
		"\n",
	},
}

func formatPrefixGroup(reader *astutil.NodeReader, matcher astutil.NodeMatcher, fn prefixFormatFn) ast.Node {
	var replaceNodes []ast.Node
	for reader.NextNode(false) {
		if reader.CurNodeIs(matcher) {
			replaceNodes = append(replaceNodes, fn(reader))
		} else if list, ok := reader.CurNode.(ast.TokenList); ok && !reader.CurNodeIs(indentMatcher) {
			newReader := astutil.NewNodeReader(list)
			replaceNodes = append(replaceNodes, formatPrefixGroup(newReader, matcher, fn))
		} else {
			replaceNodes = append(replaceNodes, reader.CurNode)
		}
	}
	reader.Node.SetTokens(replaceNodes)
	return reader.Node
}

func formatInfixGroup(reader *astutil.NodeReader, matcher astutil.NodeMatcher, ignoreWhiteSpace bool, fn infixFormatFn) ast.TokenList {
	var replaceNodes []ast.Node

	for reader.NextNode(false) {
		if reader.PeekNodeIs(ignoreWhiteSpace, matcher) {
			replaceNodes = append(replaceNodes, fn(reader))
		} else if list, ok := reader.CurNode.(ast.TokenList); ok && !reader.CurNodeIs(indentMatcher) {
			newReader := astutil.NewNodeReader(list)
			replaceNodes = append(replaceNodes, formatInfixGroup(newReader, matcher, ignoreWhiteSpace, fn))
		} else {
			replaceNodes = append(replaceNodes, reader.CurNode)
		}
	}
	reader.Node.SetTokens(replaceNodes)
	return reader.Node
}

func EvalTrailingWhitespace(node ast.Node, env *formatEnvironment) ast.Node {
	result := node
	result = formatPrefixGroup(astutil.NewNodeReaderInc(result), lineBreakMatcher, trailLinebreakAfterWhitespace)
	result = formatPrefixGroup(astutil.NewNodeReaderInc(result), indentMatcher, trailIndentAfterWhitespace)
	result = formatPrefixGroup(astutil.NewNodeReaderInc(result), indentMatcher, trailIndentAfterLinebreak)

	result = formatInfixGroup(astutil.NewNodeReaderInc(result), lineBreakMatcher, false, trailLinebreakBeforeWhitespace)
	result = formatInfixGroup(astutil.NewNodeReaderInc(result), wsMatcher, false, trailDualWhitespace)
	return result
}

func trailLinebreakAfterWhitespace(reader *astutil.NodeReader) ast.Node {
	lineBreakNode := reader.CurNode
	for reader.PeekNodeIs(false, wsMatcher) {
		reader.NextNode(false)
	}
	return lineBreakNode
}

func trailIndentAfterWhitespace(reader *astutil.NodeReader) ast.Node {
	indentNode := reader.CurNode
	for reader.PeekNodeIs(false, wsMatcher) {
		reader.NextNode(false)
	}
	return indentNode
}

func trailIndentAfterLinebreak(reader *astutil.NodeReader) ast.Node {
	indentNode := reader.CurNode
	for reader.PeekNodeIs(false, lineBreakMatcher) {
		reader.NextNode(false)
	}
	return indentNode
}

func trailLinebreakBeforeWhitespace(reader *astutil.NodeReader) ast.Node {
	curNode := reader.CurNode
	if !reader.CurNodeIs(wsMatcher) {
		formatted := formatInfixGroup(astutil.NewNodeReaderInc(reader.CurNode), lineBreakMatcher, false, trailLinebreakBeforeWhitespace)
		reader.Replace(formatted, reader.Index-1)
		return curNode
	}

	for reader.CurNodeIs(wsMatcher) {
		if reader.PeekNodeIs(false, lineBreakMatcher) {
			reader.NextNode(false)
			return reader.CurNode
		}
	}
	return curNode
}

var whitespaceInfixMatcher = astutil.NodeMatcher{
	ExpectTokens: []token.Kind{
		token.Whitespace,
	},
}

func trailDualWhitespace(reader *astutil.NodeReader) ast.Node {
	curNode := reader.CurNode
	for reader.PeekNodeIs(false, wsMatcher) {
		if reader.CurNodeIs(wsMatcher) {
			reader.NextNode(false)
		} else {
			break
		}
	}
	return curNode
}
