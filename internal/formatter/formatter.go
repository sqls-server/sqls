package formatter

import (
	"errors"

	"github.com/sqls-server/sqls/ast"
	"github.com/sqls-server/sqls/ast/astutil"
	"github.com/sqls-server/sqls/internal/config"
	"github.com/sqls-server/sqls/internal/lsp"
	"github.com/sqls-server/sqls/parser"
	"github.com/sqls-server/sqls/token"
)

func Format(text string, params lsp.DocumentFormattingParams, cfg *config.Config) ([]lsp.TextEdit, error) {
	if text == "" {
		return nil, errors.New("empty")
	}
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
	env := &formatEnvironment{
		options: params.Options,
	}
	formatted := Eval(parsed, env)

	opts := &ast.RenderOptions{
		LowerCase: cfg.LowercaseKeywords,
	}
	res := []lsp.TextEdit{
		{
			Range: lsp.Range{
				Start: st,
				End:   en,
			},
			NewText: formatted.Render(opts),
		},
	}
	return res, nil
}

type formatEnvironment struct {
	reader      *astutil.NodeReader
	indentLevel int
	options     lsp.FormattingOptions
}

func (e *formatEnvironment) indentLevelReset() {
	e.indentLevel = 0
}

func (e *formatEnvironment) indentLevelUp() {
	e.indentLevel++
}

func (e *formatEnvironment) indentLevelDown() {
	e.indentLevel--
	if e.indentLevel < 0 {
		e.indentLevel = 0
	}
}

func (e *formatEnvironment) genIndent() []ast.Node {
	indent := whiteSpaceNodes(int(e.options.TabSize))
	if !e.options.InsertSpaces {
		indent = []ast.Node{tabNode}
	}
	nodes := []ast.Node{}
	for i := 0; i < e.indentLevel; i++ {
		nodes = append(nodes, indent...)
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
	case *ast.Identifier:
		return formatIdentifier(node, env)
	case *ast.MemberIdentifier:
		return formatMemberIdentifier(node, env)
	case *ast.Operator:
		return formatOperator(node, env)
	case *ast.Comparison:
		return formatComparison(node, env)
	case *ast.Parenthesis:
		return formatParenthesis(node, env)
	// case *ast.ParenthesisInner:
	case *ast.FunctionLiteral:
		return formatFunctionLiteral(node, env)
	case *ast.IdentifierList:
		return formatIdentifierList(node, env)
	// case *ast.SwitchCase:
	// case *ast.Null:
	default:
		if list, ok := node.(ast.TokenList); ok {
			return formatTokenList(list, env)
		}
		return formatNode(node, env)
	}
}

func formatItem(node ast.Node, env *formatEnvironment) ast.Node {
	results := []ast.Node{node}

	whitespaceAfterMatcher := astutil.NodeMatcher{
		ExpectKeyword: []string{
			"JOIN",
			"ON",
			"AND",
			"OR",
			"LIMIT",
			"WHEN",
			"ELSE",
		},
	}
	if whitespaceAfterMatcher.IsMatch(node) {
		results = append(results, whitespaceNode)
	}
	whitespaceAroundMatcher := astutil.NodeMatcher{
		ExpectKeyword: []string{
			"BETWEEN",
			"USING",
			"THEN",
		},
	}
	if whitespaceAroundMatcher.IsMatch(node) {
		results = unshift(results, whitespaceNode)
		results = append(results, whitespaceNode)
	}

	// Add an adjustment indent before the cursor
	outdentBeforeMatcher := astutil.NodeMatcher{
		ExpectKeyword: []string{
			"FROM",
			"JOIN",
			"WHERE",
			"HAVING",
			"LIMIT",
			"UNION",
			"VALUES",
			"SET",
			"EXCEPT",
			"END",
		},
	}
	if outdentBeforeMatcher.IsMatch(node) {
		env.indentLevelDown()
		results = unshift(results, env.genIndent()...)
		results = unshift(results, linebreakNode)
	}
	indentBeforeMatcher := astutil.NodeMatcher{
		ExpectKeyword: []string{
			"ON",
		},
	}
	if indentBeforeMatcher.IsMatch(node) {
		env.indentLevelUp()
		results = unshift(results, env.genIndent()...)
		results = unshift(results, linebreakNode)
	}
	linebreakBeforeMatcher := astutil.NodeMatcher{
		ExpectKeyword: []string{
			"AND",
			"OR",
			"WHEN",
			"ELSE",
		},
	}
	if linebreakBeforeMatcher.IsMatch(node) {
		results = unshift(results, env.genIndent()...)
		results = unshift(results, linebreakNode)
	}

	// Add an adjustment indent after the cursor
	linebreakWithIndentAfterMatcher := astutil.NodeMatcher{
		ExpectKeyword: []string{
			"SELECT",
			"FROM",
			"WHERE",
			"HAVING",
		},
		ExpectTokens: []token.Kind{
			token.LParen,
		},
	}
	if linebreakWithIndentAfterMatcher.IsMatch(node) {
		results = append(results, linebreakNode)
		env.indentLevelUp()
		results = append(results, env.genIndent()...)
	}
	indentAfterMatcher := astutil.NodeMatcher{
		ExpectKeyword: []string{
			"CASE",
		},
	}
	if indentAfterMatcher.IsMatch(node) {
		env.indentLevelUp()
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
	commentAfterMatcher := astutil.NodeMatcher{
		ExpectTokens: []token.Kind{
			token.Comment,
		},
	}
	if commentAfterMatcher.IsMatch(node) {
		results = append(results, linebreakNode)
		env.indentLevelDown()
	}

	breakStatementAfterMatcher := astutil.NodeMatcher{
		ExpectTokens: []token.Kind{
			token.Semicolon,
		},
	}
	if breakStatementAfterMatcher.IsMatch(node) {
		results = append(results, linebreakNode)
		env.indentLevelReset()
	}

	return &ast.ItemWith{Toks: results}
}

func formatMultiKeyword(node *ast.MultiKeyword, env *formatEnvironment) ast.Node {
	results := []ast.Node{}
	for i, kw := range node.GetKeywords() {
		results = append(results, kw)
		if i != len(node.GetKeywords())-1 {
			results = append(results, whitespaceNode)
		}
	}

	insertKeyword := "INSERT INTO"
	joinKeywords := []string{
		"INNER JOIN",
		"CROSS JOIN",
		"OUTER JOIN",
		"LEFT JOIN",
		"RIGHT JOIN",
		"LEFT OUTER JOIN",
		"RIGHT OUTER JOIN",
	}
	byKeywords := []string{
		"GROUP BY",
		"ORDER BY",
	}

	whitespaceAfterMatcher := astutil.NodeMatcher{
		ExpectKeyword: append(joinKeywords, insertKeyword),
	}
	if whitespaceAfterMatcher.IsMatch(node) {
		results = append(results, whitespaceNode)
	}

	// Add an adjustment indent before the cursor
	outdentBeforeMatcher := astutil.NodeMatcher{
		ExpectKeyword: append(joinKeywords, byKeywords...),
	}
	if outdentBeforeMatcher.IsMatch(node) {
		env.indentLevelDown()
		results = unshift(results, env.genIndent()...)
		results = unshift(results, linebreakNode)
	}

	// Add an adjustment indent after the cursor
	linebreakWithIndentAfterMatcher := astutil.NodeMatcher{
		ExpectKeyword: byKeywords,
	}
	if linebreakWithIndentAfterMatcher.IsMatch(node) {
		results = append(results, linebreakNode)
		env.indentLevelUp()
		results = append(results, env.genIndent()...)
	}

	return &ast.ItemWith{Toks: results}
}

func formatAliased(node *ast.Aliased, env *formatEnvironment) ast.Node {
	var results []ast.Node
	if node.IsAs {
		results = []ast.Node{
			Eval(node.RealName, env),
			whitespaceNode,
			node.As,
			whitespaceNode,
			Eval(node.AliasedName, env),
		}
	} else {
		results = []ast.Node{
			Eval(node.RealName, env),
			whitespaceNode,
			Eval(node.AliasedName, env),
		}
	}
	return &ast.ItemWith{Toks: results}
}

func formatIdentifier(node ast.Node, env *formatEnvironment) ast.Node {
	results := []ast.Node{node}
	// results := []ast.Node{node, whitespaceNode}

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

func formatMemberIdentifier(node *ast.MemberIdentifier, env *formatEnvironment) ast.Node {
	results := []ast.Node{
		Eval(node.Parent, env),
		periodNode,
		Eval(node.Child, env),
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

func formatParenthesis(node *ast.Parenthesis, env *formatEnvironment) ast.Node {
	results := []ast.Node{}
	// results = append(results, whitespaceNode)
	results = append(results, lparenNode)
	startIndentLevel := env.indentLevel
	env.indentLevelUp()
	results = append(results, linebreakNode)
	results = append(results, env.genIndent()...)
	results = append(results, Eval(node.Inner(), env))
	env.indentLevel = startIndentLevel
	results = append(results, linebreakNode)
	results = append(results, env.genIndent()...)
	results = append(results, rparenNode)
	// results = append(results, whitespaceNode)
	return &ast.ItemWith{Toks: results}
}

func formatFunctionLiteral(node *ast.FunctionLiteral, env *formatEnvironment) ast.Node {
	results := []ast.Node{node}
	return &ast.ItemWith{Toks: results}
}

func formatIdentifierList(identifierList *ast.IdentifierList, env *formatEnvironment) ast.Node {
	idents := identifierList.GetIdentifiers()
	results := []ast.Node{}
	for i, ident := range idents {
		results = append(results, Eval(ident, env))
		if i != len(idents)-1 {
			results = append(results, commaNode, linebreakNode)
			results = append(results, env.genIndent()...)
		}
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
