package formatter

import (
	"testing"

	"github.com/lighttiger2505/sqls/ast"
	"github.com/lighttiger2505/sqls/parser"
)

func TestRenderIdentifier(t *testing.T) {
	testcases := []struct {
		name     string
		input    string
		opts     *ast.RenderOptions
		expected []string
	}{
		{
			name:  "snake case",
			input: "SELECT * FROM snake_case_table_name",
			opts: &ast.RenderOptions{
				LowerCase:       false,
				IdentiferQuated: false,
			},
			expected: []string{
				"*",
				"snake_case_table_name",
			},
		},
		{
			name:  "pascal case",
			input: "SELECT p.PascalCaseColumnName FROM \"PascalCaseTableName\" p",
			opts: &ast.RenderOptions{
				LowerCase:       false,
				IdentiferQuated: false,
			},
			expected: []string{
				"p.PascalCaseColumnName",
				"\"PascalCaseTableName\"",
			},
		},
		{
			name:  "quoted pascal case",
			input: "SELECT p.\"PascalCaseColumnName\" FROM \"PascalCaseTableName\" p",
			opts: &ast.RenderOptions{
				LowerCase:       false,
				IdentiferQuated: false,
			},
			expected: []string{
				"p.\"PascalCaseColumnName\"",
				"\"PascalCaseTableName\"",
			},
		},
	}

	for _, tt := range testcases {
		t.Run(tt.name, func(t *testing.T) {
			stmts := parseInit(t, tt.input)
			list := stmts[0].GetTokens()
			j := 0
			for _, n := range list {
				if i, ok := n.(*ast.Identifer); ok {
					if actual := i.Render(tt.opts); actual != tt.expected[j] {
						t.Errorf("expected: %s, got %s", tt.expected[j], actual)
					}
					j++
				}
			}
		})
	}
}

func parseInit(t *testing.T, input string) []*ast.Statement {
	t.Helper()
	parsed, err := parser.Parse(input)
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
