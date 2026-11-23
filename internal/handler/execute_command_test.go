package handler

import (
	"bytes"
	"strings"
	"testing"

	"github.com/olekukonko/tablewriter"
	"github.com/olekukonko/tablewriter/tw"
	"github.com/sqls-server/sqls/internal/config"
	"github.com/sqls-server/sqls/internal/database"
	"github.com/sqls-server/sqls/internal/lsp"
)

func Test_executeQuery(t *testing.T) {
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

	uri := "file:///test.sql"
	text := "SELECT 1; SELECT 2;"
	didOpenParams := lsp.DidOpenTextDocumentParams{
		TextDocument: lsp.TextDocumentItem{
			URI:        uri,
			LanguageID: "sql",
			Version:    0,
			Text:       text,
		},
	}
	if err := tx.conn.Call(tx.ctx, "textDocument/didOpen", didOpenParams, nil); err != nil {
		t.Fatal("conn.Call textDocument/didOpen:", err)
	}
	tx.testFile(t, didOpenParams.TextDocument.URI, didOpenParams.TextDocument.Text)

	// executeCommandParams := lsp.ExecuteCommandParams{
	// 	Command:   CommandExecuteQuery,
	// 	Arguments: []interface{}{uri},
	// }
	// var got interface{}
	// tx.conn.Call(tx.ctx, "workspace/executeCommand", executeCommandParams, &got)
	// pass error
}

func Test_queryResultHeaderPreservesColumnNames(t *testing.T) {
	// Regression test: column names with underscores should not be
	// auto-formatted (e.g. "user_name" must not become "USER NAME").
	buf := new(bytes.Buffer)
	columns := []string{"user_name", "created_at", "is_active"}
	rows := [][]string{{"alice", "2024-01-01", "true"}}

	table := tablewriter.NewTable(buf, tablewriter.WithHeaderConfig(tw.CellConfig{
		Formatting: tw.CellFormatting{AutoFormat: tw.Off},
	}))
	headers := make([]any, len(columns))
	for i, v := range columns {
		headers[i] = v
	}
	table.Header(headers...)
	for _, row := range rows {
		vals := make([]any, len(row))
		for i, v := range row {
			vals[i] = v
		}
		if err := table.Append(vals...); err != nil {
			t.Fatal(err)
		}
	}
	if err := table.Render(); err != nil {
		t.Fatal(err)
	}

	result := buf.String()
	for _, col := range columns {
		if !strings.Contains(result, col) {
			t.Errorf("expected column name %q to be preserved in output, got:\n%s", col, result)
		}
	}
}

func Test_extractRangeText(t *testing.T) {
	type args struct {
		text      string
		startLine int
		startChar int
		endLine   int
		endChar   int
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "extract single line",
			args: args{
				text:      "select * from city",
				startLine: 0,
				startChar: 0,
				endLine:   0,
				endChar:   8,
			},
			want: "select *",
		},
		{
			name: "extract multi line with not equal start end",
			args: args{
				text:      "select 1;\nselect 2;\nselect 3;",
				startLine: 0,
				startChar: 7,
				endLine:   2,
				endChar:   8,
			},
			want: "1;\nselect 2;\nselect 3",
		},
		{
			name: "extract multi line with equal start end",
			args: args{
				text:      "select 1;\nselect 2;\nselect 3;",
				startLine: 1,
				startChar: 2,
				endLine:   1,
				endChar:   6,
			},
			want: "lect",
		},
		{
			name: "extract unicode using utf16 offsets",
			args: args{
				text:      "SELECT 'あ😀い';",
				startLine: 0,
				startChar: 8,
				endLine:   0,
				endChar:   12,
			},
			want: "あ😀い",
		},
		{
			name: "extract long single line without scanner limit",
			args: args{
				text:      strings.Repeat("a", 70000) + "SELECT",
				startLine: 0,
				startChar: 70000,
				endLine:   0,
				endChar:   70006,
			},
			want: "SELECT",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := extractRangeText(tt.args.text, tt.args.startLine, tt.args.startChar, tt.args.endLine, tt.args.endChar); got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func Test_parseQueryFlags(t *testing.T) {
	tests := []struct {
		name         string
		args         []interface{}
		wantVertical bool
		wantMarkdown bool
	}{
		{name: "no flags", args: nil},
		{name: "show vertical", args: []interface{}{"-show-vertical"}, wantVertical: true},
		{name: "non-string argument is ignored", args: []interface{}{nil, 1.0}},
		{name: "unknown flag is ignored", args: []interface{}{"-unknown"}},
		{name: "flag after a non-string argument", args: []interface{}{nil, "-show-vertical"}, wantVertical: true},
		{name: "markdown", args: []interface{}{"-markdown"}, wantMarkdown: true},
		{name: "markdown after nil show vertical", args: []interface{}{nil, "-markdown"}, wantMarkdown: true},
		{name: "vertical takes precedence over markdown", args: []interface{}{"-show-vertical", "-markdown"}, wantVertical: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotVertical, gotMarkdown := parseQueryFlags(tt.args)
			if gotVertical != tt.wantVertical || gotMarkdown != tt.wantMarkdown {
				t.Errorf("parseQueryFlags() = (%v, %v), want (%v, %v)", gotVertical, gotMarkdown, tt.wantVertical, tt.wantMarkdown)
			}
		})
	}
}

func Test_renderQueryResult(t *testing.T) {
	columns := []string{"id", "user_name"}
	rows := [][]string{{"1", "alice"}, {"2", "bob"}}

	tests := []struct {
		name     string
		columns  []string
		rows     [][]string
		vertical bool
		markdown bool
		want     string
	}{
		{
			name:    "table",
			columns: columns,
			rows:    rows,
			want: `┌───┬─────────┐
│id │user_name│
├───┼─────────┤
│ 1 │ alice   │
│ 2 │ bob     │
└───┴─────────┘
2 rows in set

`,
		},
		{
			name:    "table without rows",
			columns: columns,
			want: `┌──┬─────────┐
│id│user_name│
└──┴─────────┘
0 rows in set

`,
		},
		{
			name:     "vertical",
			columns:  columns,
			rows:     rows,
			vertical: true,
			want: `***************************[ 1. row ]***************************
       id | 1
user_name | alice
***************************[ 2. row ]***************************
       id | 2
user_name | bob
2 rows in set

`,
		},
		{
			name:     "markdown",
			columns:  columns,
			rows:     rows,
			markdown: true,
			want:     "\n| id | user_name |\n|:--|:--------|\n| 1 | alice   |\n| 2 | bob     |\n\n2 rows in set\n\n",
		},
		{
			name:     "markdown without rows",
			columns:  columns,
			markdown: true,
			want:     "\n| id | user_name |\n|:--|:--------|\n\n0 rows in set\n\n",
		},
		{
			name:     "markdown escapes pipes, backslashes and newlines",
			columns:  []string{"a|b", "note"},
			rows:     [][]string{{"x\ny", `p\|q`}},
			markdown: true,
			want:     "\n| a\\|b   | note   |\n|:-------|:-------|\n| x<br>y | p\\\\\\|q |\n\n1 rows in set\n\n",
		},
		{
			name:     "markdown escapes HTML so values such as NULL are visible",
			columns:  []string{"a"},
			rows:     [][]string{{"<nil>"}, {"a&b"}},
			markdown: true,
			want:     "\n| a           |\n|:------------|\n| &lt;nil&gt; |\n| a&amp;b     |\n\n2 rows in set\n\n",
		},
		{
			name:     "vertical takes precedence over markdown in the renderer",
			columns:  columns,
			rows:     rows[:1],
			vertical: true,
			markdown: true,
			want: `***************************[ 1. row ]***************************
       id | 1
user_name | alice
1 rows in set

`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := renderQueryResult(tt.columns, tt.rows, tt.vertical, tt.markdown)
			if err != nil {
				t.Fatal(err)
			}
			if got != tt.want {
				t.Errorf("renderQueryResult() =\n%s\nwant\n%s", got, tt.want)
			}
		})
	}
}
