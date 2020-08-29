package handler

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/lighttiger2505/sqls/internal/formatter"
	"github.com/lighttiger2505/sqls/internal/lsp"
	"github.com/sourcegraph/jsonrpc2"
)

func (s *Server) handleTextDocumentFormatting(ctx context.Context, conn *jsonrpc2.Conn, req *jsonrpc2.Request) (result interface{}, err error) {
	if req.Params == nil {
		return nil, &jsonrpc2.Error{Code: jsonrpc2.CodeInvalidParams}
	}

	var params lsp.DocumentFormattingParams
	if err := json.Unmarshal(*req.Params, &params); err != nil {
		return nil, err
	}

	f, ok := s.files[params.TextDocument.URI]
	if !ok {
		return nil, fmt.Errorf("document not found: %s", params.TextDocument.URI)
	}

	textEdits, err := formatter.Format(f.Text, params)
	if err != nil {
		return nil, err
	}
	if len(textEdits) > 0 {
		return textEdits, nil
	}
	return nil, nil
}

func (s *Server) handleTextDocumentRangeFormatting(ctx context.Context, conn *jsonrpc2.Conn, req *jsonrpc2.Request) (result interface{}, err error) {
	if req.Params == nil {
		return nil, &jsonrpc2.Error{Code: jsonrpc2.CodeInvalidParams}
	}

	var params lsp.DocumentRangeFormattingParams
	if err := json.Unmarshal(*req.Params, &params); err != nil {
		return nil, err
	}

	_, ok := s.files[params.TextDocument.URI]
	if !ok {
		return nil, fmt.Errorf("document not found: %s", params.TextDocument.URI)
	}

	textEdits := []lsp.TextEdit{}
	if len(textEdits) > 0 {
		return textEdits, nil
	}
	return nil, nil
}

// func formatting(text string, params lsp.DocumentFormattingParams) ([]lsp.TextEdit, error) {
// 	parsed, err := parser.Parse(text)
// 	if err != nil {
// 		return nil, err
// 	}
//
// 	st := lsp.Position{
// 		Line:      parsed.Pos().Line,
// 		Character: parsed.Pos().Col,
// 	}
// 	en := lsp.Position{
// 		Line:      parsed.End().Line,
// 		Character: parsed.End().Col,
// 	}
//
// 	formatted := formattingProcess(astutil.NewNodeReader(parsed))
// 	res := []lsp.TextEdit{
// 		{
// 			Range: lsp.Range{
// 				Start: st,
// 				End:   en,
// 			},
// 			NewText: formatted.String(),
// 		},
// 	}
// 	return res, nil
// }
//
// type formatEnvironment struct {
// 	indentLevel int
// }
//
// type additionalFormatter func(nodes []ast.Node, env formatEnvironment) ([]ast.Node, formatEnvironment)
//
// func unshift(slice []ast.Node, node ast.Node) []ast.Node {
// 	return append([]ast.Node{node}, slice...)
// }
//
// var whitespaceNode = ast.NewItem(&token.Token{
// 	Kind:  token.Whitespace,
// 	Value: " ",
// })
//
// var linebreakNode = ast.NewItem(&token.Token{
// 	Kind:  token.Whitespace,
// 	Value: "\n",
// })
//
// var indentNode = ast.NewItem(&token.Token{
// 	Kind:  token.Whitespace,
// 	Value: "\t",
// })
//
// var afterWhiteSpaceMatcher = astutil.NodeMatcher{
// 	ExpectTokens: []token.Kind{
// 		token.SQLKeyword,
// 	},
// }
//
// func addAfterWhiteSpace(nodes []ast.Node, env formatEnvironment) ([]ast.Node, formatEnvironment) {
// 	return append(nodes, whitespaceNode), env
// }
//
// var afterLineBreakMatcher = astutil.NodeMatcher{
// 	ExpectKeyword: []string{
// 		"SELECT",
// 	},
// }
//
// func addAfterLinebreak(nodes []ast.Node, env formatEnvironment) ([]ast.Node, formatEnvironment) {
// 	env.indentLevel++
// 	return append(nodes, linebreakNode), env
// }
//
// var beforLineBreakMatcher = astutil.NodeMatcher{
// 	ExpectKeyword: []string{
// 		"FROM",
// 		"ON",
// 		"WHERE",
// 		"AND",
// 		"OR",
// 		"HAVING",
// 		"LIMIT",
// 		"UNION",
// 		"VALUES",
// 		"SET",
// 		"BETWEEN",
// 		"EXCEPT",
// 	},
// }
//
// func addBeforLinebreak(nodes []ast.Node, env formatEnvironment) ([]ast.Node, formatEnvironment) {
// 	return unshift(nodes, linebreakNode), env
// }
//
// type formatterMap struct {
// 	matcher   astutil.NodeMatcher
// 	ignore    astutil.NodeMatcher
// 	formatter additionalFormatter
// }
//
// func formattingProcess(reader *astutil.NodeReader) ast.TokenList {
// 	env := formatEnvironment{}
// 	fmaps := []formatterMap{
// 		{
// 			matcher:   afterWhiteSpaceMatcher,
// 			ignore:    afterWhiteSpaceMatcher,
// 			formatter: addAfterWhiteSpace,
// 		},
// 		{
// 			matcher:   afterLineBreakMatcher,
// 			formatter: addAfterLinebreak,
// 		},
// 		{
// 			matcher:   beforLineBreakMatcher,
// 			formatter: addBeforLinebreak,
// 		},
// 	}
// 	var formattedNodes []ast.Node
// 	for reader.NextNode(true) {
// 		additionalNodes := []ast.Node{reader.CurNode}
// 		isMatch := false
// 		for _, fmap := range fmaps {
// 			if reader.CurNodeIs(fmap.matcher) {
// 				additionalNodes, env = fmap.formatter(additionalNodes, env)
// 				isMatch = true
// 			}
// 		}
// 		if isMatch {
// 			formattedNodes = append(formattedNodes, additionalNodes...)
// 			continue
// 		}
//
// 		if list, ok := reader.CurNode.(ast.TokenList); ok {
// 			newReader := astutil.NewNodeReader(list)
// 			formattedNodes = append(formattedNodes, formattingProcess(newReader))
// 		} else {
// 			formattedNodes = append(formattedNodes, reader.CurNode)
// 		}
// 	}
// 	reader.Node.SetTokens(formattedNodes)
// 	return reader.Node
// }
