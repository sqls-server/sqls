package handler

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/sourcegraph/jsonrpc2"
	"github.com/sqls-server/sqls/internal/completer"
	"github.com/sqls-server/sqls/internal/lsp"
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

	c := completer.NewCompleter(s.worker.Cache())
	if s.dbConn != nil {
		c.Driver = s.dbConn.Driver
	} else {
		c.Driver = ""
	}
	completionItems, err := c.Complete(f.Text, params, s.getConfig().LowercaseKeywords)
	if err != nil {
		return nil, err
	}
	return completionItems, nil
}
