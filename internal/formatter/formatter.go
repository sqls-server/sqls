package formatter

import (
	"errors"
	"log"

	"github.com/lighttiger2505/sqls/ast"
	"github.com/lighttiger2505/sqls/ast/astutil"
	"github.com/lighttiger2505/sqls/internal/config"
	"github.com/lighttiger2505/sqls/internal/lsp"
	"github.com/lighttiger2505/sqls/parser"
	"github.com/lighttiger2505/sqls/token"
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

	// formatted := Eval(parsed, env)
	var formatted ast.Node
	formatted = parsed
	formatted = Eval(parsed, newFormatEnvironment(params.Options))
	// formatted = EvalIndent(parsed, newFormatEnvironment(params.Options))
	// formatted = EvalTrailingWhitespace(formatted, newFormatEnvironment(params.Options))

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

	indentNode ast.Node
}

func newFormatEnvironment(options lsp.FormattingOptions) *formatEnvironment {
	return &formatEnvironment{options: options}
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
		log.Printf("invalid indent level, %d\n", e.indentLevel)
		e.indentLevel = 0
	}
}

func (e *formatEnvironment) getIndent() []ast.Node {
	if e.indentNode == nil {
		indent := whiteSpaceNodes(int(e.options.TabSize))
		if !e.options.InsertSpaces {
			indent = []ast.Node{tabNode}
		}
		e.indentNode = &ast.Indent{Toks: indent}
	}

	nodes := make([]ast.Node, e.indentLevel)
	for i := 0; i < e.indentLevel; i++ {
		nodes[i] = e.indentNode
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
	case *ast.Parenthesis:
		return formatParenthesis(node, env)
	// case *ast.ParenthesisInner:
	case *ast.FunctionLiteral:
		return formatFunctionLiteral(node, env)
	case *ast.IdentiferList:
		return formatIdentiferList(node, env)
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
		results = unshift(results, env.getIndent()...)
		results = unshift(results, linebreakNode)
	}
	indentBeforeMatcher := astutil.NodeMatcher{
		ExpectKeyword: []string{
			"ON",
		},
	}
	if indentBeforeMatcher.IsMatch(node) {
		env.indentLevelUp()
		results = unshift(results, env.getIndent()...)
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
		results = unshift(results, env.getIndent()...)
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
		results = append(results, env.getIndent()...)
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
		results = append(results, env.getIndent()...)
	}

	return &ast.Formatted{Toks: results}
}

func formatMultiKeyword(node *ast.MultiKeyword, env *formatEnvironment) ast.Node {
	results := []ast.Node{}
	for i, kw := range node.GetKeywords() {
		results = append(results, kw)
		if i != len(node.GetKeywords())-1 {
			results = append(results, whitespaceNode)
		}
	}

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
		ExpectKeyword: joinKeywords,
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
		results = unshift(results, env.getIndent()...)
		results = unshift(results, linebreakNode)
	}

	// Add an adjustment indent after the cursor
	linebreakWithIndentAfterMatcher := astutil.NodeMatcher{
		ExpectKeyword: byKeywords,
	}
	if linebreakWithIndentAfterMatcher.IsMatch(node) {
		results = append(results, linebreakNode)
		env.indentLevelUp()
		results = append(results, env.getIndent()...)
	}

	return &ast.Formatted{Toks: results}
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
	node.SetTokens(results)
	return node
}

func formatIdentifer(node ast.Node, env *formatEnvironment) ast.Node {
	return node
}

func formatMemberIdentifer(node *ast.MemberIdentifer, env *formatEnvironment) ast.Node {
	results := []ast.Node{
		Eval(node.Parent, env),
		periodNode,
		Eval(node.Child, env),
	}
	node.SetTokens(results)
	return node
}

func formatOperator(node *ast.Operator, env *formatEnvironment) ast.Node {
	results := []ast.Node{
		Eval(node.Left, env),
		whitespaceNode,
		node.Operator,
		whitespaceNode,
		Eval(node.Right, env),
	}
	node.SetTokens(results)
	return node
}

func formatComparison(node *ast.Comparison, env *formatEnvironment) ast.Node {
	results := []ast.Node{
		Eval(node.Left, env),
		whitespaceNode,
		node.Comparison,
		whitespaceNode,
		Eval(node.Right, env),
	}
	node.SetTokens(results)
	return node
}

func formatParenthesis(node *ast.Parenthesis, env *formatEnvironment) ast.Node {
	results := []ast.Node{}
	results = append(results, lparenNode)
	startIndentLevel := env.indentLevel
	env.indentLevelUp()
	results = append(results, linebreakNode)
	results = append(results, env.getIndent()...)
	results = append(results, Eval(node.Inner(), env))
	env.indentLevel = startIndentLevel
	results = append(results, linebreakNode)
	results = append(results, env.getIndent()...)
	results = append(results, rparenNode)

	node.SetTokens(results)
	return node
}

func formatFunctionLiteral(node *ast.FunctionLiteral, env *formatEnvironment) ast.Node {
	return node
}

func formatIdentiferList(identiferList *ast.IdentiferList, env *formatEnvironment) ast.Node {
	idents := identiferList.GetIdentifers()
	results := []ast.Node{}
	for i, ident := range idents {
		results = append(results, Eval(ident, env))
		if i != len(idents)-1 {
			results = append(results, commaNode, linebreakNode)
			results = append(results, env.getIndent()...)
		}
	}
	identiferList.SetTokens(results)
	return identiferList
}

func formatTokenList(list ast.TokenList, env *formatEnvironment) ast.Node {
	results := []ast.Node{}
	reader := astutil.NewNodeReader(list)
	for reader.NextNode(true) {
		env.reader = reader
		res := Eval(reader.CurNode, env)
		formatted, ok := res.(*ast.Formatted)
		if ok {
			results = append(results, formatted.GetTokens()...)
		} else {
			results = append(results, res)
		}
	}
	reader.Node.SetTokens(results)
	return reader.Node
}

func formatNode(node ast.Node, env *formatEnvironment) ast.Node {
	return node
}
