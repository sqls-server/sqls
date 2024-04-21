package handler

import (
	"context"
	"fmt"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/rust"
	"github.com/sqls-server/sqls/internal/lsp"
)

func cleanInjections(file *File, position lsp.Position) string {
	switch file.LanguageID {
	case `sql`:
		return file.Text
	case `rust`:
		return cleanRustInjections(file, position)

	default:
		return file.Text
	}
}

func cleanRustInjections(file *File, position lsp.Position) string {
	result := make([]byte, len(file.Text))
	for i := range result {
		if file.Text[i] == '\n' || file.Text[i] == '\r' {
			result[i] = file.Text[i]
			continue
		}
		result[i] = byte(' ')
	}

	lang := rust.GetLanguage()
	tree, err := sitter.ParseCtx(context.Background(), []byte(file.Text), lang)
	if err != nil {
		return file.Text
	}

	// TODO: this would need to be configurable
	query := `((call_expression
  function: (scoped_identifier
    path: ((identifier) @_sqlx
      (#eq? @_sqlx "sqlx"))
    name: (identifier) @_query
    (#eq? @_query "query"))
  arguments: (arguments
    ((string_literal)
       @injection.content))))`

	q, err := sitter.NewQuery([]byte(query), lang)
	if err != nil {
		return file.Text
	}
	qc := sitter.NewQueryCursor()
	qc.Exec(q, tree)
	// Iterate over query results
	for {
		m, ok := qc.NextMatch()
		if !ok {
			break
		}
		// Apply predicates filtering
		m = qc.FilterPredicates(m, []byte(file.Text))
		for _, c := range m.Captures {
			fmt.Println(c.Node.String())
			fmt.Println(c.Node.Content([]byte(file.Text)))
			captureName := q.CaptureNameForId(c.Index)
			if captureName == "injection.content" {
				// TODO: add a check if the position is inside the injection
				copy(result[c.Node.StartByte()+1:c.Node.EndByte()-1],
					file.Text[c.Node.StartByte()+1:c.Node.EndByte()-1])
			}
		}
	}
	return string(result)
}
