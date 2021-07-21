package parseutil

import (
	"github.com/lighttiger2505/sqls/ast"
	"github.com/lighttiger2505/sqls/ast/astutil"
	"github.com/lighttiger2505/sqls/token"
)

type SyntaxPosition string

const (
	ColName        SyntaxPosition = "col_name"
	SelectExpr     SyntaxPosition = "select_expr"
	AliasName      SyntaxPosition = "alias_name"
	WhereCondition SyntaxPosition = "where_conditon"
	CaseValue      SyntaxPosition = "case_value"
	TableReference SyntaxPosition = "table_reference"
	InsertColumn   SyntaxPosition = "insert_column"
	InsertValue    SyntaxPosition = "insert_value"
	Unknown        SyntaxPosition = "unknown"
)

func CheckSyntaxPosition(nw *NodeWalker) SyntaxPosition {
	var res SyntaxPosition
	switch {
	case nw.PrevNodesIs(true, genKeywordMatcher([]string{
		// UPDATE Statement
		"SET",
		// SELECT Statement
		"ORDER BY",
		"GROUP BY",
	})):
		res = ColName
	case nw.PrevNodesIs(true, genKeywordMatcher([]string{
		// SELECT Statement
		"ALL",
		"DISTINCT",
		"DISTINCTROW",
		"SELECT",
	})):
		res = SelectExpr
	case nw.PrevNodesIs(true, genKeywordMatcher([]string{
		// Alias
		"AS",
	})):
		res = AliasName
	case nw.PrevNodesIs(true, genKeywordMatcher([]string{
		// WHERE Clause
		"WHERE",
		"HAVING",
		// JOIN Clause
		"ON",
		// Operator
		"AND",
		"OR",
		"XOR",
	})):
		res = WhereCondition
	case nw.PrevNodesIs(true, genKeywordMatcher([]string{
		// CASE Statement
		"CASE",
		"WHEN",
		"THEN",
		"ELSE",
	})):
		res = CaseValue
	case nw.PrevNodesIs(true, genKeywordMatcher([]string{
		// SELECT Statement
		"FROM",
		// UPDATE Statement
		"UPDATE",
		// DELETE Statement
		"DELETE FROM",
		// INSERT Statement
		"INSERT INTO",
		// JOIN Clause
		"JOIN",
		"INNER JOIN",
		"CROSS JOIN",
		"OUTER JOIN",
		"LEFT JOIN",
		"RIGHT JOIN",
		"LEFT OUTER JOIN",
		"RIGHT OUTER JOIN",
		// DESCRIBE Statement
		"DESCRIBE",
		"DESC",
		// TRUNCATE Statement
		"TRUNCATE",
	})):
		res = TableReference
	case isInsertColumns(nw):
		if isInsertValues(nw) {
			res = InsertValue
		} else {
			res = InsertColumn
		}
	default:
		res = Unknown
	}
	return res
}

func genKeywordMatcher(keywords []string) astutil.NodeMatcher {
	return astutil.NodeMatcher{
		ExpectKeyword: keywords,
	}
}

func genTokenMatcher(tokens []token.Kind) astutil.NodeMatcher {
	return astutil.NodeMatcher{
		ExpectTokens: tokens,
	}
}

func isInsertColumns(nw *NodeWalker) bool {
	ParenthesisMatcher := astutil.NodeMatcher{
		NodeTypes: []ast.NodeType{
			ast.TypeParenthesis,
		},
	}
	return nw.CurNodeIs(ParenthesisMatcher)
}

func isInsertValues(nw *NodeWalker) bool {
	ParenthesisMatcher := astutil.NodeMatcher{
		NodeTypes: []ast.NodeType{
			ast.TypeParenthesis,
		},
	}
	depth, ok := nw.CurNodeDepth(ParenthesisMatcher)
	if ok {
		if nw.PrevNodesIsWithDepth(true, genKeywordMatcher([]string{"VALUES"}), depth) {
			return true
		}
		if nw.PrevNodesIsWithDepth(true, genTokenMatcher([]token.Kind{token.Comma}), depth) {
			return true
		}
	}
	return false
}
