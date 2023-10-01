package parseutil

import (
	"github.com/hsanson/sqls/ast"
	"github.com/hsanson/sqls/ast/astutil"
	"github.com/hsanson/sqls/token"
)

func ExtractIdenfiers(parsed ast.TokenList, pos token.Pos) ([]ast.Node, error) {
	stmt, err := extractFocusedStatement(parsed, pos)
	if err != nil {
		return nil, err
	}

	identiferMatcher := astutil.NodeMatcher{
		NodeTypes: []ast.NodeType{
			ast.TypeIdentifer,
		},
	}
	return parsePrefix(astutil.NewNodeReader(stmt), identiferMatcher, parseIdentifer), nil
}

func parseIdentifer(reader *astutil.NodeReader) []ast.Node {
	return []ast.Node{reader.CurNode}
}
