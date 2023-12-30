package handler

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/sourcegraph/jsonrpc2"
	"github.com/sqls-server/sqls/internal/database"
	"github.com/sqls-server/sqls/internal/lsp"
	"github.com/sqls-server/sqls/parser"
	"github.com/sqls-server/sqls/parser/parseutil"
	"github.com/sqls-server/sqls/token"
)

func (s *Server) handleTextDocumentSignatureHelp(ctx context.Context, conn *jsonrpc2.Conn, req *jsonrpc2.Request) (result interface{}, err error) {
	if req.Params == nil {
		return nil, &jsonrpc2.Error{Code: jsonrpc2.CodeInvalidParams}
	}

	var params lsp.SignatureHelpParams
	if err := json.Unmarshal(*req.Params, &params); err != nil {
		return nil, err
	}

	f, ok := s.files[params.TextDocument.URI]
	if !ok {
		return nil, fmt.Errorf("document not found: %s", params.TextDocument.URI)
	}

	res, err := SignatureHelp(f.Text, params, s.worker.Cache())
	if err != nil {
		return nil, err
	}
	return res, nil
}

func SignatureHelp(text string, params lsp.SignatureHelpParams, dbCache *database.DBCache) (*lsp.SignatureHelp, error) {
	if dbCache == nil {
		return nil, nil
	}

	parsed, err := parser.Parse(text)
	if err != nil {
		return nil, err
	}

	pos := token.Pos{
		Line: params.Position.Line,
		Col:  params.Position.Character,
	}
	nodeWalker := parseutil.NewNodeWalker(parsed, pos)
	types := getSignatureHelpTypes(nodeWalker)

	switch {
	case signatureHelpIs(types, SignatureHelpTypeInsertValue):
		insert, err := parseutil.ExtractInsert(parsed, pos)
		if err != nil {
			return nil, err
		}
		if !insert.Enable() {
			return nil, err
		}

		table := insert.GetTable()
		cols := insert.GetColumns()
		paramIdx := insert.GetValues().GetIndex(pos)
		tableName := table.Name

		params := []lsp.ParameterInformation{}
		for _, col := range cols.GetIdentifiers() {
			colName := col.String()
			colDoc := ""
			colDesc, ok := dbCache.Column(tableName, colName)
			if ok {
				colDoc = colDesc.OnelineDesc()
			}
			p := lsp.ParameterInformation{
				Label:         colName,
				Documentation: colDoc,
			}
			params = append(params, p)
		}

		signatureLabel := fmt.Sprintf("%s (%s)", tableName, cols.String())
		sh := &lsp.SignatureHelp{
			Signatures: []lsp.SignatureInformation{
				{
					Label:         signatureLabel,
					Documentation: fmt.Sprintf("%s table columns", tableName),
					Parameters:    params,
				},
			},
			ActiveSignature: 0.0,
			ActiveParameter: float64(paramIdx),
		}
		return sh, nil
	default:
		// pass
		return nil, nil
	}
}

type signatureHelpType int

const (
	_ signatureHelpType = iota
	SignatureHelpTypeInsertValue
	SignatureHelpTypeUnknown = 99
)

func (sht signatureHelpType) String() string {
	switch sht {
	case SignatureHelpTypeInsertValue:
		return "InsertValue"
	default:
		return ""
	}
}

func getSignatureHelpTypes(nw *parseutil.NodeWalker) []signatureHelpType {
	syntaxPos := parseutil.CheckSyntaxPosition(nw)
	types := []signatureHelpType{}
	switch {
	case syntaxPos == parseutil.InsertValue:
		types = []signatureHelpType{
			SignatureHelpTypeInsertValue,
		}
	default:
		// pass
	}
	return types
}

func signatureHelpIs(types []signatureHelpType, expect signatureHelpType) bool {
	for _, t := range types {
		if t == expect {
			return true
		}
	}
	return false
}
