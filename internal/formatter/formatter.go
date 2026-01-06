package formatter

import (
	"errors"
	"strings"

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
	lastKeyword string // Track last processed keyword
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

	// Track certain keywords that affect formatting of subsequent keywords
	if item, ok := node.(*ast.Item); ok {
		if tok := item.GetToken(); tok.Kind == token.SQLKeyword {
			if sqlWord, ok := tok.Value.(*token.SQLWord); ok {
				kw := sqlWord.Keyword
				// Only track keywords that matter for context-sensitive formatting
				if kw == "JOIN" || kw == "DELETE" || kw == "INSERT" || kw == "CREATE" {
					env.lastKeyword = kw
				}
			}
		}
	}

	whitespaceAfterMatcher := astutil.NodeMatcher{
		ExpectKeyword: []string{
			"JOIN",
			"ON",
			"AND",
			"OR",
			"LIMIT",
			"WHEN",
			"ELSE",
			"AS",
			"DELETE",
			"UPDATE",
			"SET",
			"BEFORE",
			"AFTER",
			"TRIGGER",
			"INDEX",
			"ALTER",
			"ADD",
			"DROP",
			"MODIFY",
			"COLUMN",
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
		// Only indent ON if preceded by JOIN keyword
		// CREATE INDEX ON and CREATE TRIGGER ON should not be indented
		shouldIndent := false
		if env.lastKeyword == "JOIN" {
			shouldIndent = true
		}

		if shouldIndent {
			env.indentLevelUp()
			results = unshift(results, env.genIndent()...)
			results = unshift(results, linebreakNode)
		} else {
			// Still add linebreak for ON keyword, just without indent
			results = unshift(results, linebreakNode)
		}
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

	keywords := node.GetKeywords()
	for i, kw := range keywords {
		results = append(results, kw)
		if i != len(keywords)-1 {
			results = append(results, whitespaceNode)
		}
	}

	// Track last keyword in MultiKeyword
	if len(keywords) > 0 {
		lastKw := keywords[len(keywords)-1]
		if item, ok := lastKw.(*ast.Item); ok {
			if tok := item.GetToken(); tok.Kind == token.SQLKeyword {
				if sqlWord, ok := tok.Value.(*token.SQLWord); ok {
					env.lastKeyword = sqlWord.Keyword
				}
			}
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
	// Handle real name which may be a FunctionLiteral (e.g., INT(11))
	realNameTokens := Eval(node.RealName, env)
	results = append(results, realNameTokens)

	// Add whitespace before alias
	if node.IsAs || node.AliasedName != nil {
		results = append(results, whitespaceNode)
	}

	// Add AS keyword and aliased name
	if node.IsAs && node.As != nil {
		results = append(results,
			node.As,
			whitespaceNode,
			Eval(node.AliasedName, env),
		)
	} else if node.AliasedName != nil {
		results = append(results,
			Eval(node.AliasedName, env),
		)
	}
	return &ast.ItemWith{Toks: results}
}

func formatIdentifier(node ast.Node, env *formatEnvironment) ast.Node {
	// Handle special case: IF keyword parsed as Identifier
	if ident, ok := node.(*ast.Identifier); ok {
		keyword := ident.String()
		if keyword == "IF" {
			// Convert to lowercase keyword Item
			lowerItem := ast.NewItem(&token.Token{
				Kind: token.SQLKeyword,
				Value: &token.SQLWord{
					Value:   strings.ToLower(keyword),
					Keyword: strings.ToLower(keyword),
				},
			})
			return formatItem(lowerItem, env)
		}
	}

	results := []ast.Node{node}
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
	results = append(results, lparenNode)

	inner := node.Inner()

	// Check if inner contains IdentifierList
	innerToks := inner.GetTokens()
	if len(innerToks) == 1 {
		if idList, ok := innerToks[0].(*ast.IdentifierList); ok {
			// IdentifierList - format with linebreaks and indentation
			startIndentLevel := env.indentLevel
			env.indentLevelUp()
			results = append(results, linebreakNode)
			results = append(results, env.genIndent()...)
			results = append(results, Eval(idList, env))
			env.indentLevel = startIndentLevel
			results = append(results, linebreakNode)
			results = append(results, env.genIndent()...)
			results = append(results, rparenNode)
			return &ast.ItemWith{Toks: results}
		}
	}

	// Check if should format multi-line (has comma or SELECT keyword)
	shouldMultiLine := false

	for _, tok := range innerToks {
		if item, ok := tok.(*ast.Item); ok {
			tokKind := item.GetToken().Kind
			// Multi-line if has comma
			if tokKind == token.Comma {
				shouldMultiLine = true
				break
			}
			// Multi-line if has SELECT keyword (subquery)
			if tokKind == token.SQLKeyword {
				if sqlWord, ok := item.GetToken().Value.(*token.SQLWord); ok {
					if sqlWord.Keyword == "SELECT" {
						shouldMultiLine = true
						break
					}
				}
			}
		}
	}

	if shouldMultiLine {
		startIndentLevel := env.indentLevel
		env.indentLevelUp()
		results = append(results, linebreakNode)
		results = append(results, env.genIndent()...)
		results = append(results, Eval(inner, env))
		env.indentLevel = startIndentLevel
		results = append(results, linebreakNode)
		results = append(results, env.genIndent()...)
	} else {
		// Single line for simple cases like VARCHAR(100), function args
		results = append(results, Eval(inner, env))
	}

	results = append(results, rparenNode)
	return &ast.ItemWith{Toks: results}
}

func formatFunctionLiteral(node *ast.FunctionLiteral, env *formatEnvironment) ast.Node {
	// Check if this is a simple function call like SUM(x) or COUNT(y)
	// In this case, keep it on one line
	if len(node.Toks) == 2 {
		// First token should be the function name (Item)
		// Second token should be a Parenthesis
		if _, ok := node.Toks[0].(*ast.Item); ok {
			if paren, ok := node.Toks[1].(*ast.Parenthesis); ok {
				inner := paren.Inner()
				// Check if inner contains only a single identifier
				innerToks := inner.GetTokens()
				if len(innerToks) == 1 {
					if _, ok := innerToks[0].(*ast.Identifier); ok {
						// Simple function call like SUM(x)
						results := []ast.Node{
							node.Toks[0], // function name
							lparenNode,
							Eval(innerToks[0], env), // identifier
							rparenNode,
						}
						return &ast.ItemWith{Toks: results}
					}
				}
			}
		}
	}

	// Default: format recursively
	results := []ast.Node{}
	for _, tok := range node.Toks {
		results = append(results, Eval(tok, env))
	}
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
		formatted := Eval(reader.CurNode, env)
		results = append(results, formatted)

		// Add whitespace between adjacent tokens when needed
		// Check if we should add space after current token
		if shouldAddSpaceAfter(reader.CurNode, reader, formatted) {
			results = append(results, whitespaceNode)
		}
	}
	reader.Node.SetTokens(results)
	return reader.Node
}

func shouldAddSpaceAfter(curNode ast.Node, reader *astutil.NodeReader, formatted ast.Node) bool {
	// Don't add space if already ends with whitespace/linebreak
	if endsWithWhitespace(formatted) {
		return false
	}

	// Don't add space after comment
	if item, ok := curNode.(*ast.Item); ok {
		tok := item.GetToken()
		if tok.Kind == token.Comment || tok.Kind == token.MultilineComment {
			return false
		}
	}

	// Check next node (ignoreWhitespace=true to skip linebreaks)
	_, nextNode := reader.PeekNode(true)
	if nextNode == nil {
		return false
	}

	// Don't add space before MultiKeyword (e.g., "GROUP BY") - always
	if _, ok := nextNode.(*ast.MultiKeyword); ok {
		return false
	}

	// Debug: check if nextNode is comment
	if item, ok := nextNode.(*ast.Item); ok {
		tok := item.GetToken()
		if tok.Kind == token.Comment || tok.Kind == token.MultilineComment {
			return false
		}
	}

	// Don't add space before punctuation or comment
	if item, ok := nextNode.(*ast.Item); ok {
		tok := item.GetToken()
		if tok.Kind == token.Comma || tok.Kind == token.RParen ||
			tok.Kind == token.Semicolon || tok.Kind == token.Period ||
			tok.Kind == token.LParen || tok.Kind == token.Comment {
			return false
		}

		// Don't add space before BETWEEN, USING, THEN (they add their own space before)
		if tok.Kind == token.SQLKeyword {
			if sqlWord, ok := tok.Value.(*token.SQLWord); ok {
				kw := sqlWord.Keyword
				if kw == "BETWEEN" || kw == "USING" || kw == "THEN" {
					return false
				}
			}
		}

		// Don't add space before keywords that add linebreak before themselves
		if tok.Kind == token.SQLKeyword {
			if sqlWord, ok := tok.Value.(*token.SQLWord); ok {
				kw := sqlWord.Keyword
				// These keywords add linebreak before (via linebreakBeforeMatcher, outdentBeforeMatcher, etc)
				linebreakBeforeKeywords := []string{
					"FROM", "JOIN", "WHERE", "HAVING", "LIMIT", "UNION", "VALUES", "SET", "EXCEPT", "END",
					"ON", "AND", "OR", "WHEN", "ELSE",
				}
				for _, lbkw := range linebreakBeforeKeywords {
					if kw == lbkw {
						// Exception: if current keyword is DELETE and next is FROM, always add space
						// because "DELETE FROM" should be on one line
						if curItem, ok := curNode.(*ast.Item); ok {
							if curTok := curItem.GetToken(); curTok.Kind == token.SQLKeyword {
								if curSqlWord, ok := curTok.Value.(*token.SQLWord); ok {
									if curSqlWord.Keyword == "DELETE" && kw == "FROM" {
										return true // DELETE FROM should have space
									}
								}
							}
						}
						return false
					}
				}
			}
		}
	}

	// Add space after Item (keyword) unless specific exceptions
	if curItem, ok := curNode.(*ast.Item); ok {
		curTok := curItem.GetToken()

		// Add space before AS keyword for literals
		if nextItem, ok := nextNode.(*ast.Item); ok {
			if nextTok := nextItem.GetToken(); nextTok.Kind == token.SQLKeyword {
				if sqlWord, ok := nextTok.Value.(*token.SQLWord); ok {
					if sqlWord.Keyword == "AS" {
						// Add space before AS for any literal (number, string, boolean, etc.)
						if curTok.Kind != token.SQLKeyword {
							return true
						}
					}
				}
			}
		}

		// Only process SQL keywords
		if curTok.Kind != token.SQLKeyword {
			return false
		}

		// Check if current item is BETWEEN, USING, THEN (handled by whitespaceAroundMatcher)
		if sqlWord, ok := curTok.Value.(*token.SQLWord); ok {
			kw := sqlWord.Keyword
			if kw == "BETWEEN" || kw == "USING" || kw == "THEN" {
				return false // Already handled by formatItem
			}
		}

		// Don't add space before Parenthesis (e.g., VALUES(...))
		if _, ok := nextNode.(*ast.Parenthesis); ok {
			return false
		}

		// Don't add space before punctuation (already checked above)
		// Don't add space before BETWEEN, USING, THEN
		if item, ok := nextNode.(*ast.Item); ok {
			if tok := item.GetToken(); tok.Kind == token.SQLKeyword {
				if sqlWord, ok := tok.Value.(*token.SQLWord); ok {
					kw := sqlWord.Keyword
					if kw == "BETWEEN" || kw == "USING" || kw == "THEN" {
						return false
					}
				}
			}
		}

		// Default: add space after any SQL keyword
		return true
	}

	// Add space after Identifier if next is Item (keyword) or FunctionLiteral
	if _, ok := curNode.(*ast.Identifier); ok {
		if item, ok := nextNode.(*ast.Item); ok {
			// Don't add space before BETWEEN, USING, THEN (handled by whitespaceAroundMatcher)
			if tok := item.GetToken(); tok.Kind == token.SQLKeyword {
				if sqlWord, ok := tok.Value.(*token.SQLWord); ok {
					kw := sqlWord.Keyword
					if kw == "BETWEEN" || kw == "USING" || kw == "THEN" {
						return false
					}
				}
			}
			return true
		}
		// Don't add space before MultiKeyword (e.g., "GROUP BY") - it adds its own linebreak
		if _, ok := nextNode.(*ast.MultiKeyword); ok {
			return false
		}
		if _, ok := nextNode.(*ast.FunctionLiteral); ok {
			return true
		}
	}

	// Add space after FunctionLiteral if next is Item (keyword)
	if _, ok := curNode.(*ast.FunctionLiteral); ok {
		if _, ok := nextNode.(*ast.Item); ok {
			return true
		}
	}

	// Add space after ItemWith (e.g., MultiKeyword result like "DELETE FROM")
	if _, ok := formatted.(*ast.ItemWith); ok {
		// Don't add space before punctuation
		if item, ok := nextNode.(*ast.Item); ok {
			tok := item.GetToken()
			if tok.Kind == token.Comma || tok.Kind == token.RParen ||
				tok.Kind == token.Semicolon || tok.Kind == token.Period ||
				tok.Kind == token.LParen || tok.Kind == token.Comment {
				return false
			}
		}

		// Don't add space before Parenthesis (e.g., VALUES(...))
		if _, ok := nextNode.(*ast.Parenthesis); ok {
			return false
		}

		// Don't add space before MultiKeyword (e.g., "GROUP BY")
		if _, ok := nextNode.(*ast.MultiKeyword); ok {
			return false
		}

		// Add space before most node types
		switch nextNode.(type) {
		case *ast.Identifier, *ast.Item, *ast.Aliased:
			return true
		}
	}

	// Add space after MultiKeyword (e.g., "DELETE FROM") if it has multiple keywords
	if mk, ok := curNode.(*ast.MultiKeyword); ok {
		// Only add space if MultiKeyword has multiple keywords (e.g., "DELETE FROM")
		// Single keyword MultiKeywords (e.g., just "FROM") are handled by formatItem
		if len(mk.GetKeywords()) > 1 {
			// Don't add space if already ends with whitespace
			if endsWithWhitespace(formatted) {
				return false
			}
			// Add space before most node types
			switch nextNode.(type) {
			case *ast.Identifier, *ast.Item, *ast.Aliased:
				return true
			}
		}
	}

	return false
}

func endsWithWhitespace(node ast.Node) bool {
	if itemWith, ok := node.(*ast.ItemWith); ok {
		toks := itemWith.GetTokens()
		if len(toks) > 0 {
			return endsWithWhitespace(toks[len(toks)-1])
		}
	}
	if item, ok := node.(*ast.Item); ok {
		return item.GetToken().Kind == token.Whitespace
	}
	return false
}

func formatNode(node ast.Node, env *formatEnvironment) ast.Node {
	return node
}
