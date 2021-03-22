package handler

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/lighttiger2505/sqls/internal/config"
	"github.com/lighttiger2505/sqls/internal/database"
	"github.com/lighttiger2505/sqls/internal/lsp"
)

var definitionTestCases = []struct {
	name   string
	input  string
	output lsp.Definition
	pos    lsp.Position
}{
	{
		name:  "",
		input: "SELECT ci.ID, ci.Name FROM city as ci",
		output: []lsp.Location{
			{
				URI: testFileURI,
				Range: lsp.Range{
					Start: lsp.Position{
						Line:      0,
						Character: 35,
					},
					End: lsp.Position{
						Line:      0,
						Character: 37,
					},
				},
			},
		},
		pos: lsp.Position{
			Line:      0,
			Character: 8,
		},
	},
}

func TestDefinition(t *testing.T) {
	tx := newTestContext()
	tx.setup(t)
	defer tx.tearDown()

	cfg := &config.Config{
		Connections: []*database.DBConfig{
			{Driver: "mock"},
		},
	}
	tx.addWorkspaceConfig(t, cfg)

	for _, tt := range definitionTestCases {
		t.Run(tt.name, func(t *testing.T) {
			tx.textDocumentDidOpen(t, testFileURI, tt.input)

			params := lsp.DefinitionParams{
				TextDocumentPositionParams: lsp.TextDocumentPositionParams{
					TextDocument: lsp.TextDocumentIdentifier{
						URI: testFileURI,
					},
					Position: tt.pos,
				},
			}
			var got lsp.Definition
			err := tx.conn.Call(tx.ctx, "textDocument/definition", params, &got)
			if err != nil {
				t.Errorf("conn.Call textDocument/definition: %+v", err)
				return
			}

			if diff := cmp.Diff(tt.output, got); diff != "" {
				t.Errorf("unmatch hover contents (- want, + got):\n%s", diff)
			}
		})
	}
}

func TestTypeDefinition(t *testing.T) {
	tx := newTestContext()
	tx.setup(t)
	defer tx.tearDown()

	cfg := &config.Config{
		Connections: []*database.DBConfig{
			{Driver: "mock"},
		},
	}
	tx.addWorkspaceConfig(t, cfg)

	for _, tt := range definitionTestCases {
		t.Run(tt.name, func(t *testing.T) {
			tx.textDocumentDidOpen(t, testFileURI, tt.input)

			params := lsp.DefinitionParams{
				TextDocumentPositionParams: lsp.TextDocumentPositionParams{
					TextDocument: lsp.TextDocumentIdentifier{
						URI: testFileURI,
					},
					Position: tt.pos,
				},
			}
			var got lsp.Definition
			err := tx.conn.Call(tx.ctx, "textDocument/typeDefinition", params, &got)
			if err != nil {
				t.Errorf("conn.Call textDocument/definition: %+v", err)
				return
			}

			if diff := cmp.Diff(tt.output, got); diff != "" {
				t.Errorf("unmatch hover contents (- want, + got):\n%s", diff)
			}
		})
	}
}
