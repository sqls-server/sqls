package handler

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/sqls-server/sqls/internal/config"
	"github.com/sqls-server/sqls/internal/database"
	"github.com/sqls-server/sqls/internal/lsp"
)

var renameTestCases = []struct {
	name    string
	input   string
	newName string
	output  lsp.WorkspaceEdit
	pos     lsp.Position
}{
	{
		name:    "subquery",
		input:   "SELECT it.ID, it.Name FROM (SELECT ci.ID, ci.Name, ci.CountryCode, ci.District, ci.Population FROM city AS ci) as it",
		newName: "ct",
		output: lsp.WorkspaceEdit{
			DocumentChanges: []lsp.TextDocumentEdit{
				{
					TextDocument: lsp.OptionalVersionedTextDocumentIdentifier{
						Version: 0,
						TextDocumentIdentifier: lsp.TextDocumentIdentifier{
							URI: "file:///Users/octref/Code/css-test/test.sql",
						},
					},
					Edits: []lsp.TextEdit{
						{
							Range: lsp.Range{
								Start: lsp.Position{
									Line:      0,
									Character: 7,
								},
								End: lsp.Position{
									Line:      0,
									Character: 9,
								},
							},
							NewText: "ct",
						},
						{
							Range: lsp.Range{
								Start: lsp.Position{
									Line:      0,
									Character: 14,
								},
								End: lsp.Position{
									Line:      0,
									Character: 16,
								},
							},
							NewText: "ct",
						},
						{
							Range: lsp.Range{
								Start: lsp.Position{
									Line:      0,
									Character: 114,
								},
								End: lsp.Position{
									Line:      0,
									Character: 116,
								},
							},
							NewText: "ct",
						},
					},
				},
			},
		},
		pos: lsp.Position{
			Line:      0,
			Character: 8,
		},
	},
	{
		name:    "ok",
		input:   "SELECT ci.ID, ci.Name FROM city as ci",
		newName: "ct",
		output: lsp.WorkspaceEdit{
			DocumentChanges: []lsp.TextDocumentEdit{
				{
					TextDocument: lsp.OptionalVersionedTextDocumentIdentifier{
						Version: 0,
						TextDocumentIdentifier: lsp.TextDocumentIdentifier{
							URI: "file:///Users/octref/Code/css-test/test.sql",
						},
					},
					Edits: []lsp.TextEdit{
						{
							Range: lsp.Range{
								Start: lsp.Position{
									Line:      0,
									Character: 7,
								},
								End: lsp.Position{
									Line:      0,
									Character: 9,
								},
							},
							NewText: "ct",
						},
						{
							Range: lsp.Range{
								Start: lsp.Position{
									Line:      0,
									Character: 14,
								},
								End: lsp.Position{
									Line:      0,
									Character: 16,
								},
							},
							NewText: "ct",
						},
						{
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
							NewText: "ct",
						},
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

func TestRenameMain(t *testing.T) {
	tx := newTestContext()
	tx.setup(t)
	defer tx.tearDown()

	cfg := &config.Config{
		Connections: []*database.DBConfig{
			{Driver: "mock"},
		},
	}
	tx.addWorkspaceConfig(t, cfg)

	for _, tt := range renameTestCases {
		t.Run(tt.name, func(t *testing.T) {
			tx.textDocumentDidOpen(t, testFileURI, tt.input)

			params := lsp.RenameParams{
				TextDocument: lsp.TextDocumentIdentifier{
					URI: testFileURI,
				},
				Position: tt.pos,
				NewName:  tt.newName,
			}
			var got lsp.WorkspaceEdit
			err := tx.conn.Call(tx.ctx, "textDocument/rename", params, &got)
			if err != nil {
				t.Errorf("conn.Call textDocument/rename: %+v", err)
				return
			}

			if diff := cmp.Diff(tt.output, got); diff != "" {
				t.Errorf("unmatch rename edits (- want, + got):\n%s", diff)
			}
		})
	}
}
