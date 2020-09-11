package handler

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/lighttiger2505/sqls/internal/lsp"
)

func TestFormatting(t *testing.T) {
	tx := newTestContext()
	tx.initServer(t)
	defer tx.tearDown()

	uri := "file:///Users/octref/Code/css-test/test.sql"

	type formattingTestCase struct {
		name  string
		input string
		line  int
		col   int
		want  string
	}
	testCase := []formattingTestCase{
		{
			name:  "select",
			input: "SELECT ID , Name FROM city;",
			want:  "SELECT \n\tID, \n\tName \nFROM city;",
		},
		{
			name:  "member ident",
			input: "select ci.ID, ci.Name, co.Code, co.Name from city ci join country co on ci.CountryCode = co.Code;",
			want:  "select \n\tci.ID, \n\tci.Name, \n\tco.Code, \n\tco.Name \nfrom city ci \njoin country co \n\ton ci.CountryCode = co.Code;",
		},
	}

	for _, tt := range testCase {
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
			formattingParams := lsp.DocumentFormattingParams{
				TextDocument: lsp.TextDocumentIdentifier{
					URI: uri,
				},
				Options: lsp.FormattingOptions{
					TabSize:                0.0,
					InsertSpaces:           false,
					TrimTrailingWhitespace: false,
					InsertFinalNewline:     false,
					TrimFinalNewlines:      false,
				},
				WorkDoneProgressParams: lsp.WorkDoneProgressParams{
					WorkDoneToken: nil,
				},
			}

			var got []lsp.TextEdit
			if err := tx.conn.Call(tx.ctx, "textDocument/formatting", formattingParams, &got); err != nil {
				t.Fatal("conn.Call textDocument/formatting:", err)
			}
			if diff := cmp.Diff(tt.want, got[0].NewText); diff != "" {
				t.Errorf("unmatch (- want, + got):\n%s", diff)
			}
		})
	}
}
