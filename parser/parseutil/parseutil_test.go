package parseutil

import (
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/lighttiger2505/sqls/ast"
	"github.com/lighttiger2505/sqls/ast/astutil"
	"github.com/lighttiger2505/sqls/parser"
	"github.com/lighttiger2505/sqls/token"
)

func Test_extractFocusedStatement(t *testing.T) {
	testcases := []struct {
		name  string
		input string
		pos   token.Pos
		want  string
	}{
		{
			name:  "before semicolon",
			input: "select 1;select 2;select 3;",
			pos:   token.Pos{Line: 0, Col: 9},
			want:  "select 1;",
		},
		{
			name:  "after semicolon",
			input: "select 1;select 2;select 3;",
			pos:   token.Pos{Line: 0, Col: 10},
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
			pos:   token.Pos{Line: 0, Col: 14},
			want:  false,
		},
		{
			name:  "inner sub query",
			input: "select * from (select * from abc) as t",
			pos:   token.Pos{Line: 0, Col: 15},
			want:  true,
		},
		{
			name:  "operator",
			input: "select (1 + 1)",
			pos:   token.Pos{Line: 0, Col: 11},
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

func TestExtractSubQueryViews(t *testing.T) {
	testcases := []struct {
		name  string
		input string
		pos   token.Pos
		want  []*SubQueryInfo
	}{
		{
			name:  "single",
			input: "SELECT * FROM (SELECT ci.ID, ci.Name FROM world.city AS ci) AS sub",
			pos:   token.Pos{Line: 0, Col: 14},
			want: []*SubQueryInfo{
				{
					Name: "sub",
					Views: []*SubQueryView{
						{
							Table: &TableInfo{
								DatabaseSchema: "world",
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
		},
		{
			name:  "double",
			input: "select * from (select * from city) as sub1, (select * from country) as sub2 limit 1",
			pos:   token.Pos{Line: 0, Col: 14},
			want: []*SubQueryInfo{
				{
					Name: "sub1",
					Views: []*SubQueryView{
						{
							Table: &TableInfo{
								DatabaseSchema: "",
								Name:           "city",
								Alias:          "",
							},
							Columns: []string{
								"*",
							},
						},
					},
				},
				{
					Name: "sub2",
					Views: []*SubQueryView{
						{
							Table: &TableInfo{
								DatabaseSchema: "",
								Name:           "country",
								Alias:          "",
							},
							Columns: []string{
								"*",
							},
						},
					},
				},
			},
		},
		{
			name:  "aliased column",
			input: "SELECT * FROM (SELECT ci.ID AS city_id, ci.Name AS city_name FROM world.city AS ci) AS sub",
			pos:   token.Pos{Line: 0, Col: 14},
			want: []*SubQueryInfo{
				{
					Name: "sub",
					Views: []*SubQueryView{
						{
							Table: &TableInfo{
								DatabaseSchema: "world",
								Name:           "city",
								Alias:          "ci",
							},
							Columns: []string{
								"city_id",
								"city_name",
							},
						},
					},
				},
			},
		},
		{
			name:  "not found sub query",
			input: "SELECT * FROM (SELECT ci.ID, ci.Name FROM world.city AS ci) AS sub",
			pos:   token.Pos{Line: 0, Col: 15},
			want:  nil,
		},
		{
			name:  "astrisk identifier",
			input: "SELECT * FROM (SELECT * FROM world.city AS ci) AS sub",
			pos:   token.Pos{Line: 0, Col: 14},
			want: []*SubQueryInfo{
				{
					Name: "sub",
					Views: []*SubQueryView{
						{
							Table: &TableInfo{
								DatabaseSchema: "world",
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
		},
		{
			name:  "position of outer sub query",
			input: "SELECT * FROM (SELECT it.ID, it.Name FROM (SELECT ci.ID, ci.Name, ci.CountryCode, ci.District, ci.Population FROM world.city AS ci) AS it) AS ot",
			pos:   token.Pos{Line: 0, Col: 14},
			want: []*SubQueryInfo{
				{
					Name: "ot",
					Views: []*SubQueryView{
						{
							Table: &TableInfo{
								DatabaseSchema: "world",
								Name:           "city",
								Alias:          "it",
							},
							Columns: []string{
								"ID",
								"Name",
							},
						},
					},
				},
			},
		},
		{
			name:  "position of inner sub query",
			input: "SELECT * FROM (SELECT it.ID, it.Name FROM (SELECT ci.ID, ci.Name, ci.CountryCode, ci.District, ci.Population FROM world.city AS ci) AS it) AS ot",
			pos:   token.Pos{Line: 0, Col: 16},
			want: []*SubQueryInfo{
				{
					Name: "it",
					Views: []*SubQueryView{
						{
							Table: &TableInfo{
								DatabaseSchema: "world",
								Name:           "city",
								Alias:          "ci",
							},
							Columns: []string{
								"ID",
								"Name",
								"CountryCode",
								"District",
								"Population",
							},
						},
					},
				},
			},
		},
		{
			name:  "position of sub query in sub query",
			input: "SELECT * FROM (SELECT it.ID, it.Name FROM (SELECT ci.ID, ci.Name, ci.CountryCode, ci.District, ci.Population FROM world.city AS ci) AS it) AS ot",
			pos:   token.Pos{Line: 0, Col: 44},
			want:  nil,
		},
		{
			name:  "recurcive parse sub query",
			input: "SELECT * FROM (SELECT * FROM (SELECT * FROM (SELECT ci.ID, ci.Name FROM world.city AS ci) AS t) AS t) AS t",
			pos:   token.Pos{Line: 0, Col: 1},
			want: []*SubQueryInfo{
				{
					Name: "t",
					Views: []*SubQueryView{
						{
							Table: &TableInfo{
								DatabaseSchema: "world",
								Name:           "city",
								Alias:          "t",
							},
							Columns: []string{
								"ID",
								"Name",
							},
						},
					},
				},
			},
		},
	}
	for _, tt := range testcases {
		t.Run(tt.name, func(t *testing.T) {
			query := initExtractTable(t, tt.input)
			got, err := ExtractSubQueryViews(query, tt.pos)
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
			pos:   token.Pos{Line: 0, Col: 1},
			want: []*TableInfo{
				{
					Name: "abc",
				},
			},
		},
		{
			name:  "join only",
			input: "join abc",
			pos:   token.Pos{Line: 0, Col: 1},
			want: []*TableInfo{
				{
					Name: "abc",
				},
			},
		},
		{
			name:  "select table reference",
			input: "select * from abc",
			pos:   token.Pos{Line: 0, Col: 1},
			want: []*TableInfo{
				{
					Name: "abc",
				},
			},
		},
		{
			name:  "select table references",
			input: "select * from abc, def",
			pos:   token.Pos{Line: 0, Col: 1},
			want: []*TableInfo{
				{
					Name: "abc",
				},
				{
					Name: "def",
				},
			},
		},
		{
			name:  "select join table reference",
			input: "select * from abc left join def on abc.id = def.id",
			pos:   token.Pos{Line: 0, Col: 1},
			want: []*TableInfo{
				{
					Name: "abc",
				},
				{
					Name: "def",
				},
			},
		},
		{
			name:  "multiple statement before",
			input: "select * from abc;select * from def;select * from ghi",
			pos:   token.Pos{Line: 0, Col: 1},
			want: []*TableInfo{
				{
					Name: "abc",
				},
			},
		},
		{
			name:  "multiple statement center",
			input: "select * from abc;select * from def;select * from ghi",
			pos:   token.Pos{Line: 0, Col: 19},
			want: []*TableInfo{
				{
					Name: "def",
				},
			},
		},
		{
			name:  "multiple statement after",
			input: "select * from abc;select * from def;select * from ghi",
			pos:   token.Pos{Line: 0, Col: 37},
			want: []*TableInfo{
				{
					Name: "ghi",
				},
			},
		},
		{
			name:  "with database schema",
			input: "select * from abc.def",
			pos:   token.Pos{Line: 0, Col: 1},
			want: []*TableInfo{
				{
					DatabaseSchema: "abc",
					Name:           "def",
				},
			},
		},
		{
			name:  "with database schema and alias",
			input: "select * from abc.def as ghi",
			pos:   token.Pos{Line: 0, Col: 1},
			want: []*TableInfo{
				{
					DatabaseSchema: "abc",
					Name:           "def",
					Alias:          "ghi",
				},
			},
		},
		{
			name:  "sub query",
			input: "FROM (SELECT ID as city_id, Name as city_name FROM city) as t",
			pos:   token.Pos{Line: 0, Col: 1},
			want:  []*TableInfo{},
		},
		{
			name:  "focus outer sub query before",
			input: "select t.* from (select city_id, city_name from (select city.ID as city_id, city.Name as city_name from city) as t) as t",
			pos:   token.Pos{Line: 0, Col: 1},
			want:  []*TableInfo{},
		},
		{
			name:  "focus outer sub query after",
			input: "select t.* from (select city_id, city_name from (select city.ID as city_id, city.Name as city_name from city) as t) as t",
			pos:   token.Pos{Line: 0, Col: 120},
			want:  []*TableInfo{},
		},
		{
			name:  "focus middle sub query before",
			input: "select t.* from (select city_id, city_name from (select city.ID as city_id, city.Name as city_name from city) as t) as t",
			pos:   token.Pos{Line: 0, Col: 18},
			want:  []*TableInfo{},
		},
		{
			name:  "focus middle sub query after",
			input: "select t.* from (select city_id, city_name from (select city.ID as city_id, city.Name as city_name from city) as t) as t",
			pos:   token.Pos{Line: 0, Col: 114},
			want:  []*TableInfo{},
		},
		{
			name:  "focus deep sub query",
			input: "select t.* from (select city_id, city_name from (select ci.ID as city_id, ci.Name as city_name from city as ci) as t) as t",
			pos:   token.Pos{Line: 0, Col: 55},
			want: []*TableInfo{
				{
					DatabaseSchema: "",
					Name:           "city",
					Alias:          "ci",
				},
			},
		},
		{
			name:  "insert",
			input: "insert into abc",
			pos:   token.Pos{Line: 0, Col: 1},
			want: []*TableInfo{
				{
					DatabaseSchema: "",
					Name:           "abc",
					Alias:          "",
				},
			},
		},
		{
			name:  "update",
			input: "update abc",
			pos:   token.Pos{Line: 0, Col: 1},
			want: []*TableInfo{
				{
					DatabaseSchema: "",
					Name:           "abc",
					Alias:          "",
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

	parsed, err := parser.Parse(input)
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
			pos:   token.Pos{Line: 0, Col: 7},
			matcher: astutil.NodeMatcher{
				ExpectKeyword: []string{"SELECT"},
			},
			want: true,
		},
		{
			name:  "prev select on identifier",
			input: "SELECT abc FROM def",
			pos:   token.Pos{Line: 0, Col: 10},
			matcher: astutil.NodeMatcher{
				ExpectKeyword: []string{"SELECT"},
			},
			want: true,
		},
		{
			name:  "prev select on member identifier",
			input: "SELECT abc.xxx FROM def",
			pos:   token.Pos{Line: 0, Col: 13},
			matcher: astutil.NodeMatcher{
				ExpectKeyword: []string{"SELECT"},
			},
			want: true,
		},
		{
			name:  "prev select on invalid identifier",
			input: "SELECT abc. FROM def",
			pos:   token.Pos{Line: 0, Col: 11},
			matcher: astutil.NodeMatcher{
				ExpectKeyword: []string{"SELECT"},
			},
			want: true,
		},
		{
			name:  "prev select on invalid identifier list",
			input: "SELECT a, b,       FROM def",
			pos:   token.Pos{Line: 0, Col: 18},
			matcher: astutil.NodeMatcher{
				ExpectKeyword: []string{"SELECT"},
			},
			want: true,
		},
		{
			name:  "prev from",
			input: "SELECT * FROM ",
			pos:   token.Pos{Line: 0, Col: 14},
			matcher: astutil.NodeMatcher{
				ExpectKeyword: []string{"FROM"},
			},
			want: true,
		},
		{
			name:  "prev from on identifier",
			input: "SELECT * FROM def",
			pos:   token.Pos{Line: 0, Col: 15},
			matcher: astutil.NodeMatcher{
				ExpectKeyword: []string{"FROM"},
			},
			want: true,
		},
		{
			name:  "insert into",
			input: "insert into city (abc",
			pos:   token.Pos{Line: 0, Col: 21},
			matcher: astutil.NodeMatcher{
				ExpectTokens: []token.Kind{token.LParen},
			},
			want: true,
		},
		{
			name:  "delete from",
			input: "delete from city",
			pos:   token.Pos{Line: 0, Col: 16},
			matcher: astutil.NodeMatcher{
				ExpectKeyword: []string{"DELETE FROM"},
			},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// initialize

			parsed, err := parser.Parse(tt.input)
			if err != nil {
				t.Fatalf("error %+v\n", err)
			}
			nodeWalker := NewNodeWalker(parsed, tt.pos)

			// execute
			if got := nodeWalker.PrevNodesIs(true, tt.matcher); got != tt.want {
				t.Errorf("nodeWalker.PrevNodesIs() = %v, want %v", got, tt.want)
				curNodes := nodeWalker.CurNodes()
				prevNodes := nodeWalker.PrevNodes(true)
				for i := 0; i < len(nodeWalker.Paths); i++ {
					fmt.Printf("%d CurNode: %q, PrevNode: %q\n", i, curNodes[i], prevNodes[i])
				}
			}
		})
	}
}
