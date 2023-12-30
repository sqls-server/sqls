package parseutil

import (
	"github.com/sqls-server/sqls/ast"
	"github.com/sqls-server/sqls/ast/astutil"
	"github.com/sqls-server/sqls/token"
)

type (
	prefixParseFn func(reader *astutil.NodeReader) []ast.Node
	infixParseFn  func(reader *astutil.NodeReader) []ast.Node
)

func parsePrefix(reader *astutil.NodeReader, matcher astutil.NodeMatcher, fn prefixParseFn) []ast.Node {
	var replaceNodes []ast.Node
	for reader.NextNode(false) {
		if reader.CurNodeIs(matcher) {
			replaceNodes = append(replaceNodes, fn(reader)...)
		} else if list, ok := reader.CurNode.(ast.TokenList); ok {
			newReader := astutil.NewNodeReader(list)
			replaceNodes = append(replaceNodes, parsePrefix(newReader, matcher, fn)...)
		}
	}
	return replaceNodes
}

func ExtractSelectExpr(parsed ast.TokenList) []ast.Node {
	prefixMatcher := astutil.NodeMatcher{
		ExpectKeyword: []string{
			"SELECT",
			"ALL",
			"DISTINCT",
		},
	}
	peekMatcher := astutil.NodeMatcher{
		NodeTypes: []ast.NodeType{
			ast.TypeIdentifierList,
			ast.TypeIdentifier,
			ast.TypeMemberIdentifier,
			ast.TypeOperator,
			ast.TypeAliased,
			ast.TypeParenthesis,
			ast.TypeFunctionLiteral,
		},
	}
	return filterPrefixGroup(astutil.NewNodeReader(parsed), prefixMatcher, peekMatcher)
}

func ExtractTableReferences(parsed ast.TokenList) []ast.Node {
	prefixMatcher := astutil.NodeMatcher{
		ExpectKeyword: []string{
			"FROM",
			"UPDATE",
		},
	}
	peekMatcher := astutil.NodeMatcher{
		NodeTypes: []ast.NodeType{
			ast.TypeIdentifierList,
			ast.TypeIdentifier,
			ast.TypeMemberIdentifier,
			ast.TypeAliased,
		},
	}
	return filterPrefixGroupOnce(astutil.NewNodeReader(parsed), prefixMatcher, peekMatcher)
}

func ExtractTableReference(parsed ast.TokenList) []ast.Node {
	prefixMatcher := astutil.NodeMatcher{
		ExpectKeyword: []string{
			"INSERT INTO",
			"DELETE FROM",
		},
	}
	peekMatcher := astutil.NodeMatcher{
		NodeTypes: []ast.NodeType{
			ast.TypeIdentifier,
			ast.TypeMemberIdentifier,
			ast.TypeAliased,
		},
	}
	return filterPrefixGroup(astutil.NewNodeReader(parsed), prefixMatcher, peekMatcher)
}

func ExtractTableFactor(parsed ast.TokenList) []ast.Node {
	prefixMatcher := astutil.NodeMatcher{
		ExpectKeyword: []string{
			"JOIN",
			"INNER JOIN",
			"CROSS JOIN",
			"OUTER JOIN",
			"LEFT JOIN",
			"RIGHT JOIN",
			"LEFT OUTER JOIN",
			"RIGHT OUTER JOIN",
		},
	}
	peekMatcher := astutil.NodeMatcher{
		NodeTypes: []ast.NodeType{
			ast.TypeIdentifier,
			ast.TypeMemberIdentifier,
			ast.TypeAliased,
		},
	}
	return filterPrefixGroup(astutil.NewNodeReader(parsed), prefixMatcher, peekMatcher)
}

func ExtractWhereCondition(parsed ast.TokenList) []ast.Node {
	prefixMatcher := astutil.NodeMatcher{
		ExpectKeyword: []string{
			"WHERE",
		},
	}
	peekMatcher := astutil.NodeMatcher{
		NodeTypes: []ast.NodeType{
			ast.TypeComparison,
			ast.TypeIdentifierList,
		},
	}
	return filterPrefixGroup(astutil.NewNodeReader(parsed), prefixMatcher, peekMatcher)
}

func ExtractAliased(parsed ast.TokenList) []ast.Node {
	reader := astutil.NewNodeReader(parsed)
	matcher := astutil.NodeMatcher{NodeTypes: []ast.NodeType{ast.TypeAliased}}
	aliases := reader.FindRecursive(matcher)
	return aliases
}

func ExtractAliasedIdentifier(parsed ast.TokenList) []ast.Node {
	reader := astutil.NewNodeReader(parsed)
	matcher := astutil.NodeMatcher{NodeTypes: []ast.NodeType{ast.TypeAliased}}
	aliases := reader.FindRecursive(matcher)

	results := []ast.Node{}
	for _, node := range aliases {
		alias, ok := node.(*ast.Aliased)
		if !ok {
			continue
		}
		list, ok := alias.RealName.(ast.TokenList)
		if !ok {
			results = append(results, node)
			continue
		}
		if isSubQuery(list) {
			continue
		}
		results = append(results, node)
	}
	return results
}

func ExtractInsertColumns(parsed ast.TokenList) []ast.Node {
	insertTableIdentifier := astutil.NodeMatcher{
		NodeTypes: []ast.NodeType{
			ast.TypeIdentifier,
			ast.TypeMemberIdentifier,
			ast.TypeAliased,
		},
	}
	return parsePrefix(astutil.NewNodeReader(parsed), insertTableIdentifier, parseInsertColumns)
}

func parseInsertColumns(reader *astutil.NodeReader) []ast.Node {
	insertColumnsParenthesis := astutil.NodeMatcher{
		NodeTypes: []ast.NodeType{
			ast.TypeParenthesis,
		},
	}

	if !reader.PeekNodeIs(true, insertColumnsParenthesis) {
		return []ast.Node{}
	}

	_, parenthesisNode := reader.PeekNode(true)
	parenthesis, ok := parenthesisNode.(*ast.Parenthesis)
	if !ok {
		return []ast.Node{}
	}

	inner, ok := parenthesis.Inner().(*ast.IdentifierList)
	if ok {
		return []ast.Node{inner}
	}
	list := parenthesis.Inner().GetTokens()
	if len(list) > 0 {
		firstToken, ok := list[0].(*ast.IdentifierList)
		if ok {
			return []ast.Node{firstToken}
		}
	}
	return []ast.Node{}
}

func ExtractInsertValues(parsed ast.TokenList, pos token.Pos) []ast.Node {
	insertTableIdentifier := astutil.NodeMatcher{
		ExpectTokens: []token.Kind{
			token.Comma,
		},
		ExpectKeyword: []string{
			"VALUES",
		},
	}
	values := parsePrefix(astutil.NewNodeReader(parsed), insertTableIdentifier, parseInsertValues)
	for _, v := range values {
		if astutil.IsEnclose(v, pos) {
			return []ast.Node{v}
		}
	}
	return []ast.Node{}
}

func parseInsertValues(reader *astutil.NodeReader) []ast.Node {
	insertColumnsParenthesis := astutil.NodeMatcher{
		NodeTypes: []ast.NodeType{
			ast.TypeParenthesis,
		},
	}
	if !reader.PeekNodeIs(true, insertColumnsParenthesis) {
		return []ast.Node{}
	}

	_, parenthesisNode := reader.PeekNode(true)
	parenthesis, ok := parenthesisNode.(*ast.Parenthesis)
	if !ok {
		return []ast.Node{}
	}
	identList, ok := parenthesis.Inner().GetTokens()[0].(*ast.IdentifierList)
	if !ok {
		return []ast.Node{}
	}
	return []ast.Node{identList}
}

func filterPrefixGroup(reader *astutil.NodeReader, prefixMatcher astutil.NodeMatcher, peekMatcher astutil.NodeMatcher) []ast.Node {
	var results []ast.Node
	for reader.NextNode(false) {
		if reader.CurNodeIs(prefixMatcher) && reader.PeekNodeIs(true, peekMatcher) {
			_, node := reader.PeekNode(true)
			results = append(results, node)
		}
		if list, ok := reader.CurNode.(ast.TokenList); ok {
			newReader := astutil.NewNodeReader(list)
			results = append(results, filterPrefixGroup(newReader, prefixMatcher, peekMatcher)...)
		}
	}
	return results
}

func filterPrefixGroupOnce(reader *astutil.NodeReader, prefixMatcher astutil.NodeMatcher, peekMatcher astutil.NodeMatcher) []ast.Node {
	results := filterPrefixGroup(reader, prefixMatcher, peekMatcher)
	if len(results) > 0 {
		return []ast.Node{results[0]}
	}
	return nil
}
