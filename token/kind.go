package token

type Kind int

//go:generate stringer -type Kind kind.go
const (
	// A keyword (like SELECT)
	SQLKeyword Kind = iota
	// Numeric literal
	Number
	// A character that cloud not be tokenized
	Char
	// Single quoted string i.e: 'string'
	SingleQuotedString
	// National string i.e: N'string'
	NationalStringLiteral
	// Comma
	Comma
	// Whitespace
	Whitespace
	// comment node
	Comment
	// multiline comment node
	MultilineComment
	// = operator
	Eq
	// != or <> operator
	Neq
	// <  operator
	Lt
	// > operator
	Gt
	// <= operator
	LtEq
	// >= operator
	GtEq
	// + operator
	Plus
	// - operator
	Minus
	// * operator
	Mult
	// / operator
	Div
	// % operator
	Caret
	// ^ operator
	Mod
	// Left parenthesis `(`
	LParen
	// Right parenthesis `)`
	RParen
	// Period
	Period
	// Colon
	Colon
	// DoubleColon
	DoubleColon
	// Semicolon
	Semicolon
	// Backslash
	Backslash
	// Left bracket `]`
	LBracket
	// Right bracket `[`
	RBracket
	// &
	Ampersand
	// Left brace `{`
	LBrace
	// Right brace `}`
	RBrace
	// ILLEGAL sqltoken
	ILLEGAL
)
