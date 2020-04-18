package handler

import (
	"testing"

	"github.com/lighttiger2505/sqls/internal/database"
	"github.com/lighttiger2505/sqls/internal/lsp"
)

func TestComplete(t *testing.T) {
	tx := newTestContext()
	tx.setup(t)
	defer tx.tearDown()

	didChangeConfigurationParams := lsp.DidChangeConfigurationParams{
		Settings: struct {
			SQLS *database.Config "json:\"sqls\""
		}{
			SQLS: &database.Config{
				Driver:         "mock",
				DataSourceName: "",
			},
		},
	}
	if err := tx.conn.Call(tx.ctx, "workspace/didChangeConfiguration", didChangeConfigurationParams, nil); err != nil {
		t.Fatal("conn.Call workspace/didChangeConfiguration:", err)
	}

	uri := "file:///Users/octref/Code/css-test/test.sql"
	testcases := []struct {
		name  string
		input string
		line  int
		col   int
		want  []string
	}{
		{
			name:  "select identifier",
			input: "select  from city",
			line:  0,
			col:   7,
			want: []string{
				"ID",
				"Name",
				"CountryCode",
				"District",
				"Population",
				"city",
				"country",
				"countrylanguage",
			},
		},
		{
			name:  "select identifier with table alias",
			input: "select  from city as c",
			line:  0,
			col:   7,
			want: []string{
				"ID",
				"Name",
				"CountryCode",
				"District",
				"Population",
				"c",
				"city",
				"country",
				"countrylanguage",
			},
		},
		{
			name:  "select identifier with table alias without as",
			input: "select  from city c",
			line:  0,
			col:   7,
			want: []string{
				"ID",
				"Name",
				"CountryCode",
				"District",
				"Population",
				"c",
				"city",
				"country",
				"countrylanguage",
			},
		},
		{
			name:  "select identifier filterd",
			input: "select Cou from city",
			line:  0,
			col:   10,
			want: []string{
				"CountryCode",
				"country",
				"countrylanguage",
			},
		},
		{
			name:  "select identifier list",
			input: "select id, cou from city",
			line:  0,
			col:   14,
			want: []string{
				"CountryCode",
				"country",
				"countrylanguage",
			},
		},
		{
			name:  "select identifier with comparison",
			input: "select 1 = cou from city",
			line:  0,
			col:   14,
			want: []string{
				"CountryCode",
				"country",
				"countrylanguage",
			},
		},
		{
			name:  "select identifier with operator",
			input: "select 1 + cou from city",
			line:  0,
			col:   14,
			want: []string{
				"CountryCode",
				"country",
				"countrylanguage",
			},
		},
		{
			name:  "select has aliased table identifier",
			input: "select c. from city as c",
			line:  0,
			col:   9,
			want: []string{
				"ID",
				"Name",
				"CountryCode",
				"District",
				"Population",
			},
		},
		{
			name:  "select has aliased without as table identifier",
			input: "select c. from city as c",
			line:  0,
			col:   9,
			want: []string{
				"ID",
				"Name",
				"CountryCode",
				"District",
				"Population",
			},
		},
		{
			name:  "from identifier",
			input: "select CountryCode from ",
			line:  0,
			col:   24,
			want: []string{
				"city",
				"country",
				"countrylanguage",
			},
		},
		{
			name:  "from identifier filterd",
			input: "select CountryCode from co",
			line:  0,
			col:   26,
			want: []string{
				"country",
				"countrylanguage",
			},
		},
		{
			name:  "join identifier",
			input: "select CountryCode from city left join ",
			line:  0,
			col:   39,
			want: []string{
				"ID",
				"Name",
				"CountryCode",
				"District",
				"Population",
				"city",
				"country",
				"countrylanguage",
			},
		},
		{
			name:  "join identifier filterd",
			input: "select CountryCode from city left join co",
			line:  0,
			col:   41,
			want: []string{
				"CountryCode",
				"country",
				"countrylanguage",
			},
		},
		{
			name:  "join on identifier filterd",
			input: "select CountryCode from city left join country on co",
			line:  0,
			col:   52,
			want: []string{
				"Code",
				"Continent",
				"Code2",
			},
		},
		{
			name:  "ORDER BY identifier",
			input: "SELECT ID, Name FROM city ORDER BY ",
			line:  0,
			col:   35,
			want: []string{
				"ID",
				"Name",
				"CountryCode",
				"District",
				"Population",
				"city",
				"country",
				"countrylanguage",
			},
		},
		{
			name:  "GROUP BY identifier",
			input: "SELECT CountryCode, COUNT(*) FROM city GROUP BY ",
			line:  0,
			col:   48,
			want: []string{
				"ID",
				"Name",
				"CountryCode",
				"District",
				"Population",
				"city",
				"country",
				"countrylanguage",
			},
		},
		{
			name:  "FROM identifiers in the sub query",
			input: "SELECT * FROM (SELECT * FROM ",
			line:  0,
			col:   29,
			want: []string{
				"city",
				"country",
				"countrylanguage",
			},
		},
		{
			name:  "filterd FROM identifiers in the sub query",
			input: "SELECT * FROM (SELECT * FROM co",
			line:  0,
			col:   29,
			want: []string{
				"country",
				"countrylanguage",
			},
		},
		{
			name:  "filterd SELECT identifiers in the sub query",
			input: "SELECT * FROM (SELECT Cou FROM city)",
			line:  0,
			col:   25,
			want: []string{
				"CountryCode",
				"country",
				"countrylanguage",
			},
		},
		{
			name:  "SELECT identifiers by sub query",
			input: "SELECT  FROM (SELECT ID as city_id, Name as city_name FROM city) as t",
			line:  0,
			col:   7,
			want: []string{
				"city_id",
				"city_name",
			},
		},
		{
			name:  "SELECT identifiers with multiple statement forcused first",
			input: "SELECT c. FROM city as c;SELECT c. FROM country as c;",
			line:  0,
			col:   9,
			want: []string{
				"ID",
				"Name",
				"CountryCode",
				"District",
				"Population",
			},
		},
		{
			name:  "SELECT identifiers with multiple statement forcused second",
			input: "SELECT c. FROM city as c;SELECT c. FROM country as c;",
			line:  0,
			col:   34,
			want: []string{
				"Code",
				"Name",
				"CountryCode",
				"Region",
				"SurfaceArea",
				"IndepYear",
				"LifeExpectancy",
				"GNP",
				"GNPOld",
				"LocalName",
				"GovernmentForm",
				"HeadOfState",
				"Capital",
				"Code2",
			},
		},
		{
			name:  "insert table reference",
			input: "INSERT INTO ",
			line:  0,
			col:   12,
			want: []string{
				"city",
				"country",
				"countrylanguage",
			},
		},
		{
			name:  "insert table reference filterd",
			input: "INSERT INTO co",
			line:  0,
			col:   12,
			want: []string{
				"country",
				"countrylanguage",
			},
		},
		{
			name:  "insert column non filter",
			input: "INSERT INTO city (",
			line:  0,
			col:   18,
			want: []string{
				"ID",
				"Name",
				"CountryCode",
				"District",
				"Population",
			},
		},
		{
			name:  "insert column filterd",
			input: "INSERT INTO city (cou",
			line:  0,
			col:   21,
			want: []string{
				"CountryCode",
			},
		},
		{
			name:  "insert columns",
			input: "INSERT INTO city (id, cou",
			line:  0,
			col:   25,
			want: []string{
				"CountryCode",
			},
		},
		{
			name:  "update table references",
			input: "UPDATE ",
			line:  0,
			col:   7,
			want: []string{
				"city",
				"country",
				"countrylanguage",
			},
		},
		{
			name:  "update table references filterd",
			input: "UPDATE co",
			line:  0,
			col:   9,
			want: []string{
				"country",
				"countrylanguage",
			},
		},
		{
			name:  "update column non filter",
			input: "UPDATE city SET ",
			line:  0,
			col:   16,
			want: []string{
				"ID",
				"Name",
				"CountryCode",
				"District",
				"Population",
			},
		},
		{
			name:  "update column filterd",
			input: "UPDATE city SET cou",
			line:  0,
			col:   19,
			want: []string{
				"CountryCode",
			},
		},
		{
			name:  "update columns",
			input: "UPDATE city SET CountryCode=12, Na",
			line:  0,
			col:   34,
			want: []string{
				"Name",
			},
		},
		{
			name:  "delete table references",
			input: "DELETE FROM ",
			line:  0,
			col:   12,
			want: []string{
				"city",
				"country",
				"countrylanguage",
			},
		},
		{
			name:  "delete table references filterd",
			input: "DELETE FROM co",
			line:  0,
			col:   14,
			want: []string{
				"country",
				"countrylanguage",
			},
		},
		{
			name:  "delete column non filter",
			input: "DELETE FROM city WHERE ",
			line:  0,
			col:   23,
			want: []string{
				"ID",
				"Name",
				"CountryCode",
				"District",
				"Population",
			},
		},
		{
			name:  "delete column filterd",
			input: "DELETE FROM city WHERE co",
			line:  0,
			col:   25,
			want: []string{
				"CountryCode",
			},
		},
	}

	for _, tt := range testcases {
		t.Run(tt.name, func(t *testing.T) {
			// Open dummy file
			didOpenParams := lsp.DidOpenTextDocumentParams{
				TextDocument: lsp.TextDocumentItem{
					URI:        uri,
					LanguageID: "sql",
					Version:    0,
					Text:       tt.input,
				},
			}
			if err := tx.conn.Call(tx.ctx, "textDocument/didOpen", didOpenParams, nil); err != nil {
				t.Fatal("conn.Call textDocument/didOpen:", err)
			}
			tx.testFile(t, didOpenParams.TextDocument.URI, didOpenParams.TextDocument.Text)
			// Create completion params
			commpletionParams := lsp.CompletionParams{
				TextDocumentPositionParams: lsp.TextDocumentPositionParams{
					TextDocument: lsp.TextDocumentIdentifier{
						URI: uri,
					},
					Position: lsp.Position{
						Line:      tt.line,
						Character: tt.col,
					},
				},
				CompletionContext: lsp.CompletionContext{
					TriggerKind:      0,
					TriggerCharacter: nil,
				},
			}

			var got []lsp.CompletionItem
			if err := tx.conn.Call(tx.ctx, "textDocument/completion", commpletionParams, &got); err != nil {
				t.Fatal("conn.Call textDocument/completion:", err)
			}
			testCompletionItem(t, tt.want, got)
		})
	}
}

func testCompletionItem(t *testing.T, expectLabels []string, gotItems []lsp.CompletionItem) {
	t.Helper()

	itemMap := map[string]struct{}{}
	for _, item := range gotItems {
		itemMap[item.Label] = struct{}{}
	}

	for _, el := range expectLabels {
		_, ok := itemMap[el]
		if !ok {
			t.Errorf("not included label, expect %q", el)
		}
	}
}
