package token

import (
	"errors"
	"fmt"
	"io"
	"strings"
	"text/scanner"

	"github.com/sqls-server/sqls/dialect"
)

type SQLWord struct {
	Value      string
	QuoteStyle rune
	Keyword    string
	Kind       dialect.KeywordKind
}

func (s *SQLWord) String() string {
	switch s.QuoteStyle {
	case '"', '[', '`':
		return string(s.QuoteStyle) + s.Value + string(matchingEndQuote(s.QuoteStyle))
	case 0:
		return s.Value
	default:
		return ""
	}
}

func (s *SQLWord) NoQuoteString() string {
	return s.Value
}

func matchingEndQuote(quoteStyle rune) rune {
	switch quoteStyle {
	case '"':
		return '"'
	case '[':
		return ']'
	case '`':
		return '`'
	}
	return 0
}

func MakeKeyword(word string, quoteStyle rune) *SQLWord {
	w := strings.ToUpper(word)

	// Escaped identifier
	if quoteStyle != 0 {
		return &SQLWord{
			Value:      word,
			Keyword:    w,
			QuoteStyle: quoteStyle,
			Kind:       dialect.Unmatched,
		}
	}

	kind := dialect.MatchKeyword(w)
	return &SQLWord{
		Value:      word,
		Keyword:    w,
		QuoteStyle: quoteStyle,
		Kind:       kind,
	}
}

type Token struct {
	Kind  Kind
	Value interface{}
	From  Pos
	To    Pos
}

func NewPos(line, col int) Pos {
	return Pos{
		Line: line,
		Col:  col,
	}
}

type Pos struct {
	Line int
	Col  int
}

func (p *Pos) String() string {
	return fmt.Sprintf("{Line: %d Col: %d}", p.Line, p.Col)
}

func ComparePos(x, y Pos) int {
	if x.Line == y.Line && x.Col == y.Col {
		return 0
	}

	if x.Line > y.Line {
		return 1
	} else if x.Line < y.Line {
		return -1
	}

	if x.Col > y.Col {
		return 1
	}

	return -1
}

type Tokenizer struct {
	Dialect dialect.Dialect
	Scanner *scanner.Scanner
	Line    int
	Col     int
}

func NewTokenizer(src io.Reader, dialect dialect.Dialect) *Tokenizer {
	var scan scanner.Scanner
	return &Tokenizer{
		Dialect: dialect,
		Scanner: scan.Init(src),
		Line:    0,
		Col:     0,
	}
}

func (t *Tokenizer) Tokenize() ([]*Token, error) {
	var tokenset []*Token

	for {
		t, err := t.NextToken()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, err
		}
		tokenset = append(tokenset, t)
	}

	return tokenset, nil
}

func (t *Tokenizer) NextToken() (*Token, error) {
	pos := t.Pos()
	tok, str, err := t.next()
	if errors.Is(err, io.EOF) {
		return nil, io.EOF
	}
	if err != nil {
		return &Token{Kind: ILLEGAL, Value: "", From: pos, To: t.Pos()}, fmt.Errorf("tokenize failed: %w", err)
	}

	return &Token{Kind: tok, Value: str, From: pos, To: t.Pos()}, nil
}

func (t *Tokenizer) Pos() Pos {
	return Pos{
		Line: t.Line,
		Col:  t.Col,
	}
}

func (t *Tokenizer) next() (Kind, interface{}, error) {
	r := t.Scanner.Peek()
	switch {
	case r == ' ':
		t.Scanner.Next()
		t.Col++
		return Whitespace, " ", nil

	case r == '\t':
		t.Scanner.Next()
		t.Col += 4
		return Whitespace, "\t", nil

	case r == '\n':
		t.Scanner.Next()
		t.Line++
		t.Col = 0
		return Whitespace, "\n", nil

	case r == '\r':
		t.Scanner.Next()
		n := t.Scanner.Peek()
		if n == '\n' {
			t.Scanner.Next()
		}
		t.Line++
		t.Col = 0
		return Whitespace, "\n", nil

	case r == 'N':
		t.Scanner.Next()
		n := t.Scanner.Peek()
		if n == '\'' {
			t.Col++
			str := t.tokenizeSingleQuotedString()
			return NationalStringLiteral, str, nil
		}
		s := t.tokenizeWord('N')
		v := MakeKeyword(s, 0)
		return SQLKeyword, v, nil

	case t.Dialect.IsIdentifierStart(r):
		t.Scanner.Next()
		s := t.tokenizeWord(r)
		return SQLKeyword, MakeKeyword(s, 0), nil

	case r == '\'':
		s := t.tokenizeSingleQuotedString()
		return SingleQuotedString, s, nil

	case t.Dialect.IsDelimitedIdentifierStart(r):
		s := t.tokenizeDelimitedIdentifier(r)
		return SQLKeyword, s, nil

	case '0' <= r && r <= '9':
		var s []rune
		hasE := false
		for {
			n := t.Scanner.Peek()
			if ('0' <= n && n <= '9') || n == '.' {
				s = append(s, n)
				t.Scanner.Next()
			} else if !hasE && (n == 'e' || n == 'E') {
				// Check for scientific notation
				s = append(s, n)
				t.Scanner.Next()
				hasE = true
				// Check for optional +/- after e/E
				next := t.Scanner.Peek()
				if next == '+' || next == '-' {
					s = append(s, next)
					t.Scanner.Next()
				}
			} else {
				break
			}
		}
		t.Col += len(s)
		return Number, string(s), nil

	case r == '(':
		t.Scanner.Next()
		t.Col++
		return LParen, "(", nil

	case r == ')':
		t.Scanner.Next()
		t.Col++
		return RParen, ")", nil

	case r == ',':
		t.Scanner.Next()
		t.Col++
		return Comma, ",", nil

	case r == '-':
		t.Scanner.Next()

		if t.Scanner.Peek() == '-' {
			t.Scanner.Next()

			var s []rune
			for {
				ch := t.Scanner.Peek()
				if ch != scanner.EOF && ch != '\n' {
					t.Scanner.Next()
					s = append(s, ch)
				} else {
					t.Col += len(s) + 2
					return Comment, string(s), nil // Comment Node
				}
			}
		}
		t.Col++
		return Minus, "-", nil

	case r == '/':
		t.Scanner.Next()

		if t.Scanner.Peek() == '*' {
			t.Scanner.Next()
			str, err := t.tokenizeMultilineComment()
			if err != nil {
				return ILLEGAL, str, err
			}
			return MultilineComment, str, nil
		}
		t.Col++
		return Div, "/", nil

	case r == '+':
		t.Scanner.Next()
		t.Col++
		return Plus, "+", nil
	case r == '*':
		t.Scanner.Next()
		t.Col++
		return Mult, "*", nil
	case r == '%':
		t.Scanner.Next()
		t.Col++
		return Mod, "%", nil
	case r == '^':
		t.Scanner.Next()
		t.Col++
		return Caret, "^", nil
	case r == '=':
		t.Scanner.Next()
		t.Col++
		return Eq, "=", nil
	case r == '.':
		t.Scanner.Next()
		t.Col++
		return Period, ".", nil

	case r == '!':
		t.Scanner.Next()
		n := t.Scanner.Peek()
		if n == '=' {
			t.Scanner.Next()
			t.Col += 2
			return Neq, "!=", nil
		}
		return ILLEGAL, "", fmt.Errorf("tokenizer error: illegal sequence %s%s", string(r), string(n))

	case r == '<':
		t.Scanner.Next()
		switch t.Scanner.Peek() {
		case '=':
			t.Scanner.Next()
			t.Col += 2
			return LtEq, "<=", nil
		case '>':
			t.Scanner.Next()
			t.Col += 2
			return Neq, "<>", nil
		default:
			t.Col++
			return Lt, "<", nil
		}
	case r == '>':
		t.Scanner.Next()
		switch t.Scanner.Peek() {
		case '=':
			t.Scanner.Next()
			t.Col += 2
			return GtEq, ">=", nil
		default:
			t.Col++
			return Gt, ">", nil
		}
	case r == ':':
		t.Scanner.Next()
		n := t.Scanner.Peek()
		if n == ':' {
			t.Scanner.Next()
			t.Col += 2
			return DoubleColon, "::", nil
		}
		t.Col++
		return Colon, ":", nil
	case r == ';':
		t.Scanner.Next()
		t.Col++
		return Semicolon, ";", nil
	case r == '\\':
		t.Scanner.Next()
		t.Col++
		return Backslash, "\\", nil
	case r == '[':
		t.Scanner.Next()
		t.Col++
		return LBracket, "[", nil
	case r == ']':
		t.Scanner.Next()
		t.Col++
		return RBracket, "]", nil
	case r == '&':
		t.Scanner.Next()
		t.Col++
		return Ampersand, "&", nil
	case r == '{':
		t.Scanner.Next()
		t.Col++
		return LBrace, "{", nil
	case r == '}':
		t.Scanner.Next()
		t.Col++
		return RBrace, "}", nil
	case scanner.EOF == r:
		return ILLEGAL, "", io.EOF
	default:
		t.Scanner.Next()
		t.Col++
		return Char, string(r), nil
	}
}

func (t *Tokenizer) tokenizeWord(f rune) string {
	var str []rune
	str = append(str, f)

	for {
		r := t.Scanner.Peek()
		if t.Dialect.IsIdentifierPart(r) {
			t.Scanner.Next()
			str = append(str, r)
		} else {
			break
		}
	}
	t.Col += len(str)
	return string(str)
}

func (t *Tokenizer) tokenizeSingleQuotedString() string {
	var str []rune
	t.Scanner.Next()
	isClosed := false

	for {
		n := t.Scanner.Peek()
		if n == '\'' {
			t.Scanner.Next()
			if t.Scanner.Peek() == '\'' {
				str = append(str, '\'')
				t.Scanner.Next()
			} else {
				isClosed = true
				break
			}
			continue
		}
		if n == scanner.EOF {
			break
		}

		t.Scanner.Next()
		str = append(str, n)
	}

	if isClosed {
		t.Col += 2 + len(str)
		return "'" + string(str) + "'"
	}
	t.Col += 1 + len(str)
	return "'" + string(str)
}

func (t *Tokenizer) tokenizeDelimitedIdentifier(r rune) *SQLWord {
	t.Scanner.Next()
	end := matchingEndQuote(r)
	isClosed := false

	var s []rune
	for {
		n := t.Scanner.Next()
		if n == scanner.EOF {
			break
		}
		if n == end {
			isClosed = true
			break
		}
		s = append(s, n)
		if t.Scanner.Peek() == ' ' {
			break
		}
	}

	if isClosed {
		t.Col += 2 + len(s)
		return MakeKeyword(string(s), r)
	}
	t.Col += 1 + len(s)
	return MakeKeyword(string(r)+string(s), 0)
}

func (t *Tokenizer) tokenizeMultilineComment() (string, error) {
	var str []rune
	var mayBeClosingComment bool
	t.Col += 2
	for {
		n := t.Scanner.Next()

		switch n {
		case '\r':
			if t.Scanner.Peek() == '\n' {
				t.Scanner.Next()
			}
			t.Col = 0
			t.Line++
		case '\n':
			t.Col = 0
			t.Line++
		case scanner.EOF:
			return "", fmt.Errorf("unclosed multiline comment: %s at %+v", string(str), t.Pos())
		default:
			t.Col++
		}

		if mayBeClosingComment {
			if n == '/' {
				break
			} else {
				str = append(str, n)
			}
		}
		mayBeClosingComment = n == '*'
		if !mayBeClosingComment {
			str = append(str, n)
		}
	}

	return string(str), nil
}
