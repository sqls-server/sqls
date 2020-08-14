package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/lighttiger2505/sqls/internal/completer"
	"github.com/lighttiger2505/sqls/internal/lsp"
	"github.com/sourcegraph/jsonrpc2"
)

func (s *Server) handleTextDocumentCompletion(ctx context.Context, conn *jsonrpc2.Conn, req *jsonrpc2.Request) (result interface{}, err error) {
	if req.Params == nil {
		return nil, &jsonrpc2.Error{Code: jsonrpc2.CodeInvalidParams}
	}

	var params lsp.CompletionParams
	if err := json.Unmarshal(*req.Params, &params); err != nil {
		return nil, err
	}

	f, ok := s.files[params.TextDocument.URI]
	if !ok {
		return nil, fmt.Errorf("document not found: %s", params.TextDocument.URI)
	}

	log.Println(s.worker.Cache())
	if s.worker.Cache() == nil {
		return nil, fmt.Errorf("database cache not found")
	}
	c := completer.NewCompleter(s.worker.Cache())
	completionItems, err := c.Complete(f.Text, params)
	if err != nil {
		return nil, err
	}
	return completionItems, nil
}
