package ast

import (
	"strings"

	"github.com/sqls-server/sqls/dialect"
	"github.com/sqls-server/sqls/token"
)

type NodeType int

const (
	TypeItem NodeType = iota
	TypeMultiKeyword
	TypeMemberIdentifier
	TypeAliased
	TypeIdentifier
	TypeOperator
	TypeComparison
	TypeParenthesis
	TypeParenthesisInner
	TypeFunctionLiteral
	TypeQuery
	TypeStatement
	TypeIdentifierList
	TypeSwitchCase
	TypeNull
)

type RenderOptions struct {
	LowerCase        bool
	IdentifierQuoted bool
}

type Node interface {
	String() string
	Render(opts *RenderOptions) string
	Type() NodeType
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

type Null struct{}

func (n *Null) String() string                    { return "" }
func (n *Null) Render(opts *RenderOptions) string { return n.String() }
func (n *Null) Type() NodeType                    { return TypeNull }
func (n *Null) Pos() token.Pos                    { return token.Pos{} }
func (n *Null) End() token.Pos                    { return token.Pos{} }

type Item struct {
	Tok *SQLToken
}

func NewItem(tok *token.Token) Node {
	return &Item{NewSQLToken(tok)}
}
func (i *Item) String() string                    { return i.Tok.String() }
func (i *Item) NoQuoteString() string             { return i.Tok.NoQuoteString() }
func (i *Item) Render(opts *RenderOptions) string { return i.Tok.Render(opts) }
func (i *Item) Type() NodeType                    { return TypeItem }
func (i *Item) GetToken() *SQLToken               { return i.Tok }
func (i *Item) Pos() token.Pos                    { return i.Tok.From }
func (i *Item) End() token.Pos                    { return i.Tok.To }

type ItemWith struct {
	Toks []Node
}

func (iw *ItemWith) String() string {
	return joinString(iw.Toks)
}
func (iw *ItemWith) Render(opts *RenderOptions) string {
	return joinRender(iw.Toks, opts)
}
func (iw *ItemWith) Type() NodeType        { return TypeMultiKeyword }
func (iw *ItemWith) GetTokens() []Node     { return iw.Toks }
func (iw *ItemWith) SetTokens(toks []Node) { iw.Toks = toks }
func (iw *ItemWith) Pos() token.Pos        { return findFrom(iw) }
func (iw *ItemWith) End() token.Pos        { return findTo(iw) }

type MultiKeyword struct {
	Toks     []Node
	Keywords []Node
}

func (mk *MultiKeyword) String() string {
	return joinString(mk.Toks)
}
func (mk *MultiKeyword) Render(opts *RenderOptions) string {
	// Render keywords with spaces between them
	var parts []string
	for _, kw := range mk.Keywords {
		parts = append(parts, kw.Render(opts))
	}
	return strings.Join(parts, " ")
}
func (mk *MultiKeyword) Type() NodeType        { return TypeMultiKeyword }
func (mk *MultiKeyword) GetTokens() []Node     { return mk.Toks }
func (mk *MultiKeyword) SetTokens(toks []Node) { mk.Toks = toks }
func (mk *MultiKeyword) Pos() token.Pos        { return findFrom(mk) }
func (mk *MultiKeyword) End() token.Pos        { return findTo(mk) }
func (mk *MultiKeyword) GetKeywords() []Node   { return mk.Keywords }

type MemberIdentifier struct {
	Toks        []Node
	Parent      Node
	ParentTok   *SQLToken
	ParentIdent *Identifier
	Child       Node
	ChildTok    *SQLToken
	ChildIdent  *Identifier
}

func NewMemberIdentifierParent(nodes []Node, parent Node) *MemberIdentifier {
	memberIdentifier := &MemberIdentifier{Toks: nodes}
	memberIdentifier.setParent(parent)
	if v, ok := parent.(*Identifier); ok {
		memberIdentifier.ParentIdent = v
	}
	return memberIdentifier
}

func NewMemberIdentifier(nodes []Node, parent Node, child Node) *MemberIdentifier {
	memberIdentifier := &MemberIdentifier{Toks: nodes}
	memberIdentifier.setParent(parent)
	if v, ok := parent.(*Identifier); ok {
		memberIdentifier.ParentIdent = v
	}
	memberIdentifier.setChild(child)
	if v, ok := child.(*Identifier); ok {
		memberIdentifier.ChildIdent = v
	}
	return memberIdentifier
}

func (mi *MemberIdentifier) String() string {
	var strs []string
	for _, t := range mi.Toks {
		strs = append(strs, t.String())
	}
	return strings.Join(strs, "")
}
func (mi *MemberIdentifier) Render(opts *RenderOptions) string {
	var strs []string
	for _, t := range mi.Toks {
		strs = append(strs, t.Render(opts))
	}
	return strings.Join(strs, "")
}
func (mi *MemberIdentifier) Type() NodeType        { return TypeMemberIdentifier }
func (mi *MemberIdentifier) GetTokens() []Node     { return mi.Toks }
func (mi *MemberIdentifier) SetTokens(toks []Node) { mi.Toks = toks }
func (mi *MemberIdentifier) Pos() token.Pos        { return findFrom(mi) }
func (mi *MemberIdentifier) End() token.Pos        { return findTo(mi) }
func (mi *MemberIdentifier) setParent(node Node) {
	mi.Parent = node
	tok, ok := node.(Token)
	if ok {
		mi.ParentTok = tok.GetToken()
	}
}
func (mi *MemberIdentifier) setChild(node Node) {
	mi.Child = node
	tok, ok := node.(Token)
	if ok {
		mi.ChildTok = tok.GetToken()
	}
}
func (mi *MemberIdentifier) GetParent() Node {
	if mi.Parent == nil {
		return &Null{}
	}
	return mi.Parent
}
func (mi *MemberIdentifier) GetParentIdent() *Identifier {
	if mi.ParentIdent == nil {
		return &Identifier{}
	}
	return mi.ParentIdent
}
func (mi *MemberIdentifier) GetChild() Node {
	if mi.Child == nil {
		return &Null{}
	}
	return mi.Child
}
func (mi *MemberIdentifier) GetChildIdent() *Identifier {
	if mi.ChildIdent == nil {
		return &Identifier{}
	}
	return mi.ChildIdent
}

type Aliased struct {
	Toks        []Node
	RealName    Node
	AliasedName Node
	As          Node
	IsAs        bool
}

func (a *Aliased) String() string {
	var strs []string
	for _, t := range a.Toks {
		strs = append(strs, t.String())
	}
	return strings.Join(strs, "")
}
func (a *Aliased) Render(opts *RenderOptions) string {
	var strs []string
	for _, t := range a.Toks {
		strs = append(strs, t.Render(opts))
	}
	return strings.Join(strs, "")
}
func (a *Aliased) Type() NodeType        { return TypeAliased }
func (a *Aliased) GetTokens() []Node     { return a.Toks }
func (a *Aliased) SetTokens(toks []Node) { a.Toks = toks }
func (a *Aliased) Pos() token.Pos        { return findFrom(a) }
func (a *Aliased) End() token.Pos        { return findTo(a) }
func (a *Aliased) GetAliasedNameIdent() *Identifier {
	if v, ok := a.AliasedName.(*Identifier); ok {
		return v
	}
	return &Identifier{}
}

type Identifier struct {
	Tok *SQLToken
}

func (i *Identifier) Type() NodeType { return TypeIdentifier }
func (i *Identifier) String() string { return i.Tok.String() }
func (i *Identifier) Render(opts *RenderOptions) string {
	tmpOpts := &RenderOptions{
		LowerCase:        false,
		IdentifierQuoted: opts.IdentifierQuoted,
	}
	return i.Tok.Render(tmpOpts)
}
func (i *Identifier) NoQuoteString() string { return i.Tok.NoQuoteString() }
func (i *Identifier) GetToken() *SQLToken   { return i.Tok }
func (i *Identifier) Pos() token.Pos        { return i.Tok.From }
func (i *Identifier) End() token.Pos        { return i.Tok.To }
func (i *Identifier) IsWildcard() bool      { return i.Tok.MatchKind(token.Mult) }

type Operator struct {
	Toks     []Node
	Left     Node
	Operator Node
	Right    Node
}

func (o *Operator) String() string {
	return joinString(o.Toks)
}
func (o *Operator) Render(opts *RenderOptions) string {
	return joinRender(o.Toks, opts)
}
func (o *Operator) Type() NodeType        { return TypeOperator }
func (o *Operator) GetTokens() []Node     { return o.Toks }
func (o *Operator) SetTokens(toks []Node) { o.Toks = toks }
func (o *Operator) Pos() token.Pos        { return findFrom(o) }
func (o *Operator) End() token.Pos        { return findTo(o) }
func (o *Operator) GetLeft() Node {
	if o.Left == nil {
		return &Null{}
	}
	return o.Left
}
func (o *Operator) GetOperator() Node {
	if o.Operator == nil {
		return &Null{}
	}
	return o.Operator
}
func (o *Operator) GetRight() Node {
	if o.Right == nil {
		return &Null{}
	}
	return o.Right
}

type Comparison struct {
	Toks       []Node
	Left       Node
	Comparison Node
	Right      Node
}

func (c *Comparison) String() string {
	return joinString(c.Toks)
}
func (c *Comparison) Render(opts *RenderOptions) string {
	return joinRender(c.Toks, opts)
}
func (c *Comparison) Type() NodeType        { return TypeComparison }
func (c *Comparison) GetTokens() []Node     { return c.Toks }
func (c *Comparison) SetTokens(toks []Node) { c.Toks = toks }
func (c *Comparison) Pos() token.Pos        { return findFrom(c) }
func (c *Comparison) End() token.Pos        { return findTo(c) }
func (c *Comparison) GetLeft() Node {
	if c.Left == nil {
		return &Null{}
	}
	return c.Left
}
func (c *Comparison) GetComparison() Node {
	if c.Comparison == nil {
		return &Null{}
	}
	return c.Comparison
}
func (c *Comparison) GetRight() Node {
	if c.Right == nil {
		return &Null{}
	}
	return c.Right
}

type Parenthesis struct {
	Toks []Node
}

func (p *Parenthesis) String() string {
	return joinString(p.Toks)
}
func (p *Parenthesis) Render(opts *RenderOptions) string {
	return joinRender(p.Toks, opts)
}
func (p *Parenthesis) Type() NodeType        { return TypeParenthesis }
func (p *Parenthesis) GetTokens() []Node     { return p.Toks }
func (p *Parenthesis) SetTokens(toks []Node) { p.Toks = toks }
func (p *Parenthesis) Pos() token.Pos        { return findFrom(p) }
func (p *Parenthesis) End() token.Pos        { return findTo(p) }
func (p *Parenthesis) Inner() TokenList {
	endPos := len(p.Toks) - 1
	if p.Toks[endPos].String() != ")" {
		endPos = len(p.Toks)
	}
	return &ParenthesisInner{Toks: p.Toks[1:endPos]}
}

type ParenthesisInner struct {
	Toks []Node
}

func (p *ParenthesisInner) String() string {
	return joinString(p.Toks)
}
func (p *ParenthesisInner) Render(opts *RenderOptions) string {
	return joinRender(p.Toks, opts)
}
func (p *ParenthesisInner) Type() NodeType        { return TypeParenthesisInner }
func (p *ParenthesisInner) GetTokens() []Node     { return p.Toks }
func (p *ParenthesisInner) SetTokens(toks []Node) { p.Toks = toks }
func (p *ParenthesisInner) Pos() token.Pos        { return findFrom(p) }
func (p *ParenthesisInner) End() token.Pos        { return findTo(p) }

type FunctionLiteral struct {
	Toks []Node
}

func (fl *FunctionLiteral) String() string {
	return joinString(fl.Toks)
}
func (fl *FunctionLiteral) Render(opts *RenderOptions) string {
	return joinRender(fl.Toks, opts)
}
func (fl *FunctionLiteral) Type() NodeType        { return TypeFunctionLiteral }
func (fl *FunctionLiteral) GetTokens() []Node     { return fl.Toks }
func (fl *FunctionLiteral) SetTokens(toks []Node) { fl.Toks = toks }
func (fl *FunctionLiteral) Pos() token.Pos        { return findFrom(fl) }
func (fl *FunctionLiteral) End() token.Pos        { return findTo(fl) }

type Query struct {
	Toks []Node
}

func (q *Query) String() string {
	return joinString(q.Toks)
}
func (q *Query) Render(opts *RenderOptions) string {
	return joinRender(q.Toks, opts)
}
func (q *Query) Type() NodeType        { return TypeQuery }
func (q *Query) GetTokens() []Node     { return q.Toks }
func (q *Query) SetTokens(toks []Node) { q.Toks = toks }
func (q *Query) Pos() token.Pos        { return findFrom(q) }
func (q *Query) End() token.Pos        { return findTo(q) }

type Statement struct {
	Toks []Node
}

func (s *Statement) String() string {
	return joinString(s.Toks)
}
func (s *Statement) Render(opts *RenderOptions) string {
	return joinRender(s.Toks, opts)
}
func (s *Statement) Type() NodeType        { return TypeStatement }
func (s *Statement) GetTokens() []Node     { return s.Toks }
func (s *Statement) SetTokens(toks []Node) { s.Toks = toks }
func (s *Statement) Pos() token.Pos        { return findFrom(s) }
func (s *Statement) End() token.Pos        { return findTo(s) }

type IdentifierList struct {
	Toks        []Node
	Identifiers []Node
	Commas      []Node
}

func (il *IdentifierList) String() string {
	return joinString(il.Toks)
}
func (il *IdentifierList) Render(opts *RenderOptions) string {
	return joinRender(il.Toks, opts)
}
func (il *IdentifierList) Type() NodeType         { return TypeIdentifierList }
func (il *IdentifierList) GetTokens() []Node      { return il.Toks }
func (il *IdentifierList) SetTokens(toks []Node)  { il.Toks = toks }
func (il *IdentifierList) Pos() token.Pos         { return findFrom(il) }
func (il *IdentifierList) End() token.Pos         { return findTo(il) }
func (il *IdentifierList) GetIdentifiers() []Node { return il.Identifiers }
func (il *IdentifierList) GetIndex(pos token.Pos) int {
	if !isEnclose(il, pos) {
		return -1
	}
	var idx int
	for _, comma := range il.Commas {
		if 0 > token.ComparePos(comma.Pos(), pos) {
			idx++
		}
	}
	return idx
}

func isEnclose(node Node, pos token.Pos) bool {
	if 0 <= token.ComparePos(pos, node.Pos()) && 0 >= token.ComparePos(pos, node.End()) {
		return true
	}
	return false
}

type SwitchCase struct {
	Toks []Node
}

func (sc *SwitchCase) String() string {
	return joinString(sc.Toks)
}
func (sc *SwitchCase) Render(opts *RenderOptions) string {
	return joinRender(sc.Toks, opts)
}
func (sc *SwitchCase) Type() NodeType        { return TypeSwitchCase }
func (sc *SwitchCase) GetTokens() []Node     { return sc.Toks }
func (sc *SwitchCase) SetTokens(toks []Node) { sc.Toks = toks }
func (sc *SwitchCase) Pos() token.Pos        { return findFrom(sc) }
func (sc *SwitchCase) End() token.Pos        { return findTo(sc) }

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
		if t.Kind == token.Comment {
			return "--" + v
		}
		if t.Kind == token.MultilineComment {
			return "/*" + v + "*/"
		}
		return v
	default:
		return " "
	}
}

func (t *SQLToken) NoQuoteString() string {
	switch v := t.Value.(type) {
	case *token.SQLWord:
		return v.NoQuoteString()
	case string:
		if t.Kind == token.Comment {
			return "--" + v
		}
		if t.Kind == token.MultilineComment {
			return "/*" + v + "*/"
		}
		return v
	default:
		return " "
	}
}

func (t *SQLToken) Render(opts *RenderOptions) string {
	switch v := t.Value.(type) {
	case *token.SQLWord:
		return renderSQLWord(v, opts)
	case string:
		if t.Kind == token.Comment {
			return "--" + v
		}
		if t.Kind == token.MultilineComment {
			return "/*" + v + "*/"
		}
		return v
	default:
		return " "
	}
}

func renderSQLWord(v *token.SQLWord, opts *RenderOptions) string {
	isIdentifier := v.Kind == dialect.Unmatched
	if isIdentifier {
		if opts.IdentifierQuoted {
			v.QuoteStyle = '`'
			return v.String()
		}
		return v.String()
	}
	// is keyword
	if opts.LowerCase {
		return strings.ToLower(v.String())
	}
	return strings.ToUpper(v.String())
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
		return findTo(nodes[len(nodes)-1])
	}
	return node.End()
}

func joinString(nodes []Node) string {
	var strs []string
	for _, n := range nodes {
		strs = append(strs, n.String())
	}
	return strings.Join(strs, "")
}

func joinRender(nodes []Node, opts *RenderOptions) string {
	var strs []string
	for _, n := range nodes {
		strs = append(strs, n.Render(opts))
	}
	return strings.Join(strs, "")
}
