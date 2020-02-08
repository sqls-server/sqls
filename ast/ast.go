package ast

import (
	"strings"

	"github.com/lighttiger2505/sqls/dialect"
	"github.com/lighttiger2505/sqls/token"
)

type Node interface {
	String() string
	Pos() token.Pos
	End() token.Pos
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
func (i *Item) Pos() token.Pos      { return i.Tok.From }
func (i *Item) End() token.Pos      { return i.Tok.To }

type MemberIdentifer struct {
	Toks   []Node
	Parent Node
	Child  Node
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
func (mi *MemberIdentifer) Pos() token.Pos        { return findFrom(mi) }
func (mi *MemberIdentifer) End() token.Pos        { return findTo(mi) }

type Aliased struct {
	Toks        []Node
	RealName    Node
	AliasedName Node
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
func (a *Aliased) Pos() token.Pos        { return findFrom(a) }
func (a *Aliased) End() token.Pos        { return findTo(a) }

type Identifer struct {
	Tok *SQLToken
}

func (i *Identifer) String() string      { return i.Tok.String() }
func (i *Identifer) GetToken() *SQLToken { return i.Tok }
func (i *Identifer) Pos() token.Pos      { return findFrom(i) }
func (i *Identifer) End() token.Pos      { return findTo(i) }

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
func (o *Operator) Pos() token.Pos        { return findFrom(o) }
func (o *Operator) End() token.Pos        { return findTo(o) }

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
func (c *Comparison) Pos() token.Pos        { return findFrom(c) }
func (c *Comparison) End() token.Pos        { return findTo(c) }

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
func (p *Parenthesis) Pos() token.Pos        { return findFrom(p) }
func (p *Parenthesis) End() token.Pos        { return findTo(p) }

type FunctionLiteral struct {
	Toks []Node
}

func (fl *FunctionLiteral) String() string {
	var strs []string
	for _, t := range fl.Toks {
		strs = append(strs, t.String())
	}
	return strings.Join(strs, "")
}
func (fl *FunctionLiteral) GetTokens() []Node     { return fl.Toks }
func (fl *FunctionLiteral) SetTokens(toks []Node) { fl.Toks = toks }
func (fl *FunctionLiteral) Pos() token.Pos        { return findFrom(fl) }
func (fl *FunctionLiteral) End() token.Pos        { return findTo(fl) }

type WhereClause struct {
	Toks []Node
}

func (w *WhereClause) String() string {
	var strs []string
	for _, t := range w.Toks {
		strs = append(strs, t.String())
	}
	return strings.Join(strs, "")
}
func (w *WhereClause) GetTokens() []Node     { return w.Toks }
func (w *WhereClause) SetTokens(toks []Node) { w.Toks = toks }
func (w *WhereClause) Pos() token.Pos        { return findFrom(w) }
func (w *WhereClause) End() token.Pos        { return findTo(w) }

type FromClause struct {
	Toks []Node
}

func (f *FromClause) String() string {
	var strs []string
	for _, t := range f.Toks {
		strs = append(strs, t.String())
	}
	return strings.Join(strs, "")
}
func (f *FromClause) GetTokens() []Node     { return f.Toks }
func (f *FromClause) SetTokens(toks []Node) { f.Toks = toks }
func (f *FromClause) Pos() token.Pos        { return findFrom(f) }
func (f *FromClause) End() token.Pos        { return findTo(f) }

type JoinClause struct {
	Toks []Node
}

func (j *JoinClause) String() string {
	var strs []string
	for _, t := range j.Toks {
		strs = append(strs, t.String())
	}
	return strings.Join(strs, "")
}
func (j *JoinClause) GetTokens() []Node     { return j.Toks }
func (j *JoinClause) SetTokens(toks []Node) { j.Toks = toks }
func (j *JoinClause) Pos() token.Pos        { return findFrom(j) }
func (j *JoinClause) End() token.Pos        { return findTo(j) }

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
func (q *Query) Pos() token.Pos        { return findFrom(q) }
func (q *Query) End() token.Pos        { return findTo(q) }

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
func (s *Statement) Pos() token.Pos        { return findFrom(s) }
func (s *Statement) End() token.Pos        { return findTo(s) }

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
func (il *IdentiferList) Pos() token.Pos        { return findFrom(il) }
func (il *IdentiferList) End() token.Pos        { return findTo(il) }

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

func findFrom(node Node) token.Pos {
	if list, ok := node.(TokenList); ok {
		nodes := list.GetTokens()
		return findFrom(nodes[0])
	}
	return node.Pos()
}

func findTo(node Node) token.Pos {
	if list, ok := node.(TokenList); ok {
		nodes := list.GetTokens()
		return findFrom(nodes[len(nodes)-1])
	}
	return node.Pos()
}
