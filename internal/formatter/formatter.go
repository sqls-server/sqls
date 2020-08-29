package formatter

import (
	"fmt"
	"os"

	"github.com/lighttiger2505/sqls/ast"
	"github.com/lighttiger2505/sqls/ast/astutil"
	"github.com/lighttiger2505/sqls/internal/lsp"
	"github.com/lighttiger2505/sqls/parser"
	"github.com/lighttiger2505/sqls/token"
)

func Format(text string, params lsp.DocumentFormattingParams) ([]lsp.TextEdit, error) {
	parsed, err := parser.Parse(text)
	if err != nil {
		return nil, err
	}

	st := lsp.Position{
		Line:      parsed.Pos().Line,
		Character: parsed.Pos().Col,
	}
	en := lsp.Position{
		Line:      parsed.End().Line,
		Character: parsed.End().Col,
	}

	res := []lsp.TextEdit{
		{
			Range: lsp.Range{
				Start: st,
				End:   en,
			},
			NewText: format(parsed),
		},
	}
	return res, nil
}

func format(parsed ast.TokenList) string {
	ms := []prefixFormatMap{
		{
			matcher:   &whitespaceMatcher,
			formatter: formatWhiteSpace,
		},
		{
			matcher:   &indentAfterMatcher,
			formatter: formatIndentAfter,
		},
		{
			matcher:   &linebreakBeforeMatcher,
			formatter: formatLinebreakBefore,
		},
		{
			matcher:   &indentBeforeMatcher,
			formatter: formatIndentBefore,
		},
	}
	formatted := formattingProcess(astutil.NewNodeReader(parsed), ms)
	return formatted.String()
}

type formatEnvironment struct {
	indentLevel int
}

type prefixFormatFn func(nodes []ast.Node, reader *astutil.NodeReader, env formatEnvironment) ([]ast.Node, formatEnvironment)

type prefixFormatMap struct {
	matcher   *astutil.NodeMatcher
	ignore    *astutil.NodeMatcher
	formatter prefixFormatFn
}

func (pfm *prefixFormatMap) isIgnore(reader *astutil.NodeReader) bool {
	if pfm.ignore != nil && reader.CurNodeIs(*pfm.ignore) {
		dPrintln("ignore node", reader.CurNode)
		return true
	}
	return false
}

func (pfm *prefixFormatMap) isMatch(reader *astutil.NodeReader) bool {
	if pfm.matcher != nil && reader.CurNodeIs(*pfm.matcher) {
		return true
	}
	return false
}

func formattingProcess(reader *astutil.NodeReader, ms []prefixFormatMap) ast.TokenList {
	env := formatEnvironment{}
	var formattedNodes []ast.Node
	for reader.NextNode(true) {
		additionalNodes := []ast.Node{reader.CurNode}
		isFormatted := false
		isIgnore := false
		for _, s := range ms {
			if s.isIgnore(reader) {
				isIgnore = true
			}
			if s.isMatch(reader) {
				additionalNodes, env = s.formatter(additionalNodes, reader, env)
				isFormatted = true
			}
		}
		if isIgnore {
			formattedNodes = append(formattedNodes, reader.CurNode)
			continue
		}
		if isFormatted {
			formattedNodes = append(formattedNodes, additionalNodes...)
			continue
		}

		if list, ok := reader.CurNode.(ast.TokenList); ok {
			newReader := astutil.NewNodeReader(list)
			formattedNodes = append(formattedNodes, formattingProcess(newReader, ms))
		} else {
			formattedNodes = append(formattedNodes, reader.CurNode)
		}
	}
	reader.Node.SetTokens(formattedNodes)
	return reader.Node
}

func unshift(slice []ast.Node, node ...ast.Node) []ast.Node {
	return append(node, slice...)
}

var whitespaceNode = ast.NewItem(&token.Token{
	Kind:  token.Whitespace,
	Value: " ",
})

var linebreakNode = ast.NewItem(&token.Token{
	Kind:  token.Whitespace,
	Value: "\n",
})

var indentNode = ast.NewItem(&token.Token{
	Kind:  token.Whitespace,
	Value: "\t",
})

func genIndent(indentLevel int) []ast.Node {
	nodes := []ast.Node{}
	for i := 0; i < indentLevel; i++ {
		nodes = append(nodes, indentNode)
	}
	return nodes
}

var whitespaceMatcher = astutil.NodeMatcher{
	NodeTypes: []ast.NodeType{
		ast.TypeMemberIdentifer,
		ast.TypeIdentifer,
		ast.TypeOperator,
		ast.TypeComparison,
	},
	ExpectTokens: []token.Kind{
		token.SQLKeyword,
		token.Comma,
	},
}

func formatWhiteSpace(nodes []ast.Node, reader *astutil.NodeReader, env formatEnvironment) ([]ast.Node, formatEnvironment) {
	return append(nodes, whitespaceNode), env
}

var indentAfterMatcher = astutil.NodeMatcher{
	ExpectKeyword: []string{
		"SELECT",
	},
}

func formatIndentAfter(nodes []ast.Node, reader *astutil.NodeReader, env formatEnvironment) ([]ast.Node, formatEnvironment) {
	nodes = append(nodes, linebreakNode)
	env.indentLevel++
	nodes = append(nodes, genIndent(env.indentLevel)...)
	return nodes, env
}

var linebreakBeforeMatcher = astutil.NodeMatcher{
	ExpectKeyword: []string{
		"FROM",
		"JOIN",
		"INNER JOIN",
		"CROSS JOIN",
		"LEFT JOIN",
		"RIGHT JOIN",
		"WHERE",
		"HAVING",
		"LIMIT",
		"UNION",
		"VALUES",
		"SET",
		"BETWEEN",
		"EXCEPT",
	},
}

func formatLinebreakBefore(nodes []ast.Node, reader *astutil.NodeReader, env formatEnvironment) ([]ast.Node, formatEnvironment) {
	env.indentLevel = 0
	nodes = unshift(nodes, linebreakNode)
	return nodes, env
}

var indentBeforeMatcher = astutil.NodeMatcher{
	ExpectKeyword: []string{
		"ON",
		"AND",
		"OR",
	},
}

func formatIndentBefore(nodes []ast.Node, reader *astutil.NodeReader, env formatEnvironment) ([]ast.Node, formatEnvironment) {
	env.indentLevel++
	nodes = unshift(nodes, genIndent(env.indentLevel)...)
	nodes = unshift(nodes, linebreakNode)
	return nodes, env
}

func dPrintln(a ...interface{}) {
	fmt.Fprintln(os.Stderr, a...)
}

func dPrintf(format string, a ...interface{}) {
	fmt.Fprintf(os.Stderr, format, a...)
}
