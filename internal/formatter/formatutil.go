package formatter

import (
	"github.com/sqls-server/sqls/ast"
	"github.com/sqls-server/sqls/token"
)

func unshift(slice []ast.Node, node ...ast.Node) []ast.Node {
	return append(node, slice...)
}

var whitespaceNode = ast.NewItem(&token.Token{
	Kind:  token.Whitespace,
	Value: " ",
})

func whiteSpaceNodes(num int) []ast.Node {
	res := make([]ast.Node, num)
	for i := 0; i < num; i++ {
		res[i] = whitespaceNode
	}
	return res
}

var linebreakNode = ast.NewItem(&token.Token{
	Kind:  token.Whitespace,
	Value: "\n",
})

var tabNode = ast.NewItem(&token.Token{
	Kind:  token.Whitespace,
	Value: "\t",
})

var periodNode = ast.NewItem(&token.Token{
	Kind:  token.Period,
	Value: ".",
})

var lparenNode = ast.NewItem(&token.Token{
	Kind:  token.LParen,
	Value: "(",
})

var rparenNode = ast.NewItem(&token.Token{
	Kind:  token.RParen,
	Value: ")",
})

var commaNode = ast.NewItem(&token.Token{
	Kind:  token.Comma,
	Value: ",",
})
