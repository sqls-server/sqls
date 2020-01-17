package parser

import (
	"bytes"
	"testing"

	"github.com/lighttiger2505/sqls/ast"
	"github.com/lighttiger2505/sqls/dialect"
)

func TestParseStatement(t *testing.T) {
	input := `select 1;select 2;select`
	src := bytes.NewBuffer([]byte(input))
	parser, err := NewParser(src, &dialect.GenericSQLDialect{})
	if err != nil {
		t.Fatalf("error %+v\n", err)
	}

	got, err := parser.Parse()
	if err != nil {
		t.Fatalf("error %+v\n", err)
	}
	wantStmtLen := 3
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
	testStatement(t, stmts[0], 4, "select 1;")
	testStatement(t, stmts[1], 4, "select 2;")
	testStatement(t, stmts[2], 1, "select")
}

func TestParseParenthesis(t *testing.T) {
	// TODO Add case of not found close parenthesis
	input := `select (select (x3) x2) and (y2) bar`
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
}

func TestParseWhere(t *testing.T) {
	// TODO Add case of not found close keyword
	input := "select * from foo where bar = 1 order by id desc"
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
	testStatement(t, stmts[0], 15, input)
}

func TestParseIdentifier(t *testing.T) {
	input := `select foo.bar from "myschema"."table"`
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
	testStatement(t, stmts[0], 11, input)

	list := stmts[0].GetTokens()
	testItem(t, list[0], "select")
	testItem(t, list[1], " ")
	testIdentifier(t, list[2], "foo")
	testItem(t, list[3], ".")
	testIdentifier(t, list[4], "bar")
	testItem(t, list[5], " ")
	testItem(t, list[6], "from")
	testItem(t, list[7], " ")
	testIdentifier(t, list[8], `"myschema"`)
	testItem(t, list[9], ".")
	testIdentifier(t, list[10], `"table"`)
}

func testTokenList(t *testing.T, node ast.Node, length int) ast.TokenList {
	t.Helper()
	list, ok := node.(ast.TokenList)
	if !ok {
		t.Fatalf("invalid type want GetTokens got %T", node)
	}
	if length != len(list.GetTokens()) {
		t.Errorf("Statements does not contain %d statements, got %d", length, len(list.GetTokens()))
	}
	return list
}

func testStatement(t *testing.T, stmt *ast.Statement, length int, expect string) {
	t.Helper()
	if length != len(stmt.GetTokens()) {
		t.Errorf("Statements does not contain 3 tokens, want %d got %d", length, len(stmt.GetTokens()))
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
	if expect != item.String() {
		t.Errorf("expected %q, got %q", expect, item.String())
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
