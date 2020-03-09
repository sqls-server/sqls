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

func Test_focusedStatement(t *testing.T) {
	testcases := []struct {
		name  string
		input string
		pos   token.Pos
		want  string
	}{
		{
			name:  "",
			input: "select 1;select 2;select 3;",
			pos:   token.Pos{Line: 1, Col: 9},
			want:  "select 1;",
		},
		{
			name:  "",
			input: "select 1;select 2;select 3;",
			pos:   token.Pos{Line: 1, Col: 10},
			want:  "select 2;",
		},
	}
	for _, tt := range testcases {
		t.Run(tt.name, func(t *testing.T) {
			stmt := initExtractTable(t, tt.input)
			got, err := extractFocusedStatement(stmt, tt.pos)
			if err != nil {
				t.Fatalf("error: %+v", err)
			}
			if d := cmp.Diff(tt.want, got.String()); d != "" {
				t.Errorf("unmatched value: %s", d)
			}
		})
	}
}

func Test_encloseIsSubQuery(t *testing.T) {
	testcases := []struct {
		name  string
		input string
		pos   token.Pos
		want  bool
	}{
		{
			name:  "outer sub query",
			input: "select * from (select * from abc) as t",
			pos:   token.Pos{Line: 1, Col: 14},
			want:  false,
		},
		{
			name:  "inner sub query",
			input: "select * from (select * from abc) as t",
			pos:   token.Pos{Line: 1, Col: 15},
			want:  true,
		},
		{
			name:  "operator",
			input: "select (1 + 1)",
			pos:   token.Pos{Line: 1, Col: 11},
			want:  false,
		},
	}
	for _, tt := range testcases {
		t.Run(tt.name, func(t *testing.T) {
			stmt := initExtractTable(t, tt.input)
			list := stmt.GetTokens()[0].(ast.TokenList)
			got := encloseIsSubQuery(list, tt.pos)
			if tt.want != got {
				t.Errorf("want %v, got %v", tt.want, got)
			}
		})
	}
}

func TestExtractSubQueryView(t *testing.T) {
	testcases := []struct {
		name  string
		input string
		pos   token.Pos
		want  *SubQueryInfo
	}{
		{
			name:  "sub query",
			input: "select * (select city.ID, city.Name from dbs.city as ci) as sub",
			pos:   token.Pos{Line: 1, Col: 9},
			want: &SubQueryInfo{
				Name: "sub",
				Views: []*SubQueryView{
					&SubQueryView{
						Table: &TableInfo{
							DatabaseSchema: "dbs",
							Name:           "city",
							Alias:          "ci",
						},
						Columns: []string{
							"ID",
							"Name",
						},
					},
				},
			},
		},
		{
			name:  "not found sub query",
			input: "select * (select city.ID, city.Name from dbs.city as ci) as sub",
			pos:   token.Pos{Line: 1, Col: 10},
			want:  nil,
		},
		{
			name:  "astrisk identifier",
			input: "select * (select * from dbs.city as ci) as sub",
			pos:   token.Pos{Line: 1, Col: 9},
			want: &SubQueryInfo{
				Name: "sub",
				Views: []*SubQueryView{
					&SubQueryView{
						Table: &TableInfo{
							DatabaseSchema: "dbs",
							Name:           "city",
							Alias:          "ci",
						},
						Columns: []string{
							"*",
						},
					},
				},
			},
		},
	}
	for _, tt := range testcases {
		t.Run(tt.name, func(t *testing.T) {
			query := initExtractTable(t, tt.input)
			// stmt := query.GetTokens()[0].(ast.TokenList)
			// subQuery := stmt.GetTokens()[0].(ast.TokenList)
			got, err := ExtractSubQueryView(query, tt.pos)
			if err != nil {
				t.Fatalf("error: %+v", err)
			}
			if d := cmp.Diff(tt.want, got); d != "" {
				t.Errorf("unmatched value: %s", d)
			}
		})
	}
}

func TestExtractTable(t *testing.T) {
	testcases := []struct {
		name  string
		input string
		pos   token.Pos
		want  []*TableInfo
	}{
		{
			name:  "from only",
			input: "from abc",
			pos:   token.Pos{Line: 1, Col: 1},
			want: []*TableInfo{
				&TableInfo{
					Name: "abc",
				},
			},
		},
		{
			name:  "one table",
			input: "select * from abc",
			pos:   token.Pos{Line: 1, Col: 1},
			want: []*TableInfo{
				&TableInfo{
					Name: "abc",
				},
			},
		},
		{
			name:  "multiple table",
			input: "select * from abc, def",
			pos:   token.Pos{Line: 1, Col: 1},
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
			pos:   token.Pos{Line: 1, Col: 1},
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
			pos:   token.Pos{Line: 1, Col: 1},
			want: []*TableInfo{
				&TableInfo{
					DatabaseSchema: "abc",
					Name:           "def",
					Alias:          "ghi",
				},
			},
		},
		{
			name:  "focus outer sub query before",
			input: "select t.* from (select city_id, city_name from (select city.ID as city_id, city.Name as city_name from city) as t) as t",
			pos:   token.Pos{Line: 1, Col: 1},
			want: []*TableInfo{
				&TableInfo{
					DatabaseSchema: "",
					Name:           "",
					Alias:          "t",
				},
			},
		},
		{
			name:  "focus outer sub query after",
			input: "select t.* from (select city_id, city_name from (select city.ID as city_id, city.Name as city_name from city) as t) as t",
			pos:   token.Pos{Line: 1, Col: 120},
			want: []*TableInfo{
				&TableInfo{
					DatabaseSchema: "",
					Name:           "",
					Alias:          "t",
				},
			},
		},
		{
			name:  "focus middle sub query before",
			input: "select t.* from (select city_id, city_name from (select city.ID as city_id, city.Name as city_name from city) as t) as t",
			pos:   token.Pos{Line: 1, Col: 18},
			want: []*TableInfo{
				&TableInfo{
					DatabaseSchema: "",
					Name:           "",
					Alias:          "t",
				},
			},
		},
		{
			name:  "focus middle sub query after",
			input: "select t.* from (select city_id, city_name from (select city.ID as city_id, city.Name as city_name from city) as t) as t",
			pos:   token.Pos{Line: 1, Col: 114},
			want: []*TableInfo{
				&TableInfo{
					DatabaseSchema: "",
					Name:           "",
					Alias:          "t",
				},
			},
		},
		{
			name:  "focus deep sub query",
			input: "select t.* from (select city_id, city_name from (select ci.ID as city_id, ci.Name as city_name from city as ci) as t) as t",
			pos:   token.Pos{Line: 1, Col: 55},
			want: []*TableInfo{
				&TableInfo{
					DatabaseSchema: "",
					Name:           "city",
					Alias:          "ci",
				},
			},
		},
	}

	for _, tt := range testcases {
		t.Run(tt.name, func(t *testing.T) {
			stmt := initExtractTable(t, tt.input)
			got, err := ExtractTable(stmt, tt.pos)
			if err != nil {
				t.Fatalf("error: %+v", err)
			}
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
			name:  "prev select on identifier",
			input: "SELECT abc FROM def",
			pos:   token.Pos{Line: 1, Col: 10},
			matcher: astutil.NodeMatcher{
				ExpectKeyword: []string{"SELECT"},
			},
			want: true,
		},
		{
			name:  "prev select on member identifier",
			input: "SELECT abc.xxx FROM def",
			pos:   token.Pos{Line: 1, Col: 13},
			matcher: astutil.NodeMatcher{
				ExpectKeyword: []string{"SELECT"},
			},
			want: true,
		},
		{
			name:  "prev select on invalid identifier",
			input: "SELECT abc. FROM def",
			pos:   token.Pos{Line: 1, Col: 11},
			matcher: astutil.NodeMatcher{
				ExpectKeyword: []string{"SELECT"},
			},
			want: true,
		},
		{
			name:  "prev select on invalid identifier list",
			input: "SELECT a, b,       FROM def",
			pos:   token.Pos{Line: 1, Col: 18},
			matcher: astutil.NodeMatcher{
				ExpectKeyword: []string{"SELECT"},
			},
			want: true,
		},
		{
			name:  "prev from",
			input: "SELECT * FROM ",
			pos:   token.Pos{Line: 1, Col: 14},
			matcher: astutil.NodeMatcher{
				ExpectKeyword: []string{"FROM"},
			},
			want: true,
		},
		{
			name:  "prev from on identifier",
			input: "SELECT * FROM def",
			pos:   token.Pos{Line: 1, Col: 15},
			matcher: astutil.NodeMatcher{
				ExpectKeyword: []string{"FROM"},
			},
			want: true,
		},
		{
			name:  "prev order by",
			input: "SELECT * FROM abc ORDER BY ",
			pos:   token.Pos{Line: 1, Col: 27},
			matcher: astutil.NodeMatcher{
				ExpectKeyword: []string{"ORDER BY"},
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
