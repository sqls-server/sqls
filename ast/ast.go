package ast

import (
	"strings"

	"github.com/lighttiger2505/sqls/dialect"
	"github.com/lighttiger2505/sqls/token"
)

type Node interface {
	String() string
}

type Token interface {
	Node
	GetToken() *SQLToken
}

type TokenList interface {
	Node
	GetTokens() []Node
	SetTokens([]Node)
}

type Item struct {
	Tok *SQLToken
}

func NewItem(tok *token.Token) Node {
	return &Item{NewSQLToken(tok)}
}
func (i *Item) String() string      { return i.Tok.String() }
func (i *Item) GetToken() *SQLToken { return i.Tok }

type MemberIdentifer struct {
	Parent *SQLToken
	Period *SQLToken
	Child  *SQLToken
}

func (mi *MemberIdentifer) String() string {
	res := mi.Parent.String() + mi.Period.String()
	if mi.Child != nil {
		res = res + mi.Child.String()
	}
	return res
}
func (mi *MemberIdentifer) GetTokens() []Node {
	res := []Node{mi.Parent, mi.Period}
	if mi.Child != nil {
		res = append(res, mi.Child)
	}
	return res
}
func (mi *MemberIdentifer) SetTokens(toks []Node) {}

type Aliased struct {
	Toks []Node
}

func (a *Aliased) String() string {
	var strs []string
	for _, t := range a.Toks {
		strs = append(strs, t.String())
	}
	return strings.Join(strs, "")
}
func (a *Aliased) GetTokens() []Node     { return a.Toks }
func (a *Aliased) SetTokens(toks []Node) { a.Toks = toks }

type Identifer struct {
	Tok *SQLToken
}

func (i *Identifer) String() string      { return i.Tok.String() }
func (i *Identifer) GetToken() *SQLToken { return i.Tok }

type Parenthesis struct {
	Toks []Node
}

func (p *Parenthesis) String() string {
	var strs []string
	for _, t := range p.Toks {
		strs = append(strs, t.String())
	}
	return strings.Join(strs, "")
}
func (p *Parenthesis) GetTokens() []Node     { return p.Toks }
func (p *Parenthesis) SetTokens(toks []Node) { p.Toks = toks }

type Where struct {
	Toks []Node
}

func (w *Where) String() string {
	var strs []string
	for _, t := range w.Toks {
		strs = append(strs, t.String())
	}
	return strings.Join(strs, "")
}
func (w *Where) GetTokens() []Node     { return w.Toks }
func (w *Where) SetTokens(toks []Node) { w.Toks = toks }

type Query struct {
	Toks []Node
}

func (q *Query) String() string {
	var strs []string
	for _, t := range q.Toks {
		strs = append(strs, t.String())
	}
	return strings.Join(strs, "")
}
func (q *Query) GetTokens() []Node     { return q.Toks }
func (q *Query) SetTokens(toks []Node) { q.Toks = toks }

type Statement struct {
	Toks []Node
}

func (s *Statement) String() string {
	var strs []string
	for _, t := range s.Toks {
		strs = append(strs, t.String())
	}
	return strings.Join(strs, "")
}
func (s *Statement) GetTokens() []Node     { return s.Toks }
func (s *Statement) SetTokens(toks []Node) { s.Toks = toks }

type SQLToken struct {
	Node
	Kind  token.Kind
	Value interface{}
	From  token.Pos
	To    token.Pos
}

func NewSQLToken(tok *token.Token) *SQLToken {
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

func (t *SQLToken) MatchSQLKeyword(expect string) bool {
	if t.Kind != token.SQLKeyword {
		return false
	}
	sqlWord, _ := t.Value.(*token.SQLWord)
	return sqlWord.Keyword == expect
}

func (t *SQLToken) MatchSQLKeywords(expects []string) bool {
	for _, expect := range expects {
		if t.MatchSQLKeyword(expect) {
			return true
		}
	}
	return false
}

func (t *SQLToken) String() string {
	switch v := t.Value.(type) {
	case *token.SQLWord:
		return v.String()
	case string:
		return v
	default:
		return " "
	}
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
