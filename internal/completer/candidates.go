package completer

import (
	"strings"

	"github.com/lighttiger2505/sqls/internal/lsp"
	"github.com/lighttiger2505/sqls/parser/parseutil"
)

func (c *Completer) keywordCandidates() []lsp.CompletionItem {
	candidates := []lsp.CompletionItem{}
	for _, k := range keywords {
		candidate := lsp.CompletionItem{
			Label:  k,
			Kind:   lsp.KeywordCompletion,
			Detail: "keyword",
		}
		candidates = append(candidates, candidate)
	}
	return candidates
}

func (c *Completer) columnCandidates(targetTables []*parseutil.TableInfo, pare *parent) []lsp.CompletionItem {
	candidates := []lsp.CompletionItem{}
	if c.DBCache == nil {
		return candidates
	}

	switch pare.Type {
	case ParentTypeNone:
		for _, table := range targetTables {
			if table.DatabaseSchema != "" && table.Name != "" {
				columns, ok := c.DBCache.ColumnDatabase(table.DatabaseSchema, table.Name)
				if !ok {
					continue
				}
				for _, column := range columns {
					detail := strings.Join(
						[]string{
							"column",
							"`" + table.Name + "`",
							"(" + column.OnelineDesc() + ")",
						},
						" ",
					)
					candidate := lsp.CompletionItem{
						Label:  column.Name,
						Kind:   lsp.FieldCompletion,
						Detail: detail,
					}
					candidates = append(candidates, candidate)
				}
			} else if table.Name != "" {
				columns, ok := c.DBCache.ColumnDescs(table.Name)
				if !ok {
					continue
				}
				for _, column := range columns {
					detail := strings.Join(
						[]string{
							"column",
							"`" + table.Name + "`",
							"(" + column.OnelineDesc() + ")",
						},
						" ",
					)
					candidate := lsp.CompletionItem{
						Label:  column.Name,
						Kind:   lsp.FieldCompletion,
						Detail: detail,
					}
					candidates = append(candidates, candidate)
				}
			}
		}
	case ParentTypeSchema:
	case ParentTypeTable:
		for _, table := range targetTables {
			if table.Name != pare.Name && table.Alias != pare.Name {
				continue
			}
			if c.DBCache == nil {
				continue
			}
			columns, ok := c.DBCache.ColumnDescs(table.Name)
			if !ok {
				continue
			}
			for _, column := range columns {
				detail := strings.Join(
					[]string{
						"column",
						"`" + table.Name + "`",
						"(" + column.OnelineDesc() + ")",
					},
					" ",
				)
				candidate := lsp.CompletionItem{
					Label:  column.Name,
					Kind:   lsp.FieldCompletion,
					Detail: detail,
				}
				candidates = append(candidates, candidate)
			}
		}
	}
	return candidates
}

func (c *Completer) TableCandidates(pare *parent) []lsp.CompletionItem {
	candidates := []lsp.CompletionItem{}
	if c.DBCache == nil {
		return candidates
	}

	switch pare.Type {
	case ParentTypeNone:
		tables := c.DBCache.SortedTables()
		for _, tableName := range tables {
			candidate := lsp.CompletionItem{
				Label:  tableName,
				Kind:   lsp.FieldCompletion,
				Detail: "table",
			}
			candidates = append(candidates, candidate)
		}
	case ParentTypeSchema:
		tables, ok := c.DBCache.SortedTablesByDBName(pare.Name)
		if ok {
			for _, tableName := range tables {
				candidate := lsp.CompletionItem{
					Label:  tableName,
					Kind:   lsp.FieldCompletion,
					Detail: "table",
				}
				candidates = append(candidates, candidate)
			}
		} else {
			tables := c.DBCache.SortedTables()
			for _, tableName := range tables {
				candidate := lsp.CompletionItem{
					Label:  tableName,
					Kind:   lsp.FieldCompletion,
					Detail: "table",
				}
				candidates = append(candidates, candidate)
			}
		}
	case ParentTypeTable:
	}
	return candidates
}

func (c *Completer) aliasCandidates(targetTables []*parseutil.TableInfo) []lsp.CompletionItem {
	candidates := []lsp.CompletionItem{}
	for _, table := range targetTables {
		if table.Alias == "" {
			continue
		}
		detail := strings.Join(
			[]string{
				"alias",
				"`" + table.Name + "`",
			},
			" ",
		)
		candidate := lsp.CompletionItem{
			Label:  table.Alias,
			Kind:   lsp.FieldCompletion,
			Detail: detail,
		}
		candidates = append(candidates, candidate)
	}
	return candidates
}

func (c *Completer) SubQueryColumnCandidates(info *parseutil.SubQueryInfo) []lsp.CompletionItem {
	candidates := []lsp.CompletionItem{}
	for _, view := range info.Views {
		for _, colmun := range view.Columns {
			candidate := lsp.CompletionItem{
				Label:  colmun,
				Kind:   lsp.FieldCompletion,
				Detail: "subQuery",
			}
			candidates = append(candidates, candidate)
		}
	}
	return candidates
}

func (c *Completer) SchemaCandidates() []lsp.CompletionItem {
	candidates := []lsp.CompletionItem{}
	if c.DBCache == nil {
		return candidates
	}
	dbs := c.DBCache.SortedSchemas()
	for _, db := range dbs {
		candidate := lsp.CompletionItem{
			Label:  db,
			Kind:   lsp.FieldCompletion,
			Detail: "schema",
		}
		candidates = append(candidates, candidate)
	}
	return candidates
}
