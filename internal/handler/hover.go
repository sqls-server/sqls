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
	"github.com/lighttiger2505/sqls/internal/database"
	"github.com/lighttiger2505/sqls/internal/lsp"
	"github.com/lighttiger2505/sqls/parser"
	"github.com/lighttiger2505/sqls/parser/parseutil"
	"github.com/lighttiger2505/sqls/token"
	"github.com/sourcegraph/jsonrpc2"
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

	res, err := hover(f.Text, params, s.dbCache)
	if err != nil {
		if err == ErrNoHover {
			return nil, nil
		}
		return nil, err
	}
	return res, nil
}

func hover(text string, params lsp.HoverParams, dbCache *database.DatabaseCache) (*lsp.Hover, error) {
	pos := token.Pos{
		Line: params.Position.Line,
		Col:  params.Position.Character + 1,
	}
	parsed, err := parse(text)
	if err != nil {
		return nil, err
	}

	// Find identifiers from focused statement
	nodeWalker := parseutil.NewNodeWalker(parsed, pos)
	focusedIdentNodes := nodeWalker.CurNodeMatches(hoverTargetMatcher)
	if len(focusedIdentNodes) == 0 {
		return nil, ErrNoHover
	}
	ident, memIdent := findIdent(focusedIdentNodes)

	// Collect environment
	hoverEnv, err := collectEnvirontment(parsed, pos)
	if err != nil {
		return nil, err
	}

	// Create hover contents
	var hoverContent *lsp.MarkupContent
	if ident != nil && memIdent != nil {
		hoverContent = hoverContentFromMemberIdent(ident, memIdent, dbCache, hoverEnv)
	} else if ident == nil && memIdent != nil {
		hoverContent = hoverContentFromMemberIdentOnly(memIdent, dbCache, hoverEnv)
	} else if ident != nil && memIdent == nil {
		hoverContent = hoverContentFromIdent(ident, dbCache, hoverEnv)
	}
	if hoverContent == nil {
		return nil, ErrNoHover
	}

	var posIdent ast.Node
	posIdent = ident
	if ident == nil && memIdent != nil {
		posIdent = memIdent
	}
	res := &lsp.Hover{
		Contents: *hoverContent,
		Range: lsp.Range{
			Start: lsp.Position{
				Line:      posIdent.Pos().Line,
				Character: posIdent.Pos().Col,
			},
			End: lsp.Position{
				Line:      posIdent.End().Line,
				Character: posIdent.End().Col,
			},
		},
	}
	return res, nil
}

var hoverTargetMatcher = astutil.NodeMatcher{
	NodeTypes: []ast.NodeType{
		ast.TypeMemberIdentifer,
		ast.TypeIdentifer,
	},
}

type hoverEnvironment struct {
	aliases    []ast.Node
	tables     []*parseutil.TableInfo
	subQueries *parseutil.SubQueryInfo
}

func collectEnvirontment(parsed ast.TokenList, pos token.Pos) (*hoverEnvironment, error) {
	// Collect environment infomation
	aliases := parseutil.ExtractAliasedIdentifer(parsed)
	definedTables, err := parseutil.ExtractTable(parsed, pos)
	if err != nil {
		return nil, err
	}
	subQueries, err := parseutil.ExtractSubQueryView(parsed, pos)
	if err != nil {
		return nil, err
	}
	environment := &hoverEnvironment{
		aliases:    aliases,
		tables:     definedTables,
		subQueries: subQueries,
	}
	return environment, nil
}

func findIdent(nodes []ast.Node) (*ast.Identifer, *ast.MemberIdentifer) {
	var (
		ident    *ast.Identifer
		memIdent *ast.MemberIdentifer
	)
	for _, node := range nodes {
		switch v := node.(type) {
		case *ast.Identifer:
			ident = v
		case *ast.MemberIdentifer:
			memIdent = v
		}
	}
	return ident, memIdent
}

func hoverContentFromIdent(ident *ast.Identifer, dbCache *database.DatabaseCache, hoverEnv *hoverEnvironment) *lsp.MarkupContent {
	identName := ident.String()
	// find column
	hoverContents := []string{}
	for _, table := range hoverEnv.tables {
		columnInfo, ok := dbCache.Column(table.Name, identName)
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
		return nil
	}
	if len(hoverContents) == 1 {
		return &lsp.MarkupContent{
			Kind:  lsp.Markdown,
			Value: hoverContents[0],
		}
	}

	// translate table alias
	tableIdent := identName
	for _, table := range hoverEnv.tables {
		if table.Alias == tableIdent {
			tableIdent = table.Name
		}
	}
	// find table
	columns, ok := dbCache.ColumnDescs(tableIdent)
	if ok {
		buf := new(bytes.Buffer)
		fmt.Fprintf(buf, "%s table", tableIdent)
		fmt.Fprintln(buf)
		fmt.Fprintln(buf)
		for _, col := range columns {
			fmt.Fprintf(buf, "- %s", toHoverTextTable(col))
			fmt.Fprintln(buf)
		}
		return &lsp.MarkupContent{
			Kind:  lsp.Markdown,
			Value: buf.String(),
		}
	}
	return nil
}

func hoverContentFromMemberIdent(ident *ast.Identifer, memIdent *ast.MemberIdentifer, dbCache *database.DatabaseCache, hoverEnv *hoverEnvironment) *lsp.MarkupContent {
	// TODO ADD FROM case
	identName := ident.String()
	tableName := memIdent.Parent.String()
	colName := memIdent.Child.String()

	// translate to table alias
	for _, table := range hoverEnv.tables {
		if table.Alias == tableName {
			tableName = table.Name
		}
	}

	// find column
	if identName == colName {
		if columnInfo, ok := dbCache.Column(tableName, colName); ok {
			buf := new(bytes.Buffer)
			fmt.Fprintf(buf, "%s.%s column", tableName, colName)
			fmt.Fprintln(buf)
			fmt.Fprintln(buf)
			fmt.Fprintln(buf, toHoverTextColumn(columnInfo))
			return &lsp.MarkupContent{
				Kind:  lsp.Markdown,
				Value: buf.String(),
			}
		}
	}

	// find table
	columns, ok := dbCache.ColumnDescs(tableName)
	if ok {
		buf := new(bytes.Buffer)
		fmt.Fprintf(buf, "%s table", tableName)
		fmt.Fprintln(buf)
		fmt.Fprintln(buf)
		for _, col := range columns {
			fmt.Fprintf(buf, "- %s", toHoverTextTable(col))
			fmt.Fprintln(buf)
		}
		return &lsp.MarkupContent{
			Kind:  lsp.Markdown,
			Value: buf.String(),
		}
	}

	return nil
}

func hoverContentFromMemberIdentOnly(memIdent *ast.MemberIdentifer, dbCache *database.DatabaseCache, hoverEnv *hoverEnvironment) *lsp.MarkupContent {
	tableName := memIdent.Parent.String()
	colName := memIdent.Child.String()

	// translate to table alias
	for _, table := range hoverEnv.tables {
		if table.Alias == tableName {
			tableName = table.Name
		}
	}

	// find column
	if columnInfo, ok := dbCache.Column(tableName, colName); ok {
		buf := new(bytes.Buffer)
		fmt.Fprintf(buf, "%s.%s column", tableName, colName)
		fmt.Fprintln(buf)
		fmt.Fprintln(buf)
		fmt.Fprintln(buf, toHoverTextColumn(columnInfo))
		return &lsp.MarkupContent{
			Kind:  lsp.Markdown,
			Value: buf.String(),
		}
	}
	return nil
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
