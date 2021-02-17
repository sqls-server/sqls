package formatter

import (
	"github.com/lighttiger2505/sqls/ast"
	"github.com/lighttiger2505/sqls/ast/astutil"
	"github.com/lighttiger2505/sqls/token"
)

func EvalIndent(node ast.Node, env *formatEnvironment) ast.Node {
	switch node := node.(type) {
	// case *ast.Query:
	// 	return formatQuery(node, env)
	// case *ast.Statement:
	// 	return formatStatement(node, env)
	case *ast.Item:
		return indentItem(node, env)
	case *ast.MultiKeyword:
		return indentMultiKeyword(node, env)
	// case *ast.Aliased:
	// 	return pipeAppendEnv(indentAliased(node, env))
	// case *ast.Identifer:
	// 	return pipeAppendEnv(indentIdentifer(node, env))
	// case *ast.MemberIdentifer:
	// 	return pipeAppendEnv(indentMemberIdentifer(node, env))
	// case *ast.Operator:
	// 	return pipeAppendEnv(indentOperator(node, env))
	// case *ast.Comparison:
	// 	return pipeAppendEnv(indentComparison(node, env))
	case *ast.Parenthesis:
		return indentParenthesis(node, env)
	// case *ast.ParenthesisInner:
	// case *ast.FunctionLiteral:
	// 	return pipeAppendEnv(indentFunctionLiteral(node, env))
	case *ast.IdentiferList:
		return indentIdentiferList(node, env)
	// case *ast.SwitchCase:
	// case *ast.Null:
	default:
		if list, ok := node.(ast.TokenList); ok {
			return indentTokenList(list, env)
		} else {
			return indentNode(node, env)
		}
	}
}

func indentItem(node ast.Node, env *formatEnvironment) ast.Node {
	results := []ast.Node{node}

	// whitespaceAfterMatcher := astutil.NodeMatcher{
	// 	ExpectKeyword: []string{
	// 		"JOIN",
	// 		"ON",
	// 		"AND",
	// 		"OR",
	// 		"LIMIT",
	// 		"WHEN",
	// 		"ELSE",
	// 	},
	// }
	// if whitespaceAfterMatcher.IsMatch(node) {
	// 	results = append(results, whitespaceNode)
	// }
	// whitespaceAroundMatcher := astutil.NodeMatcher{
	// 	ExpectKeyword: []string{
	// 		"BETWEEN",
	// 		"USING",
	// 		"THEN",
	// 	},
	// }
	// if whitespaceAroundMatcher.IsMatch(node) {
	// 	results = unshift(results, whitespaceNode)
	// 	results = append(results, whitespaceNode)
	// }

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

func indentMultiKeyword(node *ast.MultiKeyword, env *formatEnvironment) ast.Node {
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

	// whitespaceAfterMatcher := astutil.NodeMatcher{
	// 	ExpectKeyword: joinKeywords,
	// }
	// if whitespaceAfterMatcher.IsMatch(node) {
	// 	results = append(results, whitespaceNode)
	// }

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

// func indentAliased(node *ast.Aliased, env *formatEnvironment) ast.Node {
// 	var results []ast.Node
// 	if node.IsAs {
// 		results = []ast.Node{
// 			Eval(node.RealName, env),
// 			whitespaceNode,
// 			node.As,
// 			whitespaceNode,
// 			Eval(node.AliasedName, env),
// 		}
// 	} else {
// 		results = []ast.Node{
// 			Eval(node.RealName, env),
// 			whitespaceNode,
// 			Eval(node.AliasedName, env),
// 		}
// 	}
// 	return &ast.ItemWith{Toks: results}
// }

// func indentIdentifer(node ast.Node, env *formatEnvironment) ast.Node {
// 	results := []ast.Node{node}
// 	// results := []ast.Node{node, whitespaceNode}
//
// 	// commaMatcher := astutil.NodeMatcher{
// 	// 	ExpectTokens: []token.Kind{
// 	// 		token.Comma,
// 	// 	},
// 	// }
// 	// if !env.reader.PeekNodeIs(true, commaMatcher) {
// 	// 	results = append(results, whitespaceNode)
// 	// }
//
// 	return &ast.ItemWith{Toks: results}
// }

// func indentMemberIdentifer(node *ast.MemberIdentifer, env *formatEnvironment) ast.Node {
// 	results := []ast.Node{
// 		Eval(node.Parent, env),
// 		periodNode,
// 		Eval(node.Child, env),
// 	}
// 	return &ast.ItemWith{Toks: results}
// }

// func indentOperator(node *ast.Operator, env *formatEnvironment) ast.Node {
// 	results := []ast.Node{
// 		Eval(node.Left, env),
// 		whitespaceNode,
// 		node.Operator,
// 		whitespaceNode,
// 		Eval(node.Right, env),
// 	}
// 	return &ast.ItemWith{Toks: results}
// }

// func indentComparison(node *ast.Comparison, env *formatEnvironment) ast.Node {
// 	results := []ast.Node{
// 		Eval(node.Left, env),
// 		whitespaceNode,
// 		node.Comparison,
// 		whitespaceNode,
// 		Eval(node.Right, env),
// 	}
// 	return &ast.ItemWith{Toks: results}
// }

func indentParenthesis(node *ast.Parenthesis, env *formatEnvironment) ast.Node {
	results := []ast.Node{}
	// results = append(results, whitespaceNode)
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
	// results = append(results, whitespaceNode)
	return &ast.Formatted{Toks: results}
}

// func indentFunctionLiteral(node *ast.FunctionLiteral, env *formatEnvironment) ast.Node {
// 	results := []ast.Node{node}
// 	return &ast.ItemWith{Toks: results}
// }

func indentIdentiferList(identiferList *ast.IdentiferList, env *formatEnvironment) ast.Node {
	idents := identiferList.GetIdentifers()
	results := []ast.Node{}
	for i, ident := range idents {
		results = append(results, Eval(ident, env))
		if i != len(idents)-1 {
			results = append(results, commaNode, linebreakNode)
			results = append(results, env.getIndent()...)
		}
	}
	return &ast.Formatted{Toks: results}
}

func indentTokenList(list ast.TokenList, env *formatEnvironment) ast.Node {
	results := []ast.Node{}
	reader := astutil.NewNodeReader(list)
	for reader.NextNode(false) {
		env.reader = reader
		results = append(results, EvalIndent(reader.CurNode, env))
	}
	reader.Node.SetTokens(results)
	return reader.Node
}

func indentNode(node ast.Node, env *formatEnvironment) ast.Node {
	return node
}
