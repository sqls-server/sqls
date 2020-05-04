package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/lighttiger2505/sqls/ast"
	"github.com/lighttiger2505/sqls/ast/astutil"
	"github.com/lighttiger2505/sqls/dialect"
	"github.com/lighttiger2505/sqls/internal/completer"
	"github.com/lighttiger2505/sqls/internal/database"
	"github.com/lighttiger2505/sqls/internal/lsp"
	"github.com/lighttiger2505/sqls/parser"
	"github.com/lighttiger2505/sqls/parser/parseutil"
	"github.com/lighttiger2505/sqls/token"
	"github.com/sourcegraph/jsonrpc2"
	"golang.org/x/xerrors"
)

var ErrNoHover = errors.New("no hover infomation found")

func (s *Server) handleTextDocumentHover(ctx context.Context, conn *jsonrpc2.Conn, req *jsonrpc2.Request) (result interface{}, err error) {
	if req.Params == nil {
		return nil, &jsonrpc2.Error{Code: jsonrpc2.CodeInvalidParams}
	}

	var params lsp.HoverParams
	if err := json.Unmarshal(*req.Params, &params); err != nil {
		return nil, err
	}

	f, ok := s.files[params.TextDocument.URI]
	if !ok {
		return nil, fmt.Errorf("document not found: %s", params.TextDocument.URI)
	}

	res, err := hover(f.Text, params, s.completer.DBInfo)
	if err != nil {
		if err == ErrNoHover {
			return nil, nil
		}
		return nil, err
	}
	return res, nil
}

var hoverTargetMatcher = astutil.NodeMatcher{
	NodeTypes: []ast.NodeType{
		ast.TypeMemberIdentifer,
		ast.TypeIdentifer,
	},
}

func hover(text string, params lsp.HoverParams, dbInfo *completer.DatabaseInfo) (*lsp.Hover, error) {
	pos := token.Pos{
		Line: params.Position.Line + 1,
		Col:  params.Position.Character + 1,
	}
	parsed, err := parse(text)
	if err != nil {
		return nil, err
	}

	definedTables, err := parseutil.ExtractTable(parsed, pos)
	if err != nil {
		return nil, err
	}

	nodeWalker := parseutil.NewNodeWalker(parsed, pos)
	curNode := nodeWalker.CurNodeButtomMatched(hoverTargetMatcher)

	if curNode == nil {
		return nil, ErrNoHover
	}

	var hoverContent *lsp.MarkupContent
	switch v := curNode.(type) {
	case *ast.Identifer:
		identName := v.String()

		// find column
		hoverContents := []string{}
		for _, table := range definedTables {
			columnInfo, ok := dbInfo.Column(table.Name, identName)
			if ok {
				buf := new(bytes.Buffer)
				fmt.Fprintf(buf, "%s.%s column", table.Name, identName)
				fmt.Fprintln(buf)
				fmt.Fprintln(buf)
				fmt.Fprintln(buf, toHoverTextColumn(columnInfo))
				hoverContents = append(hoverContents, buf.String())
			}
		}
		if len(hoverContents) >= 2 {
			return nil, ErrNoHover
		}
		if len(hoverContents) == 1 {
			hoverContent = &lsp.MarkupContent{
				Kind:  lsp.Markdown,
				Value: hoverContents[0],
			}
		}

		// translate table alias
		tableIdent := identName
		for _, table := range definedTables {
			if table.Alias == tableIdent {
				tableIdent = table.Name
			}
		}
		// find table
		columns, ok := dbInfo.ColumnDescs(tableIdent)
		if ok {
			buf := new(bytes.Buffer)
			fmt.Fprintf(buf, "%s table", tableIdent)
			fmt.Fprintln(buf)
			fmt.Fprintln(buf)
			for _, col := range columns {
				fmt.Fprintf(buf, "- %s", toHoverTextTable(col))
				fmt.Fprintln(buf)
			}
			hoverContent = &lsp.MarkupContent{
				Kind:  lsp.PlainText,
				Value: buf.String(),
			}
		}
	default:
		return nil, xerrors.Errorf("unknown node type %T", v)
	}

	if hoverContent == nil {
		return nil, ErrNoHover
	}

	res := &lsp.Hover{
		Contents: *hoverContent,
		Range: lsp.Range{
			Start: lsp.Position{
				Line:      curNode.Pos().Line,
				Character: curNode.Pos().Col,
			},
			End: lsp.Position{
				Line:      curNode.End().Line,
				Character: curNode.End().Col,
			},
		},
	}
	return res, nil
}

func toHoverTextTable(desc *database.ColumnDesc) string {
	return fmt.Sprintf(
		"%s: %s %s %s",
		desc.Name,
		desc.Type,
		desc.Key,
		desc.Extra,
	)
}

func toHoverTextColumn(desc *database.ColumnDesc) string {
	return fmt.Sprintf(
		"%s %s %s",
		desc.Type,
		desc.Key,
		desc.Extra,
	)
}

func parse(text string) (ast.TokenList, error) {
	src := bytes.NewBuffer([]byte(text))
	p, err := parser.NewParser(src, &dialect.GenericSQLDialect{})
	if err != nil {
		return nil, err
	}
	parsed, err := p.Parse()
	if err != nil {
		return nil, err
	}
	return parsed, nil
}
