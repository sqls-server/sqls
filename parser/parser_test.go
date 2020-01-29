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

func TestParseParenthesis_NotFoundClose(t *testing.T) {
	input := `select (select (x3) x2 and (y2) bar`
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
}

func TestParseWhere(t *testing.T) {
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
	testStatement(t, stmts[0], 16, input)

	list := stmts[0].GetTokens()
	testItem(t, list[0], "select")
	testItem(t, list[1], " ")
	testItem(t, list[2], "*")
	testItem(t, list[3], " ")
	testItem(t, list[4], "from")
	testItem(t, list[5], " ")
	testIdentifier(t, list[6], "foo")
	testItem(t, list[7], " ")
	testWhere(t, list[8], "where bar = 1 ")
	testItem(t, list[9], "order")
	testItem(t, list[10], " ")
	testItem(t, list[11], "by")
	testItem(t, list[12], " ")
	testIdentifier(t, list[13], "id")
	testItem(t, list[14], " ")
	testItem(t, list[15], "desc")

	where := testTokenList(t, list[8], 8).GetTokens()
	testItem(t, where[0], "where")
	testItem(t, where[1], " ")
	testIdentifier(t, where[2], "bar")
	testItem(t, where[3], " ")
	testItem(t, where[4], "=")
	testItem(t, where[5], " ")
	testItem(t, where[6], "1")
	testItem(t, where[7], " ")
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
	testStatement(t, stmts[0], 9, input)

	list := stmts[0].GetTokens()
	testItem(t, list[0], "select")
	testItem(t, list[1], " ")
	testItem(t, list[2], "*")
	testItem(t, list[3], " ")
	testItem(t, list[4], "from")
	testItem(t, list[5], " ")
	testIdentifier(t, list[6], "foo")
	testItem(t, list[7], " ")
	testWhere(t, list[8], "where bar = 1")

	where := testTokenList(t, list[8], 7).GetTokens()
	testItem(t, where[0], "where")
	testItem(t, where[1], " ")
	testIdentifier(t, where[2], "bar")
	testItem(t, where[3], " ")
	testItem(t, where[4], "=")
	testItem(t, where[5], " ")
	testItem(t, where[6], "1")
}

func TestParseWhere_WithParenthesis(t *testing.T) {
	input := "select x from (select y from foo where bar = 1) z"
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
	testIdentifier(t, list[2], "x")
	testItem(t, list[3], " ")
	testItem(t, list[4], "from")
	testItem(t, list[5], " ")
	testParenthesis(t, list[6], "(select y from foo where bar = 1)")
	testItem(t, list[7], " ")
	testIdentifier(t, list[8], "z")

	parenthesis := testTokenList(t, list[6], 11).GetTokens()
	testItem(t, parenthesis[0], "(")
	testItem(t, parenthesis[1], "select")
	testItem(t, parenthesis[2], " ")
	testIdentifier(t, parenthesis[3], "y")
	testItem(t, parenthesis[4], " ")
	testItem(t, parenthesis[5], "from")
	testItem(t, parenthesis[6], " ")
	testIdentifier(t, parenthesis[7], "foo")
	testItem(t, parenthesis[8], " ")
	testWhere(t, parenthesis[9], "where bar = 1")
	testItem(t, parenthesis[10], ")")
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

	testStatement(t, stmts[0], 4, input)

	list := stmts[0].GetTokens()
	testMemberIdentifier(t, list[0], "a.*")
	testItem(t, list[1], ",")
	testItem(t, list[2], " ")
	testMemberIdentifier(t, list[3], "b.id")
}

func TestParsePeriod_WithWildcard(t *testing.T) {
	input := `a.*`
	stmts := parseInit(t, input)

	testStatement(t, stmts[0], 1, input)

	list := stmts[0].GetTokens()
	testMemberIdentifier(t, list[0], "a.*")
}

func TestParsePeriod_Invalid(t *testing.T) {
	input := `a.`
	stmts := parseInit(t, input)

	testStatement(t, stmts[0], 1, input)

	list := stmts[0].GetTokens()
	testMemberIdentifier(t, list[0], "a.")
}

func TestParsePeriod_InvalidWithSelect(t *testing.T) {
	input := `SELECT foo. FROM foo`
	stmts := parseInit(t, input)

	testStatement(t, stmts[0], 7, input)

	list := stmts[0].GetTokens()
	testItem(t, list[0], "SELECT")
	testItem(t, list[1], " ")
	testMemberIdentifier(t, list[2], "foo.")
	testItem(t, list[3], " ")
	testItem(t, list[4], "FROM")
	testItem(t, list[5], " ")
	testIdentifier(t, list[6], "foo")
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
	testStatement(t, stmts[0], 7, input)

	list := stmts[0].GetTokens()
	testItem(t, list[0], "select")
	testItem(t, list[1], " ")
	testMemberIdentifier(t, list[2], "foo.bar")
	testItem(t, list[3], " ")
	testItem(t, list[4], "from")
	testItem(t, list[5], " ")
	testMemberIdentifier(t, list[6], `"myschema"."table"`)
}

func TestParseAliased(t *testing.T) {
	input := `select foo as bar from mytable`
	stmts := parseInit(t, input)
	testStatement(t, stmts[0], 7, input)

	list := stmts[0].GetTokens()
	testItem(t, list[0], "select")
	testItem(t, list[1], " ")
	testAliased(t, list[2], "foo as bar")
	testItem(t, list[3], " ")
	testItem(t, list[4], "from")
	testItem(t, list[5], " ")
	testIdentifier(t, list[6], `mytable`)
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
		t.Fatalf("Statements does not contain %d statements, got %d", length, len(stmt.GetTokens()))
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

func testMemberIdentifier(t *testing.T, node ast.Node, expect string) {
	t.Helper()
	_, ok := node.(*ast.MemberIdentifer)
	if !ok {
		t.Errorf("invalid type want MemberIdentifer got %T", node)
	}
	if expect != node.String() {
		t.Errorf("expected %q, got %q", expect, node.String())
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

func testFunction(t *testing.T, node ast.Node, expect string) {
	t.Helper()
	_, ok := node.(*ast.Function)
	if !ok {
		t.Errorf("invalid type want Function got %T", node)
	}
	if expect != node.String() {
		t.Errorf("expected %q, got %q", expect, node.String())
	}
}

func testWhere(t *testing.T, node ast.Node, expect string) {
	t.Helper()
	_, ok := node.(*ast.Where)
	if !ok {
		t.Errorf("invalid type want Where got %T", node)
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
