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

var indentNodeMatcher = astutil.NodeMatcher{
	NodeTypes: []ast.NodeType{
		ast.TypeIndent,
	},
}

var lineBreakMatcher = astutil.NodeMatcher{
	NodeTypes: []ast.NodeType{
		ast.TypeLineBreak,
	},
	ExpectKeyword: []string{
		"\n",
	},
}

var lineBreakNodeMatcher = astutil.NodeMatcher{
	NodeTypes: []ast.NodeType{
		ast.TypeLineBreak,
	},
}

var lineBreakTokenMatcher = astutil.NodeMatcher{
	ExpectKeyword: []string{
		"\n",
	},
}

var wsMatcher = astutil.NodeMatcher{
	ExpectTokens: []token.Kind{
		token.Whitespace,
	},
}

func formatPrefixGroup(reader *astutil.NodeReader, matcher astutil.NodeMatcher, fn prefixFormatFn) ast.Node {
	var replaceNodes []ast.Node
	for reader.NextNode(false) {
		if reader.CurNodeIs(matcher) {
			n := fn(reader)
			if n != nil {
				replaceNodes = append(replaceNodes, n)
			}
		} else if list, ok := reader.CurNode.(ast.TokenList); ok && !reader.CurNodeIs(indentNodeMatcher) && !reader.CurNodeIs(lineBreakNodeMatcher) {
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
		} else if list, ok := reader.CurNode.(ast.TokenList); ok && !reader.CurNodeIs(indentNodeMatcher) {
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
	var result ast.Node
	parent := &ast.Formatted{Toks: node.Flaten()}
	result = parent

	// trailing white space
	result = formatPrefixGroup(astutil.NewNodeReaderInc(result), lineBreakMatcher, trailWhitespaceAfterLinebreak)
	result = formatPrefixGroup(astutil.NewNodeReaderInc(result), indentNodeMatcher, trailWhitespaceAfterIndent)
	result = formatInfixGroup(astutil.NewNodeReaderInc(result), lineBreakMatcher, false, trailWhitespaceBeforeLineBreak)
	result = formatInfixGroup(astutil.NewNodeReaderInc(result), wsMatcher, false, trailDualWhitespace)

	// remove linebreak
	result = formatPrefixGroup(astutil.NewNodeReaderInc(result), lineBreakNodeMatcher, trailDualLineBreak)
	result = formatPrefixGroup(astutil.NewNodeReaderInc(result), indentNodeMatcher, trailLineBreakAfterIndent)
	result = formatPrefixGroup(astutil.NewNodeReaderInc(result), lineBreakTokenMatcher, trailLastLineBreak)
	result = formatPrefixGroup(astutil.NewNodeReaderInc(result), lineBreakTokenMatcher, trailLineBreak)

	return result
}

func trailWhitespaceAfterLinebreak(reader *astutil.NodeReader) ast.Node {
	n := reader.CurNode
	for reader.PeekNodeIs(false, wsMatcher) {
		reader.NextNode(false)
	}
	return n
}

func trailDualLineBreak(reader *astutil.NodeReader) ast.Node {
	n := reader.CurNode
	for reader.PeekNodeIs(false, lineBreakTokenMatcher) {
		reader.NextNode(false)
	}
	return n
}

func trailWhitespaceAfterIndent(reader *astutil.NodeReader) ast.Node {
	n := reader.CurNode
	for reader.PeekNodeIs(false, wsMatcher) {
		reader.NextNode(false)
	}
	return n
}

func trailLineBreakAfterIndent(reader *astutil.NodeReader) ast.Node {
	n := reader.CurNode
	for reader.PeekNodeIs(false, lineBreakTokenMatcher) {
		reader.NextNode(false)
	}
	return n
}

func trailLastWhiteSpace(reader *astutil.NodeReader) ast.Node {
	i, _ := reader.TailNode()
	if i == reader.Index {
		return nil
	}
	return reader.CurNode
}

func trailWhitespaceBeforeLineBreak(reader *astutil.NodeReader) ast.Node {
	for reader.CurNodeIs(wsMatcher) {
		reader.NextNode(false)
	}
	return reader.CurNode
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

func trailLineBreak(reader *astutil.NodeReader) ast.Node {
	if reader.CurNodeIs(lineBreakNodeMatcher) {
		return reader.CurNode
	}
	return whitespaceNode
}

func trailLastLineBreak(reader *astutil.NodeReader) ast.Node {
	i, _ := reader.TailNode()
	if i == reader.Index {
		return nil
	}
	return reader.CurNode
}
