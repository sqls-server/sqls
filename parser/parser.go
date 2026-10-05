package parser

import (
	"bytes"
	"fmt"
	"io"
	"strings"

	"github.com/sqls-server/sqls/ast"
	"github.com/sqls-server/sqls/ast/astutil"
	"github.com/sqls-server/sqls/dialect"
	"github.com/sqls-server/sqls/token"
)

type (
	prefixParseFn func(reader *astutil.NodeReader) ast.Node
	infixParseFn  func(reader *astutil.NodeReader) ast.Node
)

func parsePrefixGroup(reader *astutil.NodeReader, matcher astutil.NodeMatcher, fn prefixParseFn) ast.TokenList {
	var replaceNodes []ast.Node
	for reader.NextNode(false) {
		if list, ok := reader.CurNode.(ast.TokenList); ok {
			newReader := astutil.NewNodeReader(list)
			replaceNode := parsePrefixGroup(newReader, matcher, fn)
			reader.Replace(replaceNode, reader.Index-1)
		}
		if reader.CurNodeIs(matcher) {
			replaceNodes = append(replaceNodes, fn(reader))
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
		if list, ok := reader.CurNode.(ast.TokenList); ok {
			newReader := astutil.NewNodeReader(list)
			replaceNode := parseInfixGroup(newReader, matcher, ignoreWhiteSpace, fn)
			reader.Replace(replaceNode, reader.Index-1)
		}
		if reader.PeekNodeIs(ignoreWhiteSpace, matcher) {
			replaceNodes = append(replaceNodes, fn(reader))
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
		return nil, fmt.Errorf("tokenize err failed: %w", err)
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
	root = parsePrefixGroup(astutil.NewNodeReader(root), identifierPrefixMatcher, parseIdentifier)
	root = parseInfixGroup(astutil.NewNodeReader(root), memberIdentifierInfixMatcher, false, parseMemberIdentifier)
	root = parsePrefixGroup(astutil.NewNodeReader(root), switchCaseOpenMatcher, parseCase)

	root = parsePrefixGroup(astutil.NewNodeReader(root), expressionPrefixMatcher, parseExpressionInParenthesis)

	root = parsePrefixGroup(astutil.NewNodeReader(root), genMultiKeywordPrefixMatcher(), parseMultiKeyword)
	root = parseInfixGroup(astutil.NewNodeReader(root), operatorInfixMatcher, true, parseOperator)
	root = parseInfixGroup(astutil.NewNodeReader(root), comparisonInfixMatcher, true, parseComparison)
	root = parsePrefixGroup(astutil.NewNodeReader(root), aliasLeftMatcher, parseAliasedWithoutAs)
	root = parseInfixGroup(astutil.NewNodeReader(root), aliasInfixMatcher, true, parseAliased)
	root = parseInfixGroup(astutil.NewNodeReader(root), identifierListInfixMatcher, true, parseIdentifierList)
	return root, nil
}

func isTopLevelStatementKeyword(kw string) bool {
	switch kw {
	case "SELECT", "INSERT", "UPDATE", "DELETE", "CREATE", "ALTER", "DROP",
		"TRUNCATE", "WITH", "USE", "SHOW", "DESCRIBE", "EXPLAIN",
		"GRANT", "REVOKE", "MERGE", "CALL", "OPTIMIZE", "PRAGMA", "ATTACH", "DETACH":
		return true
	}
	return false
}

func isDMLKeyword(kw string) bool {
	switch kw {
	case "SELECT", "INSERT", "UPDATE", "DELETE":
		return true
	}
	return false
}

func isSetOperator(kw string) bool {
	switch kw {
	case "UNION", "INTERSECT", "EXCEPT", "MINUS":
		return true
	}
	return false
}

func isSetModifier(kw string) bool {
	switch kw {
	case "ALL", "DISTINCT":
		return true
	}
	return false
}

func isSpecialTrackingKeyword(kw string) bool {
	return isSetOperator(kw) || isSetModifier(kw) || kw == "AS" || kw == "BEGIN" || kw == "END" || kw == "BEFORE" || kw == "AFTER" || kw == "INSTEAD" || kw == "KEY" || kw == "DO"
}

func isAlterTableDropClause(nextKW string) bool {
	switch nextKW {
	case "COLUMN", "CONSTRAINT", "INDEX", "KEY", "FOREIGN", "PRIMARY", "CHECK", "PARTITION":
		return true
	}
	return false
}

func parseStatement(reader *astutil.NodeReader) ast.TokenList {
	tokens := reader.Node.GetTokens()
	if len(tokens) == 0 {
		return reader.Node
	}

	var replaceNodes []ast.Node
	var startIndex int

	parenDepth := 0
	blockDepth := 0
	stmtFirstKeyword := ""
	lastNonWsKeyword := ""
	prevNonWsKeyword := ""
	lastCodeTokenIndex := -1
	hasNewlineSinceLastCode := false
	hasSeenSelectInInsert := false
	hasSeenValuesInInsert := false
	hasSeenMainQueryInWith := false

	isStatementBoundary := func(i int, upperVal string) bool {
		if parenDepth > 0 || blockDepth > 0 {
			return false
		}
		if !hasNewlineSinceLastCode {
			return false
		}
		if !isTopLevelStatementKeyword(upperVal) {
			return false
		}
		if stmtFirstKeyword == "" {
			return false
		}

		// Safeguard 1: UNION / INTERSECT / EXCEPT / MINUS
		if upperVal == "SELECT" {
			if isSetOperator(lastNonWsKeyword) ||
				(isSetModifier(lastNonWsKeyword) && isSetOperator(prevNonWsKeyword)) {
				return false
			}
		}

		// Safeguard 2: CREATE ... AS SELECT
		if upperVal == "SELECT" && lastNonWsKeyword == "AS" {
			return false
		}

		// Safeguard 3: INSERT / REPLACE ... SELECT
		if upperVal == "SELECT" && (stmtFirstKeyword == "INSERT" || stmtFirstKeyword == "REPLACE") && !hasSeenSelectInInsert && !hasSeenValuesInInsert {
			hasSeenSelectInInsert = true
			return false
		}

		// Safeguard 4: WITH ... [SELECT|INSERT|UPDATE|DELETE]
		if stmtFirstKeyword == "WITH" && isDMLKeyword(upperVal) && !hasSeenMainQueryInWith {
			hasSeenMainQueryInWith = true
			return false
		}

		// Safeguard 5: ALTER TABLE ... DROP COLUMN / CONSTRAINT ...
		if stmtFirstKeyword == "ALTER" && upperVal == "DROP" {
			// Check next non-whitespace keyword
			for j := i + 1; j < len(tokens); j++ {
				nextTok, ok := tokens[j].(ast.Token)
				if !ok {
					continue
				}
				sTok := nextTok.GetToken()
				if sTok.MatchKind(token.Whitespace) || sTok.MatchKind(token.Comment) || sTok.MatchKind(token.MultilineComment) {
					continue
				}
				if isAlterTableDropClause(strings.ToUpper(sTok.String())) {
					return false
				}
				break
			}
		}

		// Safeguard 6: Trigger events (BEFORE/AFTER/INSTEAD OF UPDATE|INSERT|DELETE)
		if lastNonWsKeyword == "BEFORE" || lastNonWsKeyword == "AFTER" || lastNonWsKeyword == "INSTEAD" {
			return false
		}

		// Safeguard 7: ON DUPLICATE KEY UPDATE / ON CONFLICT DO UPDATE
		if upperVal == "UPDATE" && (lastNonWsKeyword == "KEY" || lastNonWsKeyword == "DO") {
			return false
		}

		return true
	}

	for i := 0; i < len(tokens); i++ {
		node := tokens[i]
		if list, ok := node.(ast.TokenList); ok {
			replaceNodes = append(replaceNodes, parseStatement(astutil.NewNodeReader(list)))
			continue
		}

		tok, ok := node.(ast.Token)
		if !ok {
			continue
		}
		sqlTok := tok.GetToken()

		// Track parentheses
		if sqlTok.MatchKind(token.LParen) {
			parenDepth++
			lastCodeTokenIndex = i
			hasNewlineSinceLastCode = false
			continue
		} else if sqlTok.MatchKind(token.RParen) {
			if parenDepth > 0 {
				parenDepth--
			}
			lastCodeTokenIndex = i
			hasNewlineSinceLastCode = false
			continue
		}

		// Explicit semicolon
		if sqlTok.MatchKind(token.Semicolon) {
			if parenDepth == 0 && blockDepth == 0 {
				stmt := &ast.Statement{Toks: tokens[startIndex : i+1]}
				replaceNodes = append(replaceNodes, stmt)
				startIndex = i + 1
				stmtFirstKeyword = ""
				lastNonWsKeyword = ""
				prevNonWsKeyword = ""
				lastCodeTokenIndex = i
				hasNewlineSinceLastCode = false
				hasSeenSelectInInsert = false
				hasSeenValuesInInsert = false
				hasSeenMainQueryInWith = false
				continue
			}
		}

		// Track newlines in whitespace
		if sqlTok.MatchKind(token.Whitespace) {
			if strings.Contains(sqlTok.String(), "\n") {
				hasNewlineSinceLastCode = true
			}
			continue
		}

		// Comments
		if sqlTok.MatchKind(token.Comment) || sqlTok.MatchKind(token.MultilineComment) {
			if strings.Contains(sqlTok.String(), "\n") {
				hasNewlineSinceLastCode = true
			}
			continue
		}

		upperVal := strings.ToUpper(sqlTok.String())
		if upperVal == "VALUES" && parenDepth == 0 {
			hasSeenValuesInInsert = true
		}

		// Check for REPLACE INTO
		if upperVal == "REPLACE" && i+1 < len(tokens) {
			for j := i + 1; j < len(tokens); j++ {
				nextTok, ok := tokens[j].(ast.Token)
				if !ok {
					continue
				}
				nextSQLTok := nextTok.GetToken()
				if nextSQLTok.MatchKind(token.Whitespace) || nextSQLTok.MatchKind(token.Comment) || nextSQLTok.MatchKind(token.MultilineComment) {
					continue
				}
				if strings.EqualFold(nextSQLTok.String(), "INTO") {
					upperVal = "INSERT"
				}
				break
			}
		}

		// Track BEGIN ... END blocks (PL/pgSQL, T-SQL)
		if parenDepth == 0 {
			if upperVal == "BEGIN" {
				blockDepth++
			} else if upperVal == "END" && blockDepth > 0 {
				blockDepth--
			}
		}

		// Check if this keyword marks a new statement boundary
		if isStatementBoundary(i, upperVal) {
			splitIndex := i
			for k := lastCodeTokenIndex + 1; k < i; k++ {
				tok, ok := tokens[k].(ast.Token)
				if !ok {
					continue
				}
				sTok := tok.GetToken()
				if sTok.MatchKind(token.Whitespace) && strings.Contains(sTok.String(), "\n") {
					splitIndex = k + 1
					break
				}
			}

			if splitIndex > startIndex {
				stmt := &ast.Statement{Toks: tokens[startIndex:splitIndex]}
				replaceNodes = append(replaceNodes, stmt)
			}
			startIndex = splitIndex
			stmtFirstKeyword = upperVal
			lastNonWsKeyword = upperVal
			prevNonWsKeyword = ""
			lastCodeTokenIndex = i
			hasNewlineSinceLastCode = false
			hasSeenSelectInInsert = false
			hasSeenValuesInInsert = false
			hasSeenMainQueryInWith = false
			continue
		}

		lastCodeTokenIndex = i
		hasNewlineSinceLastCode = false

		// Update keyword tracking for current statement
		if isTopLevelStatementKeyword(upperVal) || isSpecialTrackingKeyword(upperVal) {
			if stmtFirstKeyword == "" && parenDepth == 0 && isTopLevelStatementKeyword(upperVal) {
				stmtFirstKeyword = upperVal
			}
			prevNonWsKeyword = lastNonWsKeyword
			lastNonWsKeyword = upperVal
		} else {
			prevNonWsKeyword = lastNonWsKeyword
			lastNonWsKeyword = ""
		}
	}

	if startIndex < len(tokens) {
		stmt := &ast.Statement{Toks: tokens[startIndex:]}
		replaceNodes = append(replaceNodes, stmt)
	}

	reader.Node.SetTokens(replaceNodes)
	return reader.Node
}

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
	startIndex := reader.Index - 1
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

	// Include white space after the comma
	var endIndex int
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
	reader.Index = tmpReader.Index
	reader.CurNode = tmpReader.CurNode
	return &ast.Parenthesis{Toks: reader.NodesWithRange(startIndex, endIndex+1)}
}

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
	NodeTypes: []ast.NodeType{
		ast.TypeIdentifier,
	},
}

func parseMemberIdentifier(reader *astutil.NodeReader) ast.Node {
	var memberIdentifier *ast.MemberIdentifier
	if !reader.CurNodeIs(memberIdentifierTargetMatcher) {
		return reader.CurNode
	}
	parent := reader.CurNode
	startIndex := reader.Index - 1
	memberIdentifier = ast.NewMemberIdentifierParent(
		reader.NodesWithRange(startIndex, reader.Index+1),
		parent,
	)

	reader.NextNode(false)
	if !reader.PeekNodeIs(true, memberIdentifierTargetMatcher) {
		return memberIdentifier
	}
	endIndex, child := reader.PeekNode(true)
	memberIdentifier = ast.NewMemberIdentifier(
		reader.NodesWithRange(startIndex, endIndex+1),
		parent,
		child,
	)

	reader.NextNode(false)
	return memberIdentifier
}

var multiKeywordMap = map[string][]string{
	"ORDER":   {"BY"},
	"GROUP":   {"BY"},
	"INSERT":  {"INTO"},
	"DELETE":  {"FROM"},
	"INNER":   {"JOIN"},
	"CROSS":   {"JOIN"},
	"OUTER":   {"JOIN"},
	"LEFT":    {"OUTER", "JOIN"},
	"RIGHT":   {"OUTER", "JOIN"},
	"NATURAL": {"LEFT", "RIGHT", "OUTER", "JOIN"},
}

func genMultiKeywordPrefixMatcher() astutil.NodeMatcher {
	keywords := []string{}
	for k := range multiKeywordMap {
		keywords = append(keywords, k)
	}
	return astutil.NodeMatcher{ExpectKeyword: keywords}
}

func parseMultiKeyword(reader *astutil.NodeReader) ast.Node {
	keywords := []ast.Node{}
	startIndex := reader.Index - 1
	for {
		keywords = append(keywords, reader.CurNode)
		curKeyword := strings.ToUpper(reader.CurNode.String())
		peekKeywords, ok := multiKeywordMap[curKeyword]
		if !ok {
			break
		}
		if !reader.PeekNodeIs(true, astutil.NodeMatcher{ExpectKeyword: peekKeywords}) {
			if len(keywords) > 1 {
				// Partial match. Keep the keywords consumed so far
				// so they are not dropped from the statement.
				break
			}
			return reader.CurNode
		}
		reader.NextNode(true)
	}
	return &ast.MultiKeyword{
		Toks:     reader.NodesWithRange(startIndex, reader.Index),
		Keywords: keywords,
	}
}

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
	return &ast.Identifier{Tok: token.GetToken()}
}

var operatorInfixMatcher = astutil.NodeMatcher{
	ExpectTokens: []token.Kind{
		token.Plus,
		token.Minus,
		token.Mult,
		token.Div,
		token.Mod,
		token.Caret,
	},
}
var operatorTargetMatcher = astutil.NodeMatcher{
	NodeTypes: []ast.NodeType{
		ast.TypeIdentifier,
		ast.TypeMemberIdentifier,
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

func parseOperator(reader *astutil.NodeReader) ast.Node {
	if !reader.CurNodeIs(operatorTargetMatcher) {
		return reader.CurNode
	}

	left := reader.CurNode
	startIndex := reader.Index - 1

	for {
		if !reader.PeekNodeIs(true, operatorInfixMatcher) {
			return left
		}

		reader.NextNode(true)
		operator := reader.CurNode

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
			return &ast.Operator{
				Toks:     reader.NodesWithRange(startIndex, endIndex+1),
				Left:     left,
				Operator: operator,
			}
		}
		endIndex, right := reader.PeekNode(true)

		reader.NextNode(true)
		left = &ast.Operator{
			Toks:     reader.NodesWithRange(startIndex, endIndex+1),
			Left:     left,
			Operator: operator,
			Right:    right,
		}
	}
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
	ExpectKeyword: []string{
		"IS",
	},
}
var comparisonTargetMatcher = astutil.NodeMatcher{
	NodeTypes: []ast.NodeType{
		ast.TypeParenthesis,
		ast.TypeIdentifier,
		ast.TypeMemberIdentifier,
		ast.TypeOperator,
		ast.TypeFunctionLiteral,
	},
	ExpectTokens: []token.Kind{
		token.Number,
		token.Char,
		token.SingleQuotedString,
		token.NationalStringLiteral,
	},
	ExpectKeyword: []string{
		"TRUE",
		"FALSE",
	},
}

func parseComparison(reader *astutil.NodeReader) ast.Node {
	if !reader.CurNodeIs(comparisonTargetMatcher) {
		return reader.CurNode
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
		ast.TypeIdentifier,
		ast.TypeMemberIdentifier,
		ast.TypeSwitchCase,
		ast.TypeOperator,
	},
}

var aliasRightMatcher = astutil.NodeMatcher{
	NodeTypes: []ast.NodeType{
		ast.TypeIdentifier,
	},
}

var aliasRecursionMatcher = astutil.NodeMatcher{
	NodeTypes: []ast.NodeType{
		ast.TypeParenthesis,
	},
}

func parseAliasedWithoutAs(reader *astutil.NodeReader) ast.Node {
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
		IsAs:        false,
	}
}

func parseAliased(reader *astutil.NodeReader) ast.Node {
	// Check if current node is a literal Item (number, string, boolean, etc.)
	if item, ok := reader.CurNode.(*ast.Item); ok {
		tok := item.GetToken()
		// Allow literals to have aliases
		isLiteral := tok.Kind == token.Number ||
			tok.Kind == token.Char ||
			tok.Kind == token.SingleQuotedString ||
			tok.Kind == token.NationalStringLiteral

		// Also allow TRUE/FALSE/NULL keywords as literals
		if !isLiteral && tok.Kind == token.SQLKeyword {
			if sqlWord, ok := tok.Value.(*token.SQLWord); ok {
				keyword := sqlWord.Keyword
				isLiteral = keyword == "TRUE" || keyword == "FALSE" || keyword == "NULL"
			}
		}

		if !isLiteral {
			// Non-literal Item, check normal matcher
			if !reader.CurNodeIs(aliasLeftMatcher) {
				return reader.CurNode
			}
		}
	} else if !reader.CurNodeIs(aliasLeftMatcher) {
		return reader.CurNode
	}

	realName := reader.CurNode
	_, as := reader.PeekNode(true)
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
		As:          as,
		IsAs:        true,
	}
}

var commentInfixMatcher = astutil.NodeMatcher{
	ExpectTokens: []token.Kind{
		token.Comment,
		token.MultilineComment,
	},
}
var identifierListInfixMatcher = astutil.NodeMatcher{
	ExpectTokens: []token.Kind{
		token.Comma,
	},
}
var identifierListTargetMatcher = astutil.NodeMatcher{
	ExpectTokens: []token.Kind{
		token.Number,
		token.Char,
		token.SingleQuotedString,
		token.NationalStringLiteral,
	},
	NodeTypes: []ast.NodeType{
		ast.TypeFunctionLiteral,
		ast.TypeIdentifier,
		ast.TypeMemberIdentifier,
		ast.TypeAliased,
		ast.TypeComparison,
		ast.TypeOperator,
		ast.TypeSwitchCase,
	},
	ExpectKeyword: []string{
		"NULL",
		"TRUE",
		"FALSE",
	},
}

func parseIdentifierList(reader *astutil.NodeReader) ast.Node {
	// Don't start identifier list with a comment
	if item, ok := reader.CurNode.(*ast.Item); ok {
		tok := item.GetToken()
		if tok.Kind == token.Comment || tok.Kind == token.MultilineComment {
			return reader.CurNode
		}
		// Don't start with SQL keywords (except literals like TRUE/FALSE/NULL)
		if tok.Kind == token.SQLKeyword {
			if sqlWord, ok := tok.Value.(*token.SQLWord); ok {
				keyword := sqlWord.Keyword
				// Only allow literal keywords
				if keyword != "TRUE" && keyword != "FALSE" && keyword != "NULL" {
					return reader.CurNode
				}
			}
		}
	}

	if !reader.CurNodeIs(identifierListTargetMatcher) {
		return reader.CurNode
	}
	idents := []ast.Node{reader.CurNode}
	startIndex := reader.Index - 1
	tmpReader := reader.CopyReader()
	tmpReader.NextNode(true)
	commas := []ast.Node{tmpReader.CurNode}

	var (
		endIndex, peekIndex int
		peekNode            ast.Node
	)
	for {
		if !tmpReader.PeekNodeIs(true, identifierListTargetMatcher) && !tmpReader.PeekNodeIs(true, commentInfixMatcher) {
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
		for tmpReader.PeekNodeIs(true, commentInfixMatcher) {
			tmpReader.NextNode(true)
		}

		peekIndex, peekNode = tmpReader.PeekNode(true)

		// Don't include SQL keywords (except literals) in identifier list
		if item, ok := peekNode.(*ast.Item); ok {
			tok := item.GetToken()
			if tok.Kind == token.SQLKeyword {
				if sqlWord, ok := tok.Value.(*token.SQLWord); ok {
					keyword := sqlWord.Keyword
					// Only allow literal keywords
					if keyword != "TRUE" && keyword != "FALSE" && keyword != "NULL" {
						// Set endIndex to current position before breaking
						endIndex = tmpReader.Index - 1
						break
					}
				}
			}
		}

		idents = append(idents, peekNode)
		endIndex = peekIndex

		tmpReader.NextNode(true)
		if !tmpReader.PeekNodeIs(true, identifierListInfixMatcher) {
			break
		}
		tmpReader.NextNode(true)
		commas = append(commas, tmpReader.CurNode)
	}

	reader.Index = tmpReader.Index
	reader.CurNode = tmpReader.CurNode
	return &ast.IdentifierList{
		Toks:        reader.NodesWithRange(startIndex, endIndex+1),
		Identifiers: idents,
		Commas:      commas,
	}
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
		}
		nodes = append(nodes, tmpReader.CurNode)
	}

	// Not found close "END". Keep the nodes consumed so far instead of
	// returning the enclosing list, which would create a cyclic AST.
	reader.Index = tmpReader.Index
	reader.CurNode = tmpReader.CurNode
	return &ast.SwitchCase{Toks: nodes}
}

var expressionPrefixMatcher = astutil.NodeMatcher{
	NodeTypes: []ast.NodeType{
		ast.TypeParenthesis,
	},
}

func parseExpressionInParenthesis(reader *astutil.NodeReader) ast.Node {
	if list, ok := reader.CurNode.(ast.TokenList); ok {
		list = parseInfixGroup(astutil.NewNodeReader(list), operatorInfixMatcher, true, parseOperator)
		list = parseInfixGroup(astutil.NewNodeReader(list), comparisonInfixMatcher, true, parseComparison)
		return list
	}
	return reader.CurNode
}
