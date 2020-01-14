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
		t.Fatalf("parse error %s", err)
	}

	got, err := parser.Parse()
	if err != nil {
		t.Fatal(err)
	}
	var stmts []*ast.Statement
	for _, node := range got.GetTokens() {
		stmt, ok := node.(*ast.Statement)
		if !ok {
			t.Fatalf("invalid type want Statement got %T", stmt)
		}
		stmts = append(stmts, stmt)
	}
	wantStmtLen := 3
	if wantStmtLen != len(stmts) {
		t.Errorf("Statements does not contain 3 statements, want %d got %d", wantStmtLen, len(stmts))
	}
	testStatement(t, stmts[0], 4, "select 1;")
	testStatement(t, stmts[1], 4, "select 2;")
	testStatement(t, stmts[2], 1, "select")
}

func testStatement(t *testing.T, stmt *ast.Statement, length int, expect string) {
	t.Helper()
	if length != len(stmt.GetTokens()) {
		t.Errorf("Statements does not contain 3 tokens, want %d got %d", length, len(stmt.GetTokens()))
	}
	if expect != stmt.String() {
		t.Errorf("expected=%q, got=%q", expect, stmt.String())
	}
}

// func TestParseIdentifier(t *testing.T) {
// 	input := `select foo.bar from "myscheme"."table"`
// 	src := bytes.NewBuffer([]byte(input))
// 	parser, err := NewParser(src, &dialect.GenericSQLDialect{})
// 	if err != nil {
// 		t.Fatalf("error %s", err)
// 	}
//
// 	parsed, err := parser.Parse()
// 	if err != nil {
// 		t.Fatal(err)
// 	}
// 	tokenList := parsed[0].(ast.TokenList)
// 	got := tokenList.Tokens()
//
// 	// select
// 	testTokenNode(t, got[0])
// 	// " "
// 	testTokenNode(t, got[1])
// 	// foo
// 	testIdentifier(t, got[2])
// 	// "."
// 	testTokenNode(t, got[3])
// 	// bar
// 	testIdentifier(t, got[4])
// 	// " "
// 	testTokenNode(t, got[5])
// 	// from
// 	testTokenNode(t, got[6])
// 	// " "
// 	testTokenNode(t, got[7])
// 	// myschema
// 	testIdentifier(t, got[8])
// 	// "."
// 	testTokenNode(t, got[9])
// 	// "table"
// 	testIdentifier(t, got[10])
// }
//
// func testTokenNode(t *testing.T, node ast.Node) {
// 	t.Helper()
// 	if _, ok := node.(*ast.TokenNode); !ok {
// 		t.Errorf("not TokenNode got %T", node)
// 	}
// }
//
// func testIdentifier(t *testing.T, node ast.Node) {
// 	t.Helper()
// 	if _, ok := node.(*ast.Ident); !ok {
// 		t.Errorf("not Identifier got %T", node)
// 	}
// }
