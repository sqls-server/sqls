package formatter

import (
	"github.com/lighttiger2505/sqls/ast"
	"github.com/lighttiger2505/sqls/token"
)

func unshift(slice []ast.Node, node ...ast.Node) []ast.Node {
	return append(node, slice...)
}

var whitespaceNode = ast.NewItem(&token.Token{
	Kind:  token.Whitespace,
	Value: " ",
})

var linebreakNode = ast.NewItem(&token.Token{
	Kind:  token.Whitespace,
	Value: "\n",
})

var indentNode = ast.NewItem(&token.Token{
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
