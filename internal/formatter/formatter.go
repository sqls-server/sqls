package formatter

import (
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
	formatted := Eval(parsed, &formatEnvironment{})
	dPrintln("formatted", formatted)

	res := []lsp.TextEdit{
		{
			Range: lsp.Range{
				Start: st,
				End:   en,
			},
			// NewText: format(parsed),
			NewText: formatted.String(),
		},
	}
	return res, nil
}

type formatEnvironment struct {
	reader      *astutil.NodeReader
	indentLevel int
}

func (e *formatEnvironment) indentLevelReset() {
	e.indentLevel = 0
}

func (e *formatEnvironment) indentLevelUp() {
	e.indentLevel++
}

func (e *formatEnvironment) indentLevelDown() {
	e.indentLevel--
}

func (e *formatEnvironment) genIndent() []ast.Node {
	nodes := []ast.Node{}
	for i := 0; i < e.indentLevel; i++ {
		nodes = append(nodes, indentNode)
	}
	return nodes
}

type prefixFormatFn func(nodes []ast.Node, reader *astutil.NodeReader, env formatEnvironment) ([]ast.Node, formatEnvironment)

type prefixFormatMap struct {
	matcher   *astutil.NodeMatcher
	formatter prefixFormatFn
}

func (pfm *prefixFormatMap) isMatch(reader *astutil.NodeReader) bool {
	if pfm.matcher != nil && reader.CurNodeIs(*pfm.matcher) {
		return true
	}
	return false
}

func Eval(node ast.Node, env *formatEnvironment) ast.Node {
	switch node := node.(type) {
	// case *ast.Query:
	// 	return formatQuery(node, env)
	// case *ast.Statement:
	// 	return formatStatement(node, env)
	case *ast.Item:
		return formatItem(node, env)
	case *ast.MultiKeyword:
		return formatMultiKeyword(node, env)
	case *ast.Aliased:
		return formatAliased(node, env)
	case *ast.Identifer:
		return formatIdentifer(node, env)
	case *ast.MemberIdentifer:
		return formatMemberIdentifer(node, env)
	case *ast.Operator:
		return formatOperator(node, env)
	case *ast.Comparison:
		return formatComparison(node, env)
	// case *ast.Parenthesis:
	// 	return formatParenthesis(node, env)
	// case *ast.ParenthesisInner:
	// case *ast.FunctionLiteral:
	// case *ast.IdentiferList:
	// case *ast.SwitchCase:
	// case *ast.Null:
	default:
		if list, ok := node.(ast.TokenList); ok {
			return formatTokenList(list, env)
		} else {
			return formatNode(node, env)
		}
	}
}

func formatItem(node ast.Node, env *formatEnvironment) ast.Node {
	results := []ast.Node{node}

	whitespaceMatcher := astutil.NodeMatcher{
		ExpectKeyword: []string{
			"JOIN",
			"ON",
		},
	}
	if whitespaceMatcher.IsMatch(node) {
		results = append(results, whitespaceNode)
	}

	linebreakBeforeMatcher := astutil.NodeMatcher{
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
			"EXCEPT",
		},
	}
	if linebreakBeforeMatcher.IsMatch(node) {
		env.indentLevelDown()
		results = unshift(results, linebreakNode)
	}

	indentAfterMatcher := astutil.NodeMatcher{
		ExpectKeyword: []string{
			"SELECT",
			"FROM",
			"WHERE",
		},
		ExpectTokens: []token.Kind{
			token.LParen,
		},
	}
	if indentAfterMatcher.IsMatch(node) {
		results = append(results, linebreakNode)
		env.indentLevelUp()
		results = append(results, env.genIndent()...)
	}

	indentBeforeMatcher := astutil.NodeMatcher{
		ExpectKeyword: []string{
			"ON",
			"AND",
			"OR",
		},
	}
	if indentBeforeMatcher.IsMatch(node) {
		env.indentLevelUp()
		results = unshift(results, env.genIndent()...)
		results = unshift(results, linebreakNode)
	}

	linebreakAfterMatcher := astutil.NodeMatcher{
		ExpectTokens: []token.Kind{
			token.Comma,
		},
	}
	if linebreakAfterMatcher.IsMatch(node) {
		results = append(results, linebreakNode)
		results = append(results, env.genIndent()...)
	}

	return &ast.ItemWith{Toks: results}
}

func formatMultiKeyword(node ast.Node, env *formatEnvironment) ast.Node {
	linebreakBeforeMatcher := astutil.NodeMatcher{
		ExpectKeyword: []string{
			"INNER JOIN",
			"CROSS JOIN",
			"LEFT JOIN",
			"RIGHT JOIN",
		},
	}
	results := []ast.Node{node}
	if linebreakBeforeMatcher.IsMatch(node) {
		results = unshift(results, linebreakNode)
	}
	return &ast.ItemWith{Toks: results}
}

func formatAliased(node *ast.Aliased, env *formatEnvironment) ast.Node {
	var results []ast.Node
	if node.IsAs {
		results = []ast.Node{
			node.RealName,
			asNode,
			node.AliasedName,
		}
	} else {
		results = []ast.Node{
			node.RealName,
			whitespaceNode,
			node.AliasedName,
		}
	}
	return &ast.ItemWith{Toks: results}
}

func formatIdentifer(node ast.Node, env *formatEnvironment) ast.Node {
	results := []ast.Node{node}

	// commaMatcher := astutil.NodeMatcher{
	// 	ExpectTokens: []token.Kind{
	// 		token.Comma,
	// 	},
	// }
	// if !env.reader.PeekNodeIs(true, commaMatcher) {
	// 	results = append(results, whitespaceNode)
	// }

	return &ast.ItemWith{Toks: results}
}

func formatMemberIdentifer(node *ast.MemberIdentifer, env *formatEnvironment) ast.Node {
	results := []ast.Node{
		node.Parent,
		periodNode,
		node.Child,
	}
	return &ast.ItemWith{Toks: results}
}

func formatOperator(node *ast.Operator, env *formatEnvironment) ast.Node {
	results := []ast.Node{
		Eval(node.Left, env),
		whitespaceNode,
		node.Operator,
		whitespaceNode,
		Eval(node.Right, env),
	}
	return &ast.ItemWith{Toks: results}
}

func formatComparison(node *ast.Comparison, env *formatEnvironment) ast.Node {
	results := []ast.Node{
		Eval(node.Left, env),
		whitespaceNode,
		node.Comparison,
		whitespaceNode,
		Eval(node.Right, env),
	}
	return &ast.ItemWith{Toks: results}
}

func formatTokenList(list ast.TokenList, env *formatEnvironment) ast.Node {
	results := []ast.Node{}
	reader := astutil.NewNodeReader(list)
	for reader.NextNode(true) {
		env.reader = reader
		results = append(results, Eval(reader.CurNode, env))
	}
	reader.Node.SetTokens(results)
	return reader.Node
}

func formatNode(node ast.Node, env *formatEnvironment) ast.Node {
	return node
}
