package handler

import (
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/sqls-server/sqls/internal/config"
	"github.com/sqls-server/sqls/internal/database"
	"github.com/sqls-server/sqls/internal/lsp"
)

type signatureHelpTestCase struct {
	name  string
	input string
	line  int
	col   int
	want  lsp.SignatureHelp
}

var signatureHelpTestCases = []signatureHelpTestCase{
	// single record
	// input is "insert into city (ID, Name, CountryCode) VALUES (123,  NULL, '2020')"
	genSingleRecordInsertTest(50, 0),
	genSingleRecordInsertTest(52, 0),
	genSingleRecordInsertTest(53, 1),
	genSingleRecordInsertTest(59, 1),
	genSingleRecordInsertTest(60, 2),
	genSingleRecordInsertTest(67, 2),

	// multi record
	// input is "insert into city (ID, Name, CountryCode) VALUES (123, 'aaa', '2020'), (456, 'bbb', '2021')"
	genMultiRecordInsertTest(50, 0),
	genMultiRecordInsertTest(52, 0),
	genMultiRecordInsertTest(53, 1),
	genMultiRecordInsertTest(59, 1),
	genMultiRecordInsertTest(60, 2),
	genMultiRecordInsertTest(67, 2),

	genMultiRecordInsertTest(72, 0),
	genMultiRecordInsertTest(74, 0),
	genMultiRecordInsertTest(76, 1),
	genMultiRecordInsertTest(81, 1),
	genMultiRecordInsertTest(83, 2),
	genMultiRecordInsertTest(89, 2),
}

func genSingleRecordInsertTest(col int, wantActiveParameter int) signatureHelpTestCase {
	return signatureHelpTestCase{
		name:  fmt.Sprintf("single record %d-%d", col, wantActiveParameter),
		input: "insert into city (ID, Name, CountryCode) VALUES (123,  NULL, '2020')",
		line:  0,
		col:   col,
		want: lsp.SignatureHelp{
			Signatures: []lsp.SignatureInformation{
				{
					Label:         "city (ID, Name, CountryCode)",
					Documentation: "city table columns",
					Parameters: []lsp.ParameterInformation{
						{
							Label:         "ID",
							Documentation: "`int(11)` PRI auto_increment",
						},
						{
							Label:         "Name",
							Documentation: "`char(35)`",
						},
						{
							Label:         "CountryCode",
							Documentation: "`char(3)` MUL",
						},
					},
				},
			},
			ActiveSignature: 0.0,
			ActiveParameter: float64(wantActiveParameter),
		},
	}
}

func genMultiRecordInsertTest(col int, wantActiveParameter int) signatureHelpTestCase {
	return signatureHelpTestCase{
		name:  fmt.Sprintf("multi record %d-%d", col, wantActiveParameter),
		input: "insert into city (ID, Name, CountryCode) VALUES (123, 'aaa', '2020'), (456, 'bbb', '2021')",
		line:  0,
		col:   col,
		want: lsp.SignatureHelp{
			Signatures: []lsp.SignatureInformation{
				{
					Label:         "city (ID, Name, CountryCode)",
					Documentation: "city table columns",
					Parameters: []lsp.ParameterInformation{
						{
							Label:         "ID",
							Documentation: "`int(11)` PRI auto_increment",
						},
						{
							Label:         "Name",
							Documentation: "`char(35)`",
						},
						{
							Label:         "CountryCode",
							Documentation: "`char(3)` MUL",
						},
					},
				},
			},
			ActiveSignature: 0.0,
			ActiveParameter: float64(wantActiveParameter),
		},
	}
}

func TestSignatureHelpMain(t *testing.T) {
	tx := newTestContext()
	tx.initServer(t)
	defer tx.tearDown()

	cfg := &config.Config{
		Connections: []*database.DBConfig{
			{Driver: "mock"},
		},
	}
	tx.addWorkspaceConfig(t, cfg)

	for _, tt := range signatureHelpTestCases {
		t.Run(tt.name, func(t *testing.T) {
			tx.textDocumentDidOpen(t, testFileURI, tt.input)

			params := lsp.SignatureHelpParams{
				TextDocumentPositionParams: lsp.TextDocumentPositionParams{
					TextDocument: lsp.TextDocumentIdentifier{
						URI: testFileURI,
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

func TestSignatureHelpNoneDBConnection(t *testing.T) {
	tx := newTestContext()
	tx.initServer(t)
	defer tx.tearDown()

	cfg := &config.Config{
		Connections: []*database.DBConfig{},
	}
	tx.addWorkspaceConfig(t, cfg)

	uri := "file:///Users/octref/Code/css-test/test.sql"
	for _, tt := range signatureHelpTestCases {
		t.Run(tt.name, func(t *testing.T) {
			tx.textDocumentDidOpen(t, testFileURI, tt.input)

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
			// Without a DB connection, it is not possible to provide functions using the DB connection, so just make sure that no errors occur.
			var got lsp.SignatureHelp
			if err := tx.conn.Call(tx.ctx, "textDocument/signatureHelp", params, &got); err != nil {
				t.Fatal("conn.Call textDocument/signatureHelp:", err)
			}
		})
	}
}
