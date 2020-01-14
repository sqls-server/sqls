package parser

import (
	"bytes"
	"testing"

	"github.com/lighttiger2505/sqls/ast"
	"github.com/lighttiger2505/sqls/dialect"
)

// func TestParseComment(t *testing.T) {
// 	input := `/*\n * foo\n */   \n  bar`
// 	src := bytes.NewBuffer([]byte(input))
// 	parser, err := NewParser(src, &dialect.GenericSQLDialect{})
// 	if err != nil {
// 		t.Fatalf("parse error %s", err)
// 	}
//
// 	got, err := parser.Parse()
// 	if err != nil {
// 		t.Fatal(err)
// 	}
// 	list, ok := got.(*ast.SQLTokenList)
// 	if !ok {
// 		t.Fatalf("invalid type %T", got)
// 	}
//
// 	pp.Println(list)
//
// 	want := 2
// 	if want != len(list.Children) {
// 		t.Errorf("invalid statement num, want %d got %d", want, len(list.Children))
// 	}
// }

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
	list, ok := got.(*ast.SQLTokenList)
	if !ok {
		t.Fatalf("invalid type %T", got)
	}

	var want int
	want = 3
	if want != len(list.Children) {
		t.Errorf("invalid statement num, want %d got %d", want, len(list.Children))
	}

	list1, ok := list.Children[0].(*ast.SQLTokenList)
	if !ok {
		t.Fatalf("invalid type %T", got)
	}
	want = 4
	if want != len(list1.Children) {
		t.Errorf("invalid list1 num, want %d got %d", want, len(list1.Children))
	}

	list2, ok := list.Children[1].(*ast.SQLTokenList)
	if !ok {
		t.Fatalf("invalid type %T", got)
	}
	want = 4
	if want != len(list2.Children) {
		t.Errorf("invalid list2 num, want %d got %d", want, len(list2.Children))
	}

	list3, ok := list.Children[2].(*ast.SQLTokenList)
	if !ok {
		t.Fatalf("invalid type %T", got)
	}
	want = 1
	if want != len(list3.Children) {
		t.Errorf("invalid list3 num, want %d got %d", want, len(list3.Children))
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
