package parser

import (
	"bytes"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/lighttiger2505/sqls/ast"
	"github.com/lighttiger2505/sqls/ast/astutil"
	"github.com/lighttiger2505/sqls/dialect"
	"github.com/lighttiger2505/sqls/token"
)

func TestExtractTable(t *testing.T) {
	testcases := []struct {
		name  string
		input string
		want  []*TableInfo
	}{
		{
			name:  "from only",
			input: "from abc",
			want: []*TableInfo{
				&TableInfo{
					Name: "abc",
				},
			},
		},
		{
			name:  "one table",
			input: "select * from abc",
			want: []*TableInfo{
				&TableInfo{
					Name: "abc",
				},
			},
		},
		{
			name:  "multiple table",
			input: "select * from abc, def",
			want: []*TableInfo{
				&TableInfo{
					Name: "abc",
				},
				&TableInfo{
					Name: "def",
				},
			},
		},
		{
			name:  "with database schema",
			input: "select * from abc.def",
			want: []*TableInfo{
				&TableInfo{
					DatabaseSchema: "abc",
					Name:           "def",
				},
			},
		},
		{
			name:  "with database schema and alias",
			input: "select * from abc.def as ghi",
			want: []*TableInfo{
				&TableInfo{
					DatabaseSchema: "abc",
					Name:           "def",
					Alias:          "ghi",
				},
			},
		},
	}

	for _, tt := range testcases {
		t.Run(tt.name, func(t *testing.T) {
			stmt := initExtractTable(t, tt.input)
			got := ExtractTable(stmt)
			if d := cmp.Diff(tt.want, got); d != "" {
				t.Errorf("unmatched value: %s", d)
			}
		})
	}
}

func initExtractTable(t *testing.T, input string) ast.TokenList {
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
	return parsed
}

func TestNodeWalker_PrevNodesIs(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		pos     token.Pos
		matcher astutil.NodeMatcher
		want    bool
	}{
		{
			name:  "prev select",
			input: "SELECT  FROM def",
			pos:   token.Pos{Line: 1, Col: 7},
			matcher: astutil.NodeMatcher{
				ExpectKeyword: []string{"SELECT"},
			},
			want: true,
		},
		{
			name:  "prev FROM",
			input: "SELECT * FROM ",
			pos:   token.Pos{Line: 1, Col: 14},
			matcher: astutil.NodeMatcher{
				ExpectKeyword: []string{"From"},
			},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// initialize
			src := bytes.NewBuffer([]byte(tt.input))
			parser, err := NewParser(src, &dialect.GenericSQLDialect{})
			if err != nil {
				t.Fatalf("error %+v\n", err)
			}
			parsed, err := parser.Parse()
			if err != nil {
				t.Fatalf("error %+v\n", err)
			}
			nodeWalker := NewNodeWalker(parsed, tt.pos)

			// execute
			if got := nodeWalker.PrevNodesIs(true, tt.matcher); got != tt.want {
				t.Errorf("nodeWalker.PrevNodesIs() = %v, want %v", got, tt.want)
			}
		})
	}
}
