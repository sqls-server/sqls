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
	Toks []Node
}

func (mi *MemberIdentifer) String() string {
	var strs []string
	for _, t := range mi.Toks {
		strs = append(strs, t.String())
	}
	return strings.Join(strs, "")
}
func (mi *MemberIdentifer) GetTokens() []Node     { return mi.Toks }
func (mi *MemberIdentifer) SetTokens(toks []Node) { mi.Toks = toks }

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

type Operator struct {
	Toks []Node
}

func (o *Operator) String() string {
	var strs []string
	for _, t := range o.Toks {
		strs = append(strs, t.String())
	}
	return strings.Join(strs, "")
}
func (o *Operator) GetTokens() []Node     { return o.Toks }
func (o *Operator) SetTokens(toks []Node) { o.Toks = toks }

type Comparison struct {
	Toks []Node
}

func (c *Comparison) String() string {
	var strs []string
	for _, t := range c.Toks {
		strs = append(strs, t.String())
	}
	return strings.Join(strs, "")
}
func (c *Comparison) GetTokens() []Node     { return c.Toks }
func (c *Comparison) SetTokens(toks []Node) { c.Toks = toks }

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

type Function struct {
	Toks []Node
}

func (f *Function) String() string {
	var strs []string
	for _, t := range f.Toks {
		strs = append(strs, t.String())
	}
	return strings.Join(strs, "")
}
func (f *Function) GetTokens() []Node     { return f.Toks }
func (f *Function) SetTokens(toks []Node) { f.Toks = toks }

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

type From struct {
	Toks []Node
}

func (f *From) String() string {
	var strs []string
	for _, t := range f.Toks {
		strs = append(strs, t.String())
	}
	return strings.Join(strs, "")
}
func (f *From) GetTokens() []Node     { return f.Toks }
func (f *From) SetTokens(toks []Node) { f.Toks = toks }

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

type IdentiferList struct {
	Toks []Node
}

func (il *IdentiferList) String() string {
	var strs []string
	for _, t := range il.Toks {
		strs = append(strs, t.String())
	}
	return strings.Join(strs, "")
}
func (il *IdentiferList) GetTokens() []Node     { return il.Toks }
func (il *IdentiferList) SetTokens(toks []Node) { il.Toks = toks }

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
	return strings.EqualFold(sqlWord.Keyword, expect)
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
