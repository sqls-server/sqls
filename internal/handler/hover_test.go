package handler

import (
	"strings"
	"testing"

	"github.com/lighttiger2505/sqls/internal/config"
	"github.com/lighttiger2505/sqls/internal/database"
	"github.com/lighttiger2505/sqls/internal/lsp"
)

func TestHover(t *testing.T) {
	tx := newTestContext()
	tx.setup(t)
	defer tx.tearDown()

	didChangeConfigurationParams := lsp.DidChangeConfigurationParams{
		Settings: struct {
			SQLS *config.Config "json:\"sqls\""
		}{
			SQLS: &config.Config{
				Connections: []*database.DBConfig{
					{
						Driver:         "mock",
						DataSourceName: "",
					},
				},
			},
		},
	}
	if err := tx.conn.Call(tx.ctx, "workspace/didChangeConfiguration", didChangeConfigurationParams, nil); err != nil {
		t.Fatal("conn.Call workspace/didChangeConfiguration:", err)
	}

	uri := "file:///Users/octref/Code/css-test/test.sql"
	testcases := []struct {
		name   string
		input  string
		output string
		line   int
		col    int
	}{
		{
			name:   "not found head",
			input:  "SELECT ID, Name FROM city",
			output: "",
			line:   0,
			col:    7,
		},
		{
			name:   "not found tail",
			input:  "SELECT ID, Name FROM city",
			output: "",
			line:   0,
			col:    16,
		},
		{
			name:   "not found duplicate ident",
			input:  "SELECT Name FROM city, country",
			output: "",
			line:   0,
			col:    9,
		},
		{
			name:   "select ident head",
			input:  "SELECT ID, Name FROM city",
			output: "city.ID column",
			line:   0,
			col:    8,
		},
		{
			name:   "select ident tail",
			input:  "SELECT ID, Name FROM city",
			output: "city.Name column",
			line:   0,
			col:    15,
		},
		{
			name:   "select quated ident head",
			input:  "SELECT `ID`, Name FROM city",
			output: "city.ID column",
			line:   0,
			col:    8,
		},
		{
			name:   "select quated ident head",
			input:  "SELECT `ID`, Name FROM city",
			output: "city.ID column",
			line:   0,
			col:    11,
		},
		{
			name:   "table ident head",
			input:  "SELECT ID, Name FROM city",
			output: "city table",
			line:   0,
			col:    22,
		},
		{
			name:   "table ident tail",
			input:  "SELECT ID, Name FROM city",
			output: "city table",
			line:   0,
			col:    25,
		},
		{
			name:   "select member ident parent head",
			input:  "SELECT city.ID, city.Name FROM city",
			output: "city table",
			line:   0,
			col:    8,
		},
		{
			name:   "select member ident parent tail",
			input:  "SELECT city.ID, city.Name FROM city",
			output: "city table",
			line:   0,
			col:    20,
		},
		{
			name:   "select member ident child dot",
			input:  "SELECT city.ID, city.Name FROM city",
			output: "city.ID column",
			line:   0,
			col:    12,
		},
		{
			name:   "select member ident child head",
			input:  "SELECT city.ID, city.Name FROM city",
			output: "city.ID column",
			line:   0,
			col:    13,
		},
		{
			name:   "select member ident child tail",
			input:  "SELECT city.ID, city.Name FROM city",
			output: "city.Name column",
			line:   0,
			col:    25,
		},
		{
			name:   "select aliased member ident parent head",
			input:  "SELECT ci.ID, ci.Name FROM city AS ci",
			output: "city table",
			line:   0,
			col:    8,
		},
		{
			name:   "select aliased member ident parent tail",
			input:  "SELECT ci.ID, ci.Name FROM city AS ci",
			output: "city table",
			line:   0,
			col:    16,
		},
		{
			name:   "select aliased member ident child head",
			input:  "SELECT ci.ID, ci.Name FROM city AS ci",
			output: "city.ID column",
			line:   0,
			col:    10,
		},
		{
			name:   "select aliased member ident child head",
			input:  "SELECT ci.ID, ci.Name FROM city AS ci",
			output: "city.ID column",
			line:   0,
			col:    11,
		},
		{
			name:   "select aliased member ident child tail",
			input:  "SELECT ci.ID, ci.Name FROM city AS ci",
			output: "city.Name column",
			line:   0,
			col:    21,
		},
		{
			name:   "select subquery ident parent head",
			input:  "SELECT ID, Name FROM (SELECT ci.ID, ci.Name, ci.CountryCode, ci.District, ci.Population FROM city AS ci) as it",
			output: "ID subquery column",
			line:   0,
			col:    8,
		},
		{
			name:   "select subquery ident parent head",
			input:  "SELECT ID, Name FROM (SELECT ci.ID, ci.Name, ci.CountryCode, ci.District, ci.Population FROM city AS ci) as it",
			output: "Name subquery column",
			line:   0,
			col:    15,
		},
		{
			name:   "select subquery member ident parent head",
			input:  "SELECT it.ID, it.Name FROM (SELECT ci.ID, ci.Name, ci.CountryCode, ci.District, ci.Population FROM city AS ci) as it",
			output: "it subquery",
			line:   0,
			col:    8,
		},
		{
			name:   "select subquery member ident parent tail",
			input:  "SELECT it.ID, it.Name FROM (SELECT ci.ID, ci.Name, ci.CountryCode, ci.District, ci.Population FROM city AS ci) as it",
			output: "it subquery",
			line:   0,
			col:    16,
		},
		{
			name:   "select subquery member ident child head",
			input:  "SELECT it.ID, it.Name FROM (SELECT ci.ID, ci.Name, ci.CountryCode, ci.District, ci.Population FROM city AS ci) as it",
			output: "ID subquery column",
			line:   0,
			col:    11,
		},
		{
			name:   "select subquery member ident child tail",
			input:  "SELECT it.ID, it.Name FROM (SELECT ci.ID, ci.Name, ci.CountryCode, ci.District, ci.Population FROM city AS ci) as it",
			output: "Name subquery column",
			line:   0,
			col:    21,
		},
		{
			name:   "select aliased select identifer head",
			input:  "SELECT ID AS city_id, Name AS city_name FROM city",
			output: "city.ID column",
			line:   0,
			col:    14,
		},
		{
			name:   "select aliased select identifer tail",
			input:  "SELECT ID AS city_id, Name AS city_name FROM city",
			output: "city.Name column",
			line:   0,
			col:    39,
		},
		{
			name:   "select aliased select member identifer head",
			input:  "SELECT city.ID AS city_id, city.Name AS city_name FROM city",
			output: "city.ID column",
			line:   0,
			col:    19,
		},
		{
			name:   "select aliased select member identifer tail",
			input:  "SELECT city.ID AS city_id, city.Name AS city_name FROM city",
			output: "city.Name column",
			line:   0,
			col:    49,
		},
		{
			name: "multi line head",
			input: `SELECT
  ID,
  Name
FROM city
`,
			output: "city.Name column",
			line:   2,
			col:    3,
		},
		{
			name: "multi line tail",
			input: `SELECT
  ID,
  Name
FROM city
`,
			output: "city.Name column",
			line:   2,
			col:    6,
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
			// Create hover params
			hoverParams := lsp.HoverParams{
				TextDocumentPositionParams: lsp.TextDocumentPositionParams{
					TextDocument: lsp.TextDocumentIdentifier{
						URI: uri,
					},
					Position: lsp.Position{
						Line:      tt.line,
						Character: tt.col - 1,
					},
				},
			}

			var got lsp.Hover
			err := tx.conn.Call(tx.ctx, "textDocument/hover", hoverParams, &got)
			if err != nil {
				t.Errorf("conn.Call textDocument/hover: %+v", err)
				return
			}
			if tt.output == "" && got.Contents.Value != "" {
				t.Errorf("found hover, %q", got.Contents.Value)
				return
			}
			testHover(t, tt.output, got)
		})
	}
}

func testHover(t *testing.T, want string, hover lsp.Hover) {
	t.Helper()
	if !strings.HasPrefix(hover.Contents.Value, want) {
		t.Errorf("unmatched hover content prefix got: %q, expect: %q", hover.Contents.Value, want)
	}
}
