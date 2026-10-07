package completer

import (
	"reflect"
	"testing"

	"github.com/sqls-server/sqls/internal/database"
	"github.com/sqls-server/sqls/internal/lsp"
)

func TestGetBeforeCursorText(t *testing.T) {
	input := `SELECT
a, b, c
FROM
hogetable
`
	tests := []struct {
		in   string
		line int
		char int
		out  string
	}{
		{input, 1, 2, "SE"},
		{input, 2, 3, "SELECT\na, "},
		{input, 3, 4, "SELECT\na, b, c\nFROM"},
		{input, 4, 5, "SELECT\na, b, c\nFROM\nhoget"},
		{"select 'テスト', ci", 1, 16, "select 'テスト', ci"},
		{"select '😀', ci", 1, 15, "select '😀', ci"},
		{"select '😀', ci", 1, 13, "select '😀', "},
		{"select 1", 1, 100, "select 1"},
	}
	for _, tt := range tests {
		got := getBeforeCursorText(tt.in, tt.line, tt.char)
		if tt.out != got {
			t.Errorf("want %#v, got %#v", tt.out, got)
		}
	}
}

func TestGetLastWord(t *testing.T) {
	input := `SELECT
    a, b, c
FROM  
    hogetable
`
	tests := []struct {
		name string
		in   string
		line int
		char int
		out  string
	}{
		{"", "SELECT  FROM def", 1, 7, ""},
		{"", input, 1, 2, "SE"},
		{"", input, 2, 3, ""},
		{"", input, 3, 4, "FROM"},
		{"", input, 3, 6, ""},
		{"", input, 4, 5, "h"},
		{"", "`ident", 1, 6, "`ident"},
		{"", "parent.`ident", 1, 13, "`ident"},
		{"", "`parent`.`ident", 1, 15, "`ident"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getLastWord(tt.in, tt.line, tt.char)
			if tt.out != got {
				t.Errorf("want %#v, got %#v", tt.out, got)
			}
		})
	}
}

func Test_completionTypeIs(t *testing.T) {
	type args struct {
	}
	tests := []struct {
		name            string
		completionTypes []completionType
		expect          completionType
		want            bool
	}{
		{
			completionTypes: []completionType{
				CompletionTypeColumn,
			},
			expect: CompletionTypeColumn,
			want:   true,
		},
		{
			completionTypes: []completionType{
				CompletionTypeTable,
				CompletionTypeView,
				CompletionTypeFunction,
				CompletionTypeColumn,
			},
			expect: CompletionTypeColumn,
			want:   true,
		},
		{
			completionTypes: []completionType{
				CompletionTypeTable,
				CompletionTypeView,
				CompletionTypeFunction,
			},
			expect: CompletionTypeColumn,
			want:   false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := completionTypeIs(tt.completionTypes, tt.expect); got != tt.want {
				t.Errorf("completionTypeIs() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestComplete(t *testing.T) {
	tests := []struct {
		name      string
		text      string
		lowerCase bool
		expected  []lsp.CompletionItem
	}{
		{
			name: "keyword",
			text: "sel",
			expected: []lsp.CompletionItem{
				{
					Label:    "SELECT",
					Kind:     lsp.KeywordCompletion,
					Detail:   "keyword",
					SortText: "9999SELECT",
				},
			},
		},
		{
			name:      "keyword-lowercase",
			text:      "sel",
			lowerCase: true,
			expected: []lsp.CompletionItem{
				{
					Label:    "select",
					Kind:     lsp.KeywordCompletion,
					Detail:   "keyword",
					SortText: "9999select",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			c := NewCompleter(nil)
			got, err := c.Complete("sel", lsp.CompletionParams{
				TextDocumentPositionParams: lsp.TextDocumentPositionParams{
					Position: lsp.Position{
						Line:      0,
						Character: len(tt.text),
					},
				},
			}, tt.lowerCase)
			if err != nil {
				t.Fatal(err)
			}

			if !reflect.DeepEqual(got, tt.expected) {
				t.Errorf("\nwant: %v\ngot:  %v", tt.expected, got)
			}
		})
	}
}

func TestGenerateAlias(t *testing.T) {
	noMatchesTable := make(map[string]interface{})
	noMatchesTable["XX"] = true
	matchesTable := make(map[string]interface{})
	matchesTable["XX"] = true
	matchesTable["T1"] = true

	tests := []struct {
		name  string
		table string
		tMap  map[string]interface{}
		want  string
	}{
		{
			"no matches",
			"Table",
			noMatchesTable,
			"T1",
		},
		{
			"matches",
			"Table",
			matchesTable,
			"T2",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := generateTableAlias(tt.table, tt.tMap); got != tt.want {
				t.Errorf("generateAlias() = %v, want  %v", got, tt.want)
			}
		})
	}
}

func TestComplete_SchemaAware(t *testing.T) {
	// Setup a DBCache with a non-default schema
	cache := &database.DBCache{
		Schemas: map[string]string{
			"PUBLIC":    "public",
			"MY_SCHEMA": "my_schema",
		},
		SchemaTables: map[string][]string{
			"PUBLIC":    {"cat"},
			"MY_SCHEMA": {"owner"},
		},
		ColumnsWithParent: map[string][]*database.ColumnDesc{
			"PUBLIC\tCAT": {
				{
					ColumnBase: database.ColumnBase{Schema: "public", Table: "cat", Name: "age"},
					Type:       "integer",
				},
				{
					ColumnBase: database.ColumnBase{Schema: "public", Table: "cat", Name: "owner_name"},
					Type:       "varchar(40)",
				},
			},
			"MY_SCHEMA\tOWNER": {
				{
					ColumnBase: database.ColumnBase{Schema: "my_schema", Table: "owner", Name: "age"},
					Type:       "integer",
				},
				{
					ColumnBase: database.ColumnBase{Schema: "my_schema", Table: "owner", Name: "name"},
					Type:       "varchar(40)",
				},
			},
		},
	}

	c := NewCompleter(cache)

	t.Run("column candidates with alias on non-default schema", func(t *testing.T) {
		text := "SELECT o. FROM my_schema.owner AS o"
		// Position cursor right after 'o.'
		items, err := c.Complete(text, lsp.CompletionParams{
			TextDocumentPositionParams: lsp.TextDocumentPositionParams{
				Position: lsp.Position{
					Line:      0,
					Character: 9,
				},
			},
		}, false)
		if err != nil {
			t.Fatal(err)
		}

		labels := make(map[string]bool)
		for _, it := range items {
			labels[it.Label] = true
		}
		if !labels["age"] {
			t.Errorf("expected 'age' in completion items, got: %+v", items)
		}
		if !labels["name"] {
			t.Errorf("expected 'name' in completion items, got: %+v", items)
		}
	})

	t.Run("column candidates without alias on non-default schema", func(t *testing.T) {
		text := "SELECT owner. FROM my_schema.owner"
		// Position cursor right after 'owner.'
		items, err := c.Complete(text, lsp.CompletionParams{
			TextDocumentPositionParams: lsp.TextDocumentPositionParams{
				Position: lsp.Position{
					Line:      0,
					Character: 13,
				},
			},
		}, false)
		if err != nil {
			t.Fatal(err)
		}

		labels := make(map[string]bool)
		for _, it := range items {
			labels[it.Label] = true
		}
		if !labels["age"] {
			t.Errorf("expected 'age' in completion items, got: %+v", items)
		}
		if !labels["name"] {
			t.Errorf("expected 'name' in completion items, got: %+v", items)
		}
	})

	t.Run("table candidates after schema prefix", func(t *testing.T) {
		text := "SELECT * FROM my_schema."
		items, err := c.Complete(text, lsp.CompletionParams{
			TextDocumentPositionParams: lsp.TextDocumentPositionParams{
				Position: lsp.Position{
					Line:      0,
					Character: 24,
				},
			},
		}, false)
		if err != nil {
			t.Fatal(err)
		}

		labels := make(map[string]bool)
		for _, it := range items {
			labels[it.Label] = true
			if it.Label == "owner" && it.Documentation == nil {
				t.Errorf("expected documentation with columns for 'owner', got nil")
			}
		}
		if !labels["owner"] {
			t.Errorf("expected 'owner' table in completion items, got: %+v", items)
		}
	})
}
