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
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := extractRangeText(tt.args.text, tt.args.startLine, tt.args.startChar, tt.args.endLine, tt.args.endChar); got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}
