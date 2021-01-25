package handler

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/lighttiger2505/sqls/internal/config"
	"github.com/lighttiger2505/sqls/internal/database"
	"github.com/lighttiger2505/sqls/internal/lsp"
)

type signatureHelpTestCase struct {
	name  string
	input string
	line  int
	col   int
	want  lsp.SignatureHelp
}

func genInsertPositionTest(col int, wantActiveParameter int) signatureHelpTestCase {
	return signatureHelpTestCase{
		name:  "",
		input: "insert into city (ID, Name, CountryCode) VALUES (123, 'aaa', '2020')",
		line:  0,
		col:   col,
		want: lsp.SignatureHelp{
			Signatures: []lsp.SignatureInformation{
				{
					Label:         "city (ID, Name, CountryCode)",
					Documentation: "hogehoge",
					Parameters: []lsp.ParameterInformation{
						{
							Label:         "ID",
							Documentation: "",
						},
						{
							Label:         "Name",
							Documentation: "",
						},
						{
							Label:         "CountryCode",
							Documentation: "",
						},
					},
				},
			},
			ActiveSignature: 0.0,
			ActiveParameter: float64(wantActiveParameter),
		},
	}
}

func TestSignatureHelp(t *testing.T) {
	tx := newTestContext()
	tx.initServer(t)
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

	// input is "insert into city (ID, Name, CountryCode) VALUES (123, 'aaa', '2020')"
	cases := []signatureHelpTestCase{
		genInsertPositionTest(50, 0),
		genInsertPositionTest(52, 0),
		genInsertPositionTest(53, 1),
		genInsertPositionTest(59, 1),
		genInsertPositionTest(60, 2),
		genInsertPositionTest(67, 2),
	}

	uri := "file:///Users/octref/Code/css-test/test.sql"
	for _, tt := range cases {
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
			params := lsp.SignatureHelpParams{
				TextDocumentPositionParams: lsp.TextDocumentPositionParams{
					TextDocument: lsp.TextDocumentIdentifier{
						URI: uri,
					},
					Position: lsp.Position{
						Line:      tt.line,
						Character: tt.col,
					},
				},
			}

			var got lsp.SignatureHelp
			if err := tx.conn.Call(tx.ctx, "textDocument/signatureHelp", params, &got); err != nil {
				t.Fatal("conn.Call textDocument/signatureHelp:", err)
			}
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("unmatch (- want, + got):\n%s", diff)
			}
		})
	}
}
