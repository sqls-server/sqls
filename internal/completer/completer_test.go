package completer

import (
	"reflect"
	"testing"

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
