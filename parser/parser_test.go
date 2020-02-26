package parser

import (
	"bytes"
	"reflect"
	"testing"

	"github.com/lighttiger2505/sqls/ast"
	"github.com/lighttiger2505/sqls/dialect"
	"github.com/lighttiger2505/sqls/token"
)

func TestParseStatement(t *testing.T) {
	var input string
	var stmts []*ast.Statement

	input = "select 1;select 2;select 3;"
	stmts = parseInit(t, input)
	testStatement(t, stmts[0], 4, "select 1;")
	testPos(t, stmts[0], genPosOneline(1), genPosOneline(10))
	testStatement(t, stmts[1], 4, "select 2;")
	testPos(t, stmts[1], genPosOneline(10), genPosOneline(19))
	testStatement(t, stmts[2], 4, "select 3;")
	testPos(t, stmts[2], genPosOneline(19), genPosOneline(28))

	input = "select 1;select 2;select 3"
	stmts = parseInit(t, input)
	testStatement(t, stmts[0], 4, "select 1;")
	testPos(t, stmts[0], genPosOneline(1), genPosOneline(10))
	testStatement(t, stmts[1], 4, "select 2;")
	testPos(t, stmts[1], genPosOneline(10), genPosOneline(19))
	testStatement(t, stmts[2], 3, "select 3")
	testPos(t, stmts[2], genPosOneline(19), genPosOneline(27))
}

func TestParseParenthesis(t *testing.T) {
	testcases := []struct {
		name    string
		input   string
		checkFn func(t *testing.T, stmts []*ast.Statement, input string)
	}{
		{
			name:  "single",
			input: "(3)",
			checkFn: func(t *testing.T, stmts []*ast.Statement, input string) {
				testStatement(t, stmts[0], 1, input)
				list := stmts[0].GetTokens()
				testParenthesis(t, list[0], input)
				testPos(t, stmts[0], genPosOneline(1), genPosOneline(4))
			},
		},
		{
			name:  "with operator",
			input: "(3 - 4)",
			checkFn: func(t *testing.T, stmts []*ast.Statement, input string) {
				testStatement(t, stmts[0], 1, input)
				list := stmts[0].GetTokens()
				testParenthesis(t, list[0], input)
				testPos(t, stmts[0], genPosOneline(1), genPosOneline(8))
			},
		},
		{
			name:  "inner parenthesis",
			input: "(1 * 2 + (3 - 4))",
			checkFn: func(t *testing.T, stmts []*ast.Statement, input string) {
				testStatement(t, stmts[0], 1, input)
				list := stmts[0].GetTokens()
				testParenthesis(t, list[0], input)
				testPos(t, stmts[0], genPosOneline(1), genPosOneline(18))
			},
		},
		{
			name:  "with select",
			input: "select (select (x3) x2) and (y2) bar",
			checkFn: func(t *testing.T, stmts []*ast.Statement, input string) {
				testStatement(t, stmts[0], 9, input)

				list := stmts[0].GetTokens()
				testItem(t, list[0], "select")
				testItem(t, list[1], " ")
				testParenthesis(t, list[2], "(select (x3) x2)")
				testItem(t, list[3], " ")
				testItem(t, list[4], "and")
				testItem(t, list[5], " ")
				testParenthesis(t, list[6], "(y2)")
				testItem(t, list[7], " ")
				testIdentifier(t, list[8], `bar`)

				parenthesis := testTokenList(t, list[2], 7).GetTokens()
				testItem(t, parenthesis[0], "(")
				testItem(t, parenthesis[1], "select")
				testItem(t, parenthesis[2], " ")
				testParenthesis(t, parenthesis[3], "(x3)")
				testItem(t, parenthesis[4], " ")
				testIdentifier(t, parenthesis[5], "x2")
				testItem(t, parenthesis[6], ")")
			},
		},
		{
			name:  "not close parenthesis",
			input: "select (select (x3) x2 and (y2) bar",
			checkFn: func(t *testing.T, stmts []*ast.Statement, input string) {
				testStatement(t, stmts[0], 14, input)

				list := stmts[0].GetTokens()
				testItem(t, list[0], "select")
				testItem(t, list[1], " ")
				testItem(t, list[2], "(")
				testItem(t, list[3], "select")
				testItem(t, list[4], " ")
				testParenthesis(t, list[5], "(x3)")
				testItem(t, list[6], " ")
				testIdentifier(t, list[7], "x2")
				testItem(t, list[8], " ")
				testItem(t, list[9], "and")
				testItem(t, list[10], " ")
				testParenthesis(t, list[11], "(y2)")
				testItem(t, list[12], " ")
				testIdentifier(t, list[13], "bar")
			},
		},
	}

	for _, tt := range testcases {
		t.Run(tt.name, func(t *testing.T) {
			stmts := parseInit(t, tt.input)
			tt.checkFn(t, stmts, tt.input)
		})
	}
}

func TestParseWhere(t *testing.T) {
	testcases := []struct {
		name    string
		input   string
		checkFn func(t *testing.T, stmts []*ast.Statement, input string)
	}{
		{
			name:  "select with where",
			input: "select * from foo where bar = 1",
			checkFn: func(t *testing.T, stmts []*ast.Statement, input string) {
				testStatement(t, stmts[0], 6, input)

				list := stmts[0].GetTokens()
				testItem(t, list[0], "select")
				testItem(t, list[1], " ")
				testItem(t, list[2], "*")
				testItem(t, list[3], " ")
				testFrom(t, list[4], "from foo ")
				testWhere(t, list[5], "where bar = 1")

				where := testTokenList(t, list[5], 3).GetTokens()
				testItem(t, where[0], "where")
				testItem(t, where[1], " ")
				testComparison(t, where[2], "bar = 1")
			},
		},
	}

	for _, tt := range testcases {
		t.Run(tt.name, func(t *testing.T) {
			stmts := parseInit(t, tt.input)
			tt.checkFn(t, stmts, tt.input)
		})
	}
}

func TestParseFrom(t *testing.T) {
	testcases := []struct {
		name    string
		input   string
		checkFn func(t *testing.T, stmts []*ast.Statement, input string)
	}{
		{
			name:  "from",
			input: "from ",
			checkFn: func(t *testing.T, stmts []*ast.Statement, input string) {
				testStatement(t, stmts[0], 1, input)
				list := stmts[0].GetTokens()
				testFrom(t, list[0], "from ")
			},
		},
		{
			name:  "from with identifier",
			input: "from abc",
			checkFn: func(t *testing.T, stmts []*ast.Statement, input string) {
				testStatement(t, stmts[0], 1, input)
				list := stmts[0].GetTokens()
				testFrom(t, list[0], "from abc")
			},
		},
		{
			name:  "simple select",
			input: "select * from abc",
			checkFn: func(t *testing.T, stmts []*ast.Statement, input string) {
				testStatement(t, stmts[0], 5, input)
				list := stmts[0].GetTokens()
				testItem(t, list[0], "select")
				testPos(t, list[0], genPosOneline(1), genPosOneline(7))
				testItem(t, list[1], " ")
				testPos(t, list[1], genPosOneline(7), genPosOneline(8))
				testItem(t, list[2], "*")
				testPos(t, list[2], genPosOneline(8), genPosOneline(9))
				testItem(t, list[3], " ")
				testPos(t, list[3], genPosOneline(9), genPosOneline(10))
				testFrom(t, list[4], "from abc")
				testPos(t, list[4], genPosOneline(10), genPosOneline(18))
			},
		},
		{
			name:  "invalid select none select identifier",
			input: "select from abc",
			checkFn: func(t *testing.T, stmts []*ast.Statement, input string) {
				testStatement(t, stmts[0], 3, input)
				list := stmts[0].GetTokens()
				testItem(t, list[0], "select")
				testItem(t, list[1], " ")
				testFrom(t, list[2], "from abc")
			},
		},
		{
			name:  "invalid select none from identifier",
			input: "select * from ",
			checkFn: func(t *testing.T, stmts []*ast.Statement, input string) {
				testStatement(t, stmts[0], 5, input)
				list := stmts[0].GetTokens()
				testItem(t, list[0], "select")
				testItem(t, list[1], " ")
				testItem(t, list[2], "*")
				testItem(t, list[3], " ")
				testFrom(t, list[4], "from ")
				testPos(t, list[4], genPosOneline(10), genPosOneline(15))

				where := testTokenList(t, list[4], 2).GetTokens()
				testItem(t, where[0], "from")
				testPos(t, where[0], genPosOneline(10), genPosOneline(14))
				testItem(t, where[1], " ")
				testPos(t, where[1], genPosOneline(14), genPosOneline(15))
			},
		},
		{
			name:  "sub query",
			input: "select * from (select * from abc) as t",
			checkFn: func(t *testing.T, stmts []*ast.Statement, input string) {
				testStatement(t, stmts[0], 5, input)
				list := stmts[0].GetTokens()
				testItem(t, list[0], "select")
				testItem(t, list[1], " ")
				testItem(t, list[2], "*")
				testItem(t, list[3], " ")
				testFrom(t, list[4], "from (select * from abc) as t")

				from := testTokenList(t, list[4], 3).GetTokens()
				testItem(t, from[0], "from")
				testItem(t, from[1], " ")
				testAliased(t, from[2], "(select * from abc) as t")

				aliased := testTokenList(t, from[2], 5).GetTokens()
				testParenthesis(t, aliased[0], "(select * from abc)")
				testItem(t, aliased[1], " ")
				testItem(t, aliased[2], "as")
				testItem(t, aliased[3], " ")
				testIdentifier(t, aliased[4], "t")

				fromInParenthesis := testTokenList(t, aliased[0], 7).GetTokens()
				testItem(t, fromInParenthesis[0], "(")
				testItem(t, fromInParenthesis[1], "select")
				testItem(t, fromInParenthesis[2], " ")
				testItem(t, fromInParenthesis[3], "*")
				testItem(t, fromInParenthesis[4], " ")
				testFrom(t, fromInParenthesis[5], "from abc")
				testItem(t, fromInParenthesis[6], ")")
			},
		},
	}

	for _, tt := range testcases {
		t.Run(tt.name, func(t *testing.T) {
			stmts := parseInit(t, tt.input)
			tt.checkFn(t, stmts, tt.input)
		})
	}
}

func TestParseJoin(t *testing.T) {
	input := "select * from abc join efd"

	stmts := parseInit(t, input)
	testStatement(t, stmts[0], 6, input)

	list := stmts[0].GetTokens()
	testItem(t, list[0], "select")
	testItem(t, list[1], " ")
	testItem(t, list[2], "*")
	testItem(t, list[3], " ")
	testFrom(t, list[4], "from abc ")
	testJoin(t, list[5], "join efd")
}

func TestParseJoin_WithOn(t *testing.T) {
	input := "select * from abc join efd on abc.id = efd.id"

	stmts := parseInit(t, input)
	testStatement(t, stmts[0], 9, input)

	list := stmts[0].GetTokens()
	testItem(t, list[0], "select")
	testItem(t, list[1], " ")
	testItem(t, list[2], "*")
	testItem(t, list[3], " ")
	testFrom(t, list[4], "from abc ")
	testJoin(t, list[5], "join efd ")
	testItem(t, list[6], "on")
	testItem(t, list[7], " ")
	testComparison(t, list[8], "abc.id = efd.id")
}

func TestParseWhere_NotFoundClose(t *testing.T) {
	input := "select * from foo where bar = 1"
	src := bytes.NewBuffer([]byte(input))
	parser, err := NewParser(src, &dialect.GenericSQLDialect{})
	if err != nil {
		t.Fatalf("error %+v\n", err)
	}

	got, err := parser.Parse()
	if err != nil {
		t.Fatalf("error %+v\n", err)
	}
	wantStmtLen := 1
	if wantStmtLen != len(got.GetTokens()) {
		t.Errorf("Statements does not contain %d statements, got %d", wantStmtLen, len(got.GetTokens()))
	}
	var stmts []*ast.Statement
	for _, node := range got.GetTokens() {
		stmt, ok := node.(*ast.Statement)
		if !ok {
			t.Fatalf("invalid type want Statement got %T", stmt)
		}
		stmts = append(stmts, stmt)
	}
	testStatement(t, stmts[0], 6, input)

	list := stmts[0].GetTokens()
	testItem(t, list[0], "select")
	testItem(t, list[1], " ")
	testItem(t, list[2], "*")
	testItem(t, list[3], " ")
	testFrom(t, list[4], "from foo ")
	testWhere(t, list[5], "where bar = 1")

	where := testTokenList(t, list[5], 3).GetTokens()
	testItem(t, where[0], "where")
	testItem(t, where[1], " ")
	testComparison(t, where[2], "bar = 1")
}

func TestParseWhere_WithParenthesis(t *testing.T) {
	input := "select x from (select y from foo where bar = 1) z"
	stmts := parseInit(t, input)
	testStatement(t, stmts[0], 5, input)

	list := stmts[0].GetTokens()
	testItem(t, list[0], "select")
	testItem(t, list[1], " ")
	testIdentifier(t, list[2], "x")
	testItem(t, list[3], " ")
	testFrom(t, list[4], "from (select y from foo where bar = 1) z")

	from := testTokenList(t, list[4], 5).GetTokens()
	parenthesis := testTokenList(t, from[2], 8).GetTokens()
	testItem(t, parenthesis[0], "(")
	testItem(t, parenthesis[1], "select")
	testItem(t, parenthesis[2], " ")
	testIdentifier(t, parenthesis[3], "y")
	testItem(t, parenthesis[4], " ")
	testFrom(t, parenthesis[5], "from foo ")
	testWhere(t, parenthesis[6], "where bar = 1")
	testItem(t, parenthesis[7], ")")
}

func TestParseFunction(t *testing.T) {
	input := `foo()`
	stmts := parseInit(t, input)
	testStatement(t, stmts[0], 1, input)

	list := stmts[0].GetTokens()
	testFunction(t, list[0], "foo()")
}

func TestParsePeriod_Double(t *testing.T) {
	input := `a.*, b.id`
	stmts := parseInit(t, input)

	testStatement(t, stmts[0], 1, input)

	list := stmts[0].GetTokens()
	testIdentifierList(t, list[0], input)

	il := testTokenList(t, list[0], 4).GetTokens()
	testMemberIdentifier(t, il[0], "a.*", "a", "*")
	testItem(t, il[1], ",")
	testItem(t, il[2], " ")
	testMemberIdentifier(t, il[3], "b.id", "b", "id")
}

func TestParsePeriod_WithWildcard(t *testing.T) {
	input := `a.*`
	stmts := parseInit(t, input)

	testStatement(t, stmts[0], 1, input)

	list := stmts[0].GetTokens()
	testMemberIdentifier(t, list[0], "a.*", "a", "*")
}

func TestParsePeriod_Invalid(t *testing.T) {
	input := `a.`
	stmts := parseInit(t, input)

	testStatement(t, stmts[0], 1, input)

	list := stmts[0].GetTokens()
	testMemberIdentifier(t, list[0], "a.", "a", "")
}

func TestParsePeriod_InvalidWithSelect(t *testing.T) {
	input := `SELECT foo. FROM foo`
	stmts := parseInit(t, input)

	testStatement(t, stmts[0], 5, input)

	list := stmts[0].GetTokens()
	testItem(t, list[0], "SELECT")
	testItem(t, list[1], " ")
	testMemberIdentifier(t, list[2], "foo.", "foo", "")
	testItem(t, list[3], " ")
	testFrom(t, list[4], "FROM foo")
}

func TestParseIdentifier(t *testing.T) {
	testcases := []struct {
		name    string
		input   string
		checkFn func(t *testing.T, stmts []*ast.Statement, input string)
	}{
		{
			name:  "identifier",
			input: "abc",
			checkFn: func(t *testing.T, stmts []*ast.Statement, input string) {
				testStatement(t, stmts[0], 1, input)
				list := stmts[0].GetTokens()
				testIdentifier(t, list[0], "abc")
			},
		},
		{
			name:  "double quate identifier",
			input: `"abc"`,
			checkFn: func(t *testing.T, stmts []*ast.Statement, input string) {
				testStatement(t, stmts[0], 1, input)
				list := stmts[0].GetTokens()
				testIdentifier(t, list[0], `"abc"`)
			},
		},
		{
			name:  "select identifier",
			input: "select abc",
			checkFn: func(t *testing.T, stmts []*ast.Statement, input string) {
				testStatement(t, stmts[0], 3, input)
				list := stmts[0].GetTokens()
				testItem(t, list[0], "select")
				testItem(t, list[1], " ")
				testIdentifier(t, list[2], "abc")
			},
		},
		{
			name:  "from identifier",
			input: "select abc from def",
			checkFn: func(t *testing.T, stmts []*ast.Statement, input string) {
				testStatement(t, stmts[0], 5, input)
				list := stmts[0].GetTokens()
				testItem(t, list[0], "select")
				testItem(t, list[1], " ")
				testIdentifier(t, list[2], "abc")
				testItem(t, list[3], " ")
				testFrom(t, list[4], "from def")
				from := testTokenList(t, list[4], 3).GetTokens()
				testItem(t, from[0], "from")
				testItem(t, from[1], " ")
				testIdentifier(t, from[2], "def")
			},
		},
	}
	for _, tt := range testcases {
		t.Run(tt.name, func(t *testing.T) {
			stmts := parseInit(t, tt.input)
			tt.checkFn(t, stmts, tt.input)
		})
	}
}

func TestMemberIdentifier(t *testing.T) {
	testcases := []struct {
		name    string
		input   string
		checkFn func(t *testing.T, stmts []*ast.Statement, input string)
	}{
		{
			name:  "member identifier",
			input: "a.b",
			checkFn: func(t *testing.T, stmts []*ast.Statement, input string) {
				testStatement(t, stmts[0], 1, input)
				list := stmts[0].GetTokens()
				testMemberIdentifier(t, list[0], input, "a", "b")
			},
		},
		{
			name:  "double quate member identifier",
			input: `"abc"."def"`,
			checkFn: func(t *testing.T, stmts []*ast.Statement, input string) {
				testStatement(t, stmts[0], 1, input)
				list := stmts[0].GetTokens()
				testMemberIdentifier(t, list[0], input, `"abc"`, `"def"`)
			},
		},
		{
			name:  "invalid member identifier",
			input: "a.",
			checkFn: func(t *testing.T, stmts []*ast.Statement, input string) {
				testStatement(t, stmts[0], 1, input)
				list := stmts[0].GetTokens()
				testMemberIdentifier(t, list[0], input, "a", "")
			},
		},
		{
			name:  "member identifier wildcard",
			input: "a.*",
			checkFn: func(t *testing.T, stmts []*ast.Statement, input string) {
				testStatement(t, stmts[0], 1, input)
				list := stmts[0].GetTokens()
				testMemberIdentifier(t, list[0], input, "a", "*")
			},
		},
		{
			name:  "member identifier select",
			input: "select foo.bar from table",
			checkFn: func(t *testing.T, stmts []*ast.Statement, input string) {
				testStatement(t, stmts[0], 5, input)
				list := stmts[0].GetTokens()
				testItem(t, list[0], "select")
				testItem(t, list[1], " ")
				testMemberIdentifier(t, list[2], "foo.bar", "foo", "bar")
				testItem(t, list[3], " ")
				testFrom(t, list[4], `from table`)
			},
		},
		{
			name:  "invalid member identifier select",
			input: "select foo. from table",
			checkFn: func(t *testing.T, stmts []*ast.Statement, input string) {
				testStatement(t, stmts[0], 5, input)
				list := stmts[0].GetTokens()

				testItem(t, list[0], "select")
				testPos(t, list[0], genPosOneline(1), genPosOneline(7))

				testItem(t, list[1], " ")
				testPos(t, list[1], genPosOneline(7), genPosOneline(8))

				testMemberIdentifier(t, list[2], "foo.", "foo", "")
				testPos(t, list[2], genPosOneline(8), genPosOneline(12))

				testItem(t, list[3], " ")
				testPos(t, list[3], genPosOneline(12), genPosOneline(13))

				testFrom(t, list[4], `from table`)
				testPos(t, list[4], genPosOneline(13), genPosOneline(23))
			},
		},
		{
			name:  "member identifier from",
			input: "select foo from myschema.table",
			checkFn: func(t *testing.T, stmts []*ast.Statement, input string) {
				testStatement(t, stmts[0], 5, input)
				list := stmts[0].GetTokens()
				testItem(t, list[0], "select")
				testItem(t, list[1], " ")
				testIdentifier(t, list[2], "foo")
				testItem(t, list[3], " ")
				testFrom(t, list[4], `from myschema.table`)
			},
		},
	}
	for _, tt := range testcases {
		t.Run(tt.name, func(t *testing.T) {
			stmts := parseInit(t, tt.input)
			tt.checkFn(t, stmts, tt.input)
		})
	}
}

func TestParseMultiKeyword(t *testing.T) {
	testcases := []struct {
		name    string
		input   string
		checkFn func(t *testing.T, stmts []*ast.Statement, input string)
	}{
		{
			name:  "order keyword",
			input: "order by",
			checkFn: func(t *testing.T, stmts []*ast.Statement, input string) {
				testStatement(t, stmts[0], 1, input)
				list := stmts[0].GetTokens()
				testMultiKeyword(t, list[0], input)
				testPos(t, stmts[0], genPosOneline(1), genPosOneline(9))
			},
		},
		{
			name:  "group keyword",
			input: "group by",
			checkFn: func(t *testing.T, stmts []*ast.Statement, input string) {
				testStatement(t, stmts[0], 1, input)
				list := stmts[0].GetTokens()
				testMultiKeyword(t, list[0], input)
				testPos(t, stmts[0], genPosOneline(1), genPosOneline(9))
			},
		},
		{
			name:  "select with group keyword",
			input: "select a, b, c from abc group by d, e, f",
			checkFn: func(t *testing.T, stmts []*ast.Statement, input string) {
				testStatement(t, stmts[0], 8, input)
				list := stmts[0].GetTokens()
				testItem(t, list[0], "select")
				testItem(t, list[1], " ")
				testIdentifierList(t, list[2], "a, b, c")
				testItem(t, list[3], " ")
				testFrom(t, list[4], "from abc ")
				testMultiKeyword(t, list[5], "group by")
				testItem(t, list[6], " ")
				testIdentifierList(t, list[7], "d, e, f")
			},
		},
	}
	for _, tt := range testcases {
		t.Run(tt.name, func(t *testing.T) {
			stmts := parseInit(t, tt.input)
			tt.checkFn(t, stmts, tt.input)
		})
	}
}

func TestParseOperator(t *testing.T) {
	var input string
	var stmts []*ast.Statement
	var list []ast.Node

	input = "foo+100"
	stmts = parseInit(t, input)
	testStatement(t, stmts[0], 1, input)
	list = stmts[0].GetTokens()
	testOperator(t, list[0], input)

	input = "foo + 100"
	stmts = parseInit(t, input)
	testStatement(t, stmts[0], 1, input)
	list = stmts[0].GetTokens()
	testOperator(t, list[0], input)

	input = "foo*100"
	stmts = parseInit(t, input)
	testStatement(t, stmts[0], 1, input)
	list = stmts[0].GetTokens()
	testOperator(t, list[0], input)
}

func TestParseComparison(t *testing.T) {
	var input string
	var stmts []*ast.Statement
	var list []ast.Node

	input = "foo = 25.5"
	stmts = parseInit(t, input)
	testStatement(t, stmts[0], 1, input)
	list = stmts[0].GetTokens()
	testComparison(t, list[0], input)

	input = "foo = 'bar'"
	stmts = parseInit(t, input)
	testStatement(t, stmts[0], 1, input)
	list = stmts[0].GetTokens()
	testComparison(t, list[0], input)

	input = "(3 + 4) = 7"
	stmts = parseInit(t, input)
	testStatement(t, stmts[0], 1, input)
	list = stmts[0].GetTokens()
	testComparison(t, list[0], input)

	input = "foo = DATE(bar.baz)"
	stmts = parseInit(t, input)
	testStatement(t, stmts[0], 1, input)
	list = stmts[0].GetTokens()
	testComparison(t, list[0], input)

	input = "foo = DATE(bar.baz)"
	stmts = parseInit(t, input)
	testStatement(t, stmts[0], 1, input)
	list = stmts[0].GetTokens()
	testComparison(t, list[0], input)

	input = "DATE(foo.bar) = bar.baz"
	stmts = parseInit(t, input)
	testStatement(t, stmts[0], 1, input)
	list = stmts[0].GetTokens()
	testComparison(t, list[0], input)
}

func TestParseAliased(t *testing.T) {
	input := `select foo as bar from mytable`
	stmts := parseInit(t, input)
	testStatement(t, stmts[0], 5, input)

	list := stmts[0].GetTokens()
	testItem(t, list[0], "select")
	testItem(t, list[1], " ")
	testAliased(t, list[2], "foo as bar")
	testItem(t, list[3], " ")
	testFrom(t, list[4], "from mytable")
}

func TestParseIdentifierList(t *testing.T) {
	testcases := []struct {
		name    string
		input   string
		checkFn func(t *testing.T, stmts []*ast.Statement, input string)
	}{
		{
			name:  "IndentifierList",
			input: "foo, bar",
			checkFn: func(t *testing.T, stmts []*ast.Statement, input string) {
				testStatement(t, stmts[0], 1, input)
				list := stmts[0].GetTokens()
				testIdentifierList(t, list[0], input)
			},
		},
		{
			name:  "invalid IndentifierList",
			input: "foo, bar,",
			checkFn: func(t *testing.T, stmts []*ast.Statement, input string) {
				testStatement(t, stmts[0], 1, input)
				list := stmts[0].GetTokens()
				testIdentifierList(t, list[0], input)
			},
		},
		{
			name:  "invalid single IndentifierList in select statement",
			input: "select foo,  from table",
			checkFn: func(t *testing.T, stmts []*ast.Statement, input string) {
				testStatement(t, stmts[0], 4, input)
				list := stmts[0].GetTokens()
				testItem(t, list[0], "select")
				testItem(t, list[1], " ")
				testIdentifierList(t, list[2], "foo,  ")
				testFrom(t, list[3], "from table")
			},
		},
		{
			name:  "invalid multiplue IndentifierList in select statement",
			input: "select foo, bar, from table",
			checkFn: func(t *testing.T, stmts []*ast.Statement, input string) {
				testStatement(t, stmts[0], 4, input)
				list := stmts[0].GetTokens()
				testItem(t, list[0], "select")
				testItem(t, list[1], " ")
				testIdentifierList(t, list[2], "foo, bar, ")
				testFrom(t, list[3], "from table")
			},
		},
		{
			name:  "IndentifierList function",
			input: "sum(a), sum(b)",
			checkFn: func(t *testing.T, stmts []*ast.Statement, input string) {
				testStatement(t, stmts[0], 1, input)
				list := stmts[0].GetTokens()
				testIdentifierList(t, list[0], input)
			},
		},
		{
			name:  "IndentifierList Aliased",
			input: "sum(a) as x, b as y",
			checkFn: func(t *testing.T, stmts []*ast.Statement, input string) {
				testStatement(t, stmts[0], 1, input)
				list := stmts[0].GetTokens()
				testIdentifierList(t, list[0], input)
			},
		},
	}
	for _, tt := range testcases {
		t.Run(tt.name, func(t *testing.T) {
			stmts := parseInit(t, tt.input)
			tt.checkFn(t, stmts, tt.input)
		})
	}
}

func parseInit(t *testing.T, input string) []*ast.Statement {
	t.Helper()
	src := bytes.NewBuffer([]byte(input))
	parser, err := NewParser(src, &dialect.GenericSQLDialect{})
	if err != nil {
		t.Fatalf("error %+v\n", err)
	}

	parsed, err := parser.Parse()
	if err != nil {
		t.Fatalf("error %+v\n", err)
	}

	var stmts []*ast.Statement
	for _, node := range parsed.GetTokens() {
		stmt, ok := node.(*ast.Statement)
		if !ok {
			t.Fatalf("invalid type want Statement parsed %T", stmt)
		}
		stmts = append(stmts, stmt)
	}
	return stmts
}

func testTokenList(t *testing.T, node ast.Node, length int) ast.TokenList {
	t.Helper()
	list, ok := node.(ast.TokenList)
	if !ok {
		t.Fatalf("invalid type want GetTokens got %T", node)
	}
	if length != len(list.GetTokens()) {
		t.Fatalf("Statements does not contain %d statements, got %d", length, len(list.GetTokens()))
	}
	return list
}

func testStatement(t *testing.T, stmt *ast.Statement, length int, expect string) {
	t.Helper()
	if length != len(stmt.GetTokens()) {
		t.Fatalf("Statements does not contain %d statements, got %d, (expect %q got: %q)", length, len(stmt.GetTokens()), expect, stmt.String())
	}
	if expect != stmt.String() {
		t.Errorf("expected %q, got %q", expect, stmt.String())
	}
}

func testItem(t *testing.T, node ast.Node, expect string) {
	t.Helper()
	item, ok := node.(*ast.Item)
	if !ok {
		t.Errorf("invalid type want Item got %T", node)
	}
	if item != nil {
		if expect != item.String() {
			t.Errorf("expected %q, got %q", expect, item.String())
		}
	} else {
		t.Errorf("item is null")
	}
}

func testMemberIdentifier(t *testing.T, node ast.Node, expect, parent, child string) {
	t.Helper()
	mi, ok := node.(*ast.MemberIdentifer)
	if !ok {
		t.Errorf("invalid type want MemberIdentifer got %T", node)
	}
	if expect != node.String() {
		t.Errorf("expected %q, got %q", expect, node.String())
	}
	if parent != "" {
		if mi.Parent != nil {
			if parent != mi.Parent.String() {
				t.Errorf("parent expected %q , got %q", parent, mi.Parent.String())
			}
		} else {
			t.Errorf("parent is nil , got %q", parent)
		}
	}
	if child != "" {
		if mi.Child != nil {
			if child != mi.Child.String() {
				t.Errorf("child expected %q , got %q", child, mi.Parent.String())
			}
		} else {
			t.Errorf("child is nil , got %q", child)
		}
	}
}

func testIdentifier(t *testing.T, node ast.Node, expect string) {
	t.Helper()
	_, ok := node.(*ast.Identifer)
	if !ok {
		t.Errorf("invalid type want Identifier got %T", node)
	}
	if expect != node.String() {
		t.Errorf("expected %q, got %q", expect, node.String())
	}
}

func testMultiKeyword(t *testing.T, node ast.Node, expect string) {
	t.Helper()
	_, ok := node.(*ast.MultiKeyword)
	if !ok {
		t.Errorf("invalid type want MultiKeyword got %T", node)
	}
	if expect != node.String() {
		t.Errorf("expected %q, got %q", expect, node.String())
	}
}

func testOperator(t *testing.T, node ast.Node, expect string) {
	t.Helper()
	_, ok := node.(*ast.Operator)
	if !ok {
		t.Errorf("invalid type want Operator got %T", node)
	}
	if expect != node.String() {
		t.Errorf("expected %q, got %q", expect, node.String())
	}
}

func testComparison(t *testing.T, node ast.Node, expect string) {
	t.Helper()
	_, ok := node.(*ast.Comparison)
	if !ok {
		t.Errorf("invalid type want Comparison got %T", node)
	}
	if expect != node.String() {
		t.Errorf("expected %q, got %q", expect, node.String())
	}
}

func testParenthesis(t *testing.T, node ast.Node, expect string) {
	t.Helper()
	_, ok := node.(*ast.Parenthesis)
	if !ok {
		t.Errorf("invalid type want Parenthesis got %T", node)
	}
	if expect != node.String() {
		t.Errorf("expected %q, got %q", expect, node.String())
	}
}

func testFunction(t *testing.T, node ast.Node, expect string) {
	t.Helper()
	_, ok := node.(*ast.FunctionLiteral)
	if !ok {
		t.Errorf("invalid type want Function got %T", node)
	}
	if expect != node.String() {
		t.Errorf("expected %q, got %q", expect, node.String())
	}
}

func testWhere(t *testing.T, node ast.Node, expect string) {
	t.Helper()
	_, ok := node.(*ast.WhereClause)
	if !ok {
		t.Errorf("invalid type want Where got %T", node)
	}
	if expect != node.String() {
		t.Errorf("expected %q, got %q", expect, node.String())
	}
}

func testFrom(t *testing.T, node ast.Node, expect string) {
	t.Helper()
	_, ok := node.(*ast.FromClause)
	if !ok {
		t.Errorf("invalid type want From got %T", node)
	}
	if expect != node.String() {
		t.Errorf("expected %q, got %q", expect, node.String())
	}
}

func testJoin(t *testing.T, node ast.Node, expect string) {
	t.Helper()
	_, ok := node.(*ast.JoinClause)
	if !ok {
		t.Errorf("invalid type want Join got %T", node)
	}
	if expect != node.String() {
		t.Errorf("expected %q, got %q", expect, node.String())
	}
}

func testAliased(t *testing.T, node ast.Node, expect string) {
	t.Helper()
	_, ok := node.(*ast.Aliased)
	if !ok {
		t.Errorf("invalid type want Identifier got %T", node)
	}
	if expect != node.String() {
		t.Errorf("expected %q, got %q", expect, node.String())
	}
}

func testIdentifierList(t *testing.T, node ast.Node, expect string) {
	t.Helper()
	_, ok := node.(*ast.IdentiferList)
	if !ok {
		t.Errorf("invalid type want IdentiferList got %T", node)
	}
	if expect != node.String() {
		t.Errorf("expected %q, got %q", expect, node.String())
	}
}

func testPos(t *testing.T, node ast.Node, pos, end token.Pos) {
	t.Helper()
	if !reflect.DeepEqual(pos, node.Pos()) {
		t.Errorf("PosExpected %+v, got %+v", pos, node.Pos())
	}
	if !reflect.DeepEqual(end, node.End()) {
		t.Errorf("EndExpected %+v, got %+v", end, node.End())
	}
}

func genPosOneline(col int) token.Pos {
	return token.Pos{Line: 1, Col: col}
}
