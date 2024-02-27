package formatter

import (
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"

	"github.com/sqls-server/sqls/ast"
	"github.com/sqls-server/sqls/internal/config"
	"github.com/sqls-server/sqls/internal/lsp"
	"github.com/sqls-server/sqls/parser"
)

func TestEval(t *testing.T) {
	testcases := []struct {
		name     string
		input    string
		params   lsp.DocumentFormattingParams
		config   *config.Config
		expected string
	}{
		{
			name:     "InsertIntoFormat",
			input:    "INSERT INTO users (NAME, email) VALUES ('john doe', 'example@host.com')",
			expected: "INSERT INTO users(\n\tNAME,\n\temail\n)\nVALUES(\n\t'john doe',\n\t'example@host.com'\n)",
			params:   lsp.DocumentFormattingParams{},
			config: &config.Config{
				LowercaseKeywords: false,
			},
		},
	}

	for _, tt := range testcases {
		t.Run(tt.name, func(t *testing.T) {
			actual, _ := Format(tt.input, tt.params, tt.config)
			if actual[0].NewText != tt.expected {
				t.Errorf("expected: %s, got %s", tt.expected, actual[0].NewText)
			}
		})
	}
}

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
				LowerCase:        false,
				IdentifierQuoted: false,
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
				LowerCase:        false,
				IdentifierQuoted: false,
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
				LowerCase:        false,
				IdentifierQuoted: false,
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
				if i, ok := n.(*ast.Identifier); ok {
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

func TestFormat(t *testing.T) {
	_, filename, _, _ := runtime.Caller(0)
	dir := filepath.Dir(filename)

	files, err := filepath.Glob(filepath.Join(dir, "testdata", "*.sql"))
	if err != nil {
		t.Fatal(err)
	}
	slices.Sort(files)

	opts := &ast.RenderOptions{
		LowerCase:        false,
		IdentifierQuoted: false,
	}
	for _, fname := range files {
		b, err := os.ReadFile(fname)
		if err != nil {
			t.Fatal(err)
		}
		parsed, err := parser.Parse(string(b))
		if err != nil {
			t.Fatal(err)
		}
		env := &formatEnvironment{}
		formatted := Eval(parsed, env)
		got := strings.TrimRight(formatted.Render(opts), "\n") + "\n"

		b, err = os.ReadFile(fname[:len(fname)-4] + ".golden")
		if err != nil {
			t.Fatal(err)
		}
		want := string(b)
		if got != want {
			if _, err := os.Stat(fname[:len(fname)-4] + ".ignore"); err == nil {
				t.Logf("%s:\n"+
					"    want: %q\n"+
					"     got: %q\n",
					fname, want, got)
			} else {
				t.Errorf("%s:\n"+
					"    want: %q\n"+
					"     got: %q\n",
					fname, want, got)
			}
		}
	}
}
