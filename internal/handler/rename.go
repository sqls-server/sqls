package handler

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/lighttiger2505/sqls/ast"
	"github.com/lighttiger2505/sqls/ast/astutil"
	"github.com/lighttiger2505/sqls/internal/lsp"
	"github.com/lighttiger2505/sqls/parser"
	"github.com/lighttiger2505/sqls/parser/parseutil"
	"github.com/lighttiger2505/sqls/token"
	"github.com/sourcegraph/jsonrpc2"
)

func (s *Server) handleTextDocumentRename(ctx context.Context, conn *jsonrpc2.Conn, req *jsonrpc2.Request) (result interface{}, err error) {
	if req.Params == nil {
		return nil, &jsonrpc2.Error{Code: jsonrpc2.CodeInvalidParams}
	}

	var params lsp.RenameParams
	if err := json.Unmarshal(*req.Params, &params); err != nil {
		return nil, err
	}

	f, ok := s.files[params.TextDocument.URI]
	if !ok {
		return nil, fmt.Errorf("document not found: %s", params.TextDocument.URI)
	}

	res, err := rename(f.Text, params)
	if err != nil {
		return nil, err
	}
	return res, nil
}

func rename(text string, params lsp.RenameParams) (*lsp.WorkspaceEdit, error) {
	parsed, err := parser.Parse(text)
	if err != nil {
		return nil, err
	}

	pos := token.Pos{
		Line: params.Position.Line,
		Col:  params.Position.Character,
	}

	// Get the identifer on focus
	nodeWalker := parseutil.NewNodeWalker(parsed, pos)
	m := astutil.NodeMatcher{
		NodeTypes: []ast.NodeType{ast.TypeIdentifer},
	}
	currentVariable := nodeWalker.CurNodeButtomMatched(m)
	if currentVariable == nil {
		return nil, nil
	}

	// Get all identifiers in the statement
	idents, err := parseutil.ExtractIdenfiers(parsed, pos)
	if err != nil {
		return nil, err
	}

	// Extract only those with matching names
	renameTarget := []ast.Node{}
	for _, ident := range idents {
		if ident.String() == currentVariable.String() {
			renameTarget = append(renameTarget, ident)
		}
	}
	if len(renameTarget) == 0 {
		return nil, nil
	}

	edits := make([]lsp.TextEdit, len(renameTarget))
	for i, target := range renameTarget {
		edit := lsp.TextEdit{
			Range: lsp.Range{
				Start: lsp.Position{
					Line:      target.Pos().Line,
					Character: target.Pos().Col,
				},
				End: lsp.Position{
					Line:      target.End().Line,
					Character: target.End().Col,
				},
			},
			NewText: params.NewName,
		}
		edits[i] = edit
	}

	res := &lsp.WorkspaceEdit{
		DocumentChanges: []lsp.TextDocumentEdit{
			{
				TextDocument: lsp.OptionalVersionedTextDocumentIdentifier{
					Version: 0,
					TextDocumentIdentifier: lsp.TextDocumentIdentifier{
						URI: params.TextDocument.URI,
					},
				},
				Edits: edits,
			},
		},
	}

	return res, nil
}
