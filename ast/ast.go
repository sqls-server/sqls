package ast

import (
	"github.com/lighttiger2505/sqls/dialect"
	"github.com/lighttiger2505/sqls/token"
)

type Node interface {
	// Pos() token.Pos
	// End() token.Pos
	String() string
}

type Token interface {
	Node
	Token() *SQLToken
}

type TokenList interface {
	Node
	Tokens() []*SQLToken
}

type SQLToken struct {
	Node
	Kind  token.Kind
	Value interface{}
	From  token.Pos
	To    token.Pos
}

func NewSQLToken(tok *token.Token) Node {
	return &SQLToken{
		Kind:  tok.Kind,
		Value: tok.Value,
		From:  tok.From,
		To:    tok.To,
	}
}

func (t *SQLToken) MatchKind(expect token.Kind) bool {
	return t.Kind == expect
}

func (t *SQLToken) MatchSQLKind(expect dialect.KeywordKind) bool {
	if t.Kind != token.SQLKeyword {
		return false
	}
	sqlWord, _ := t.Value.(*token.SQLWord)
	return sqlWord.Kind == expect
}

type SQLTokenList struct {
	Node
	Children []Node
}

func NewSQLTokenList(children []Node) Node {
	return &SQLTokenList{Children: children}
}

// Token
//   TokenList
//     Statement
//     Identifier
//     IdentifierList
//     TypedLiteral
//     Parenthesis
//     SquareBrakets
//     If
//     For
//     Comparison
//     Comment
//     Where
//     Having
//     Case
//     Function
//     Begin
