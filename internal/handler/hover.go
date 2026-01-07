package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/sourcegraph/jsonrpc2"
	"github.com/sqls-server/sqls/ast"
	"github.com/sqls-server/sqls/ast/astutil"
	"github.com/sqls-server/sqls/internal/database"
	"github.com/sqls-server/sqls/internal/lsp"
	"github.com/sqls-server/sqls/parser"
	"github.com/sqls-server/sqls/parser/parseutil"
	"github.com/sqls-server/sqls/token"
)

var ErrNoHover = errors.New("no hover information found")

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

	res, err := hover(f.Text, params, s.worker.Cache())
	if err != nil {
		if errors.Is(ErrNoHover, err) {
			return nil, nil
		}
		return nil, err
	}
	return res, nil
}

func hover(text string, params lsp.HoverParams, dbCache *database.DBCache) (*lsp.Hover, error) {
	if dbCache == nil {
		return nil, nil
	}

	pos := token.Pos{
		Line: params.Position.Line,
		Col:  params.Position.Character + 1,
	}
	parsed, err := parser.Parse(text)
	if err != nil {
		return nil, err
	}

	// Find identifiers from focused statement
	nodeWalker := parseutil.NewNodeWalker(parsed, pos)
	hoverTargetMatcher := astutil.NodeMatcher{
		NodeTypes: []ast.NodeType{
			ast.TypeMemberIdentifier,
			ast.TypeIdentifier,
		},
	}
	focusedIdentNodes := nodeWalker.CurNodeMatches(hoverTargetMatcher)
	if len(focusedIdentNodes) == 0 {
		return nil, ErrNoHover
	}
	ident, memIdent := findIdent(focusedIdentNodes)

	// Collect environment
	hoverEnv, err := collectEnvironment(parsed, pos)
	if err != nil {
		return nil, err
	}

	// Check hover type
	ctx := getHoverTypes(nodeWalker, hoverEnv)

	// Create hover contents
	var hoverContent *lsp.MarkupContent
	if ident != nil && memIdent != nil {
		identName := ident.NoQuoteString()
		parentName := memIdent.ParentTok.NoQuoteString()
		childName := memIdent.ChildTok.NoQuoteString()
		switch identName {
		case parentName:
			// The cursor is on the member identifier parent.
			// example "w[o]rld.city"
			hoverContent = hoverContentFromParentIdent(ctx, identName, dbCache, hoverEnv)
		case childName:
			// The cursor is on the member identifier child.
			// example "world.c[i]ty"
			hoverContent = hoverContentFromChildIdent(ctx, identName, dbCache, hoverEnv)
		default:
			// Invalid
			hoverContent = nil
		}
	} else if ident == nil && memIdent != nil {
		// The cursor is on the dot with the member identifier
		// example "world[.]city"
		hoverContent = hoverContentFromChildIdent(ctx, memIdent.ChildTok.NoQuoteString(), dbCache, hoverEnv)
	} else if ident != nil && memIdent == nil {
		// The cursor is on the identifier
		// example "c[i]ty"
		hoverContent = hoverContentFromIdent(ctx, ident.NoQuoteString(), dbCache, hoverEnv)
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

type hoverEnvironment struct {
	aliases    []ast.Node
	tables     []*parseutil.TableInfo
	subQueries []*parseutil.SubQueryInfo
}

func (e *hoverEnvironment) getTableRealName(aliasName string) (string, bool) {
	for _, table := range e.tables {
		if table.Alias == aliasName {
			return table.Name, true
		}
	}
	return "", false
}

func (e *hoverEnvironment) getColumnRealName(aliasedName string) (string, bool) {
	for _, v := range e.aliases {
		alias, _ := v.(*ast.Aliased)

		if alias.AliasedName.String() == aliasedName {
			switch v := alias.RealName.(type) {
			case *ast.Identifier:
				return v.String(), true
			case *ast.MemberIdentifier:
				return v.Child.String(), true
			}
		}
	}
	return "", false
}

func (e *hoverEnvironment) getSubQueryView(name string) (*parseutil.SubQueryInfo, bool) {
	for _, subQuery := range e.subQueries {
		if subQuery.Name == name {
			return subQuery, true
		}
	}
	return nil, false
}

func (e *hoverEnvironment) getSubQueryViewOne() (*parseutil.SubQueryInfo, bool) {
	if len(e.subQueries) == 1 {
		return e.subQueries[0], true
	}
	return nil, false
}

func (e *hoverEnvironment) isSubQuery(name string) bool {
	_, ok := e.getSubQueryView(name)
	return ok
}

func collectEnvironment(parsed ast.TokenList, pos token.Pos) (*hoverEnvironment, error) {
	// Collect environment information
	aliases := parseutil.ExtractAliasedIdentifier(parsed)
	definedTables, err := parseutil.ExtractTable(parsed, pos)
	if err != nil {
		return nil, err
	}
	subQueries, err := parseutil.ExtractSubQueryViews(parsed, pos)
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

func findIdent(nodes []ast.Node) (*ast.Identifier, *ast.MemberIdentifier) {
	var (
		ident    *ast.Identifier
		memIdent *ast.MemberIdentifier
	)
	for _, node := range nodes {
		switch v := node.(type) {
		case *ast.Identifier:
			ident = v
		case *ast.MemberIdentifier:
			memIdent = v
		}
	}
	return ident, memIdent
}

func hoverContentFromIdent(ctx *hoverContext, identName string, dbCache *database.DBCache, hoverEnv *hoverEnvironment) *lsp.MarkupContent {
	if hoverTypeIs(ctx.types, hoverTypeColumn) {
		columnName := identName
		if realName, ok := hoverEnv.getColumnRealName(columnName); ok {
			columnName = realName
		}
		hoverContents := []*lsp.MarkupContent{}
		for _, table := range hoverEnv.tables {
			colDesc, ok := dbCache.Column(table.Name, columnName)
			if ok {
				hoverContents = append(
					hoverContents,
					columnHoverInfo(table.Name, columnName, colDesc),
				)
			}
		}
		if len(hoverContents) >= 2 {
			return nil
		}
		if len(hoverContents) == 1 {
			return hoverContents[0]
		}
	}
	if hoverTypeIs(ctx.types, hoverTypeTable) {
		// translate table alias
		tableName := identName
		for _, table := range hoverEnv.tables {
			if table.Alias == tableName {
				tableName = table.Name
			}
		}
		// find table
		cols, ok := dbCache.ColumnDescs(tableName)
		if ok {
			return tableHoverInfo(tableName, cols)
		}
	}
	if hoverTypeIs(ctx.types, hoverTypeSubQueryColumn) {
		columnName := identName
		subQueryView, ok := hoverEnv.getSubQueryViewOne()
		if !ok {
			return nil
		}
		return subqueryColumnHoverInfo(columnName, subQueryView, dbCache)
	}
	return nil
}

func hoverContentFromParentIdent(ctx *hoverContext, identName string, dbCache *database.DBCache, hoverEnv *hoverEnvironment) *lsp.MarkupContent {
	switch ctx.parent.Type {
	case parentTypeNone:
		return nil
	case parentTypeSchema:
	case parentTypeTable:
		tableName := identName
		realName, ok := hoverEnv.getTableRealName(tableName)
		if ok {
			tableName = realName
		}
		columns, ok := dbCache.ColumnDescs(tableName)
		if ok {
			return tableHoverInfo(tableName, columns)
		}
	case parentTypeSubQuery:
		subQueryName := identName
		subQueryView, ok := hoverEnv.getSubQueryView(subQueryName)
		if !ok {
			return nil
		}
		return subqueryHoverInfo(subQueryView, dbCache)
	}
	return nil
}

func hoverContentFromChildIdent(ctx *hoverContext, identName string, dbCache *database.DBCache, hoverEnv *hoverEnvironment) *lsp.MarkupContent {
	switch ctx.parent.Type {
	case parentTypeNone:
		return nil
	case parentTypeSchema:
		columns, ok := dbCache.ColumnDescs(identName)
		if ok {
			return tableHoverInfo(identName, columns)
		}
	case parentTypeTable:
		tableName := ctx.parent.Name
		realName, ok := hoverEnv.getTableRealName(tableName)
		if ok {
			tableName = realName
		}
		if colDesc, ok := dbCache.Column(tableName, identName); ok {
			return columnHoverInfo(tableName, identName, colDesc)
		}
		return nil
	case parentTypeSubQuery:
		subQueryView, ok := hoverEnv.getSubQueryView(ctx.parent.Name)
		if !ok {
			return nil
		}
		return subqueryColumnHoverInfo(identName, subQueryView, dbCache)
	}
	return nil
}

func columnHoverInfo(tableName, colName string, colDesc *database.ColumnDesc) *lsp.MarkupContent {
	return &lsp.MarkupContent{
		Kind:  lsp.Markdown,
		Value: database.ColumnDoc(tableName, colDesc),
	}
}

func tableHoverInfo(tableName string, cols []*database.ColumnDesc) *lsp.MarkupContent {
	return &lsp.MarkupContent{
		Kind:  lsp.Markdown,
		Value: database.TableDoc(tableName, cols),
	}
}

func subqueryHoverInfo(subQuery *parseutil.SubQueryInfo, dbCache *database.DBCache) *lsp.MarkupContent {
	return &lsp.MarkupContent{
		Kind:  lsp.Markdown,
		Value: database.SubqueryDoc(subQuery.Name, subQuery.Views, dbCache),
	}
}

func subqueryColumnHoverInfo(identName string, subQuery *parseutil.SubQueryInfo, dbCache *database.DBCache) *lsp.MarkupContent {
	return &lsp.MarkupContent{
		Kind:  lsp.Markdown,
		Value: database.SubqueryColumnDoc(identName, subQuery.Views, dbCache),
	}
}

func hoverTypeIs(hoverTypes []hoverType, expect hoverType) bool {
	for _, t := range hoverTypes {
		if t == expect {
			return true
		}
	}
	return false
}

type hoverType int

const (
	_ hoverType = iota
	hoverTypeKeyword
	hoverTypeFunction
	hoverTypeAlias
	hoverTypeColumn
	hoverTypeTable
	hoverTypeView
	hoverTypeSubQueryView
	hoverTypeSubQueryColumn
	hoverTypeChange
	hoverTypeUser
	hoverTypeSchema
)

type parentType int

const (
	parentTypeNone parentType = iota
	parentTypeSchema
	parentTypeTable
	parentTypeSubQuery
)

type hoverParent struct {
	Type parentType
	Name string
}

var noneParent = &hoverParent{Type: parentTypeNone}

type hoverContext struct {
	types  []hoverType
	parent *hoverParent
}

func getHoverTypes(nw *parseutil.NodeWalker, hoverEnv *hoverEnvironment) *hoverContext {
	memberIdentifierMatcher := astutil.NodeMatcher{
		NodeTypes: []ast.NodeType{ast.TypeMemberIdentifier},
	}
	syntaxPos := parseutil.CheckSyntaxPosition(nw)

	t := []hoverType{hoverTypeKeyword}
	p := noneParent
	switch {
	case syntaxPos == parseutil.ColName:
		if nw.CurNodeIs(memberIdentifierMatcher) {
			// has parent
			mi := nw.CurNodeTopMatched(memberIdentifierMatcher).(*ast.MemberIdentifier)
			t = []hoverType{
				hoverTypeColumn,
				hoverTypeSubQueryColumn,
				hoverTypeView,
				hoverTypeFunction,
			}
			name := mi.Parent.String()
			p = &hoverParent{
				Type: parentTypeTable,
				Name: name,
			}
			if hoverEnv.isSubQuery(name) {
				p = &hoverParent{
					Type: parentTypeSubQuery,
					Name: name,
				}
			}
		} else {
			t = []hoverType{
				hoverTypeColumn,
				hoverTypeTable,
				hoverTypeSubQueryColumn,
				hoverTypeSubQueryView,
				hoverTypeAlias,
				hoverTypeView,
				hoverTypeFunction,
				hoverTypeKeyword,
			}
		}
	case syntaxPos == parseutil.AliasName:
		// pass
	case syntaxPos == parseutil.SelectExpr || syntaxPos == parseutil.CaseValue:
		if nw.CurNodeIs(memberIdentifierMatcher) {
			// has parent
			mi := nw.CurNodeTopMatched(memberIdentifierMatcher).(*ast.MemberIdentifier)
			t = []hoverType{
				hoverTypeColumn,
				hoverTypeView,
				hoverTypeSubQueryColumn,
				hoverTypeFunction,
			}
			name := mi.Parent.String()
			p = &hoverParent{
				Type: parentTypeTable,
				Name: name,
			}
			if hoverEnv.isSubQuery(name) {
				p = &hoverParent{
					Type: parentTypeSubQuery,
					Name: name,
				}
			}
		} else {
			t = []hoverType{
				hoverTypeColumn,
				hoverTypeTable,
				hoverTypeAlias,
				hoverTypeView,
				hoverTypeSubQueryColumn,
				hoverTypeSubQueryView,
				hoverTypeFunction,
				hoverTypeKeyword,
			}
		}
	case syntaxPos == parseutil.TableReference:
		if nw.CurNodeIs(memberIdentifierMatcher) {
			// has parent
			mi := nw.CurNodeTopMatched(memberIdentifierMatcher).(*ast.MemberIdentifier)
			t = []hoverType{
				hoverTypeTable,
				hoverTypeView,
				hoverTypeSubQueryColumn,
				hoverTypeFunction,
			}
			p = &hoverParent{
				Type: parentTypeSchema,
				Name: mi.Parent.String(),
			}
		} else {
			t = []hoverType{
				hoverTypeColumn,
				hoverTypeTable,
				hoverTypeSchema,
				hoverTypeView,
				hoverTypeSubQueryColumn,
				hoverTypeSubQueryView,
				hoverTypeFunction,
				hoverTypeKeyword,
			}
		}
	case syntaxPos == parseutil.WhereCondition:
		if nw.CurNodeIs(memberIdentifierMatcher) {
			// has parent
			mi := nw.CurNodeTopMatched(memberIdentifierMatcher).(*ast.MemberIdentifier)
			t = []hoverType{
				hoverTypeTable,
				hoverTypeView,
				hoverTypeSubQueryColumn,
				hoverTypeFunction,
			}
			p = &hoverParent{
				Type: parentTypeTable,
				Name: mi.Parent.String(),
			}
		} else {
			t = []hoverType{
				hoverTypeColumn,
				hoverTypeTable,
				hoverTypeSchema,
				hoverTypeView,
				hoverTypeSubQueryColumn,
				hoverTypeSubQueryView,
				hoverTypeFunction,
				hoverTypeKeyword,
			}
		}
	case syntaxPos == parseutil.InsertColumn:
		t = []hoverType{
			hoverTypeColumn,
			hoverTypeTable,
			hoverTypeView,
		}
	default:
		// pass
	}
	return &hoverContext{
		types:  t,
		parent: p,
	}
}
