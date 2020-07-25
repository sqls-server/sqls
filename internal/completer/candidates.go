package completer

import (
	"strings"

	"github.com/lighttiger2505/sqls/internal/database"
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

func (c *Completer) columnCandidates(targetTables []*parseutil.TableInfo, parent *completionParent) []lsp.CompletionItem {
	candidates := []lsp.CompletionItem{}
	if c.DBCache == nil {
		return candidates
	}

	switch parent.Type {
	case ParentTypeNone:
		for _, table := range targetTables {
			if table.DatabaseSchema != "" && table.Name != "" {
				columns, ok := c.DBCache.ColumnDatabase(table.DatabaseSchema, table.Name)
				if !ok {
					continue
				}
				candidates = append(candidates, generateColumnCandidates(table.Name, columns)...)
			} else if table.Name != "" {
				columns, ok := c.DBCache.ColumnDescs(table.Name)
				if !ok {
					continue
				}
				candidates = append(candidates, generateColumnCandidates(table.Name, columns)...)
			}
		}
	case ParentTypeSchema:
	case ParentTypeTable:
		for _, table := range targetTables {
			if table.Name != parent.Name && table.Alias != parent.Name {
				continue
			}
			columns, ok := c.DBCache.ColumnDescs(table.Name)
			if !ok {
				continue
			}
			candidates = append(candidates, generateColumnCandidates(table.Name, columns)...)
		}
	}
	return candidates
}

func generateColumnCandidates(tableName string, columns []*database.ColumnDesc) []lsp.CompletionItem {
	candidates := []lsp.CompletionItem{}
	for _, column := range columns {
		candidate := lsp.CompletionItem{
			Label:  column.Name,
			Kind:   lsp.FieldCompletion,
			Detail: columnDetail(tableName, column),
			Documentation: lsp.MarkupContent{
				Kind:  lsp.Markdown,
				Value: database.ColumnDoc(tableName, column),
			},
		}
		candidates = append(candidates, candidate)
	}
	return candidates
}

func columnDetail(tableName string, column *database.ColumnDesc) string {
	detail := strings.Join(
		[]string{
			"column from",
			"\"" + tableName + "\"",
		},
		" ",
	)
	return detail
}

func (c *Completer) ReferencedTableCandidates(targetTables []*parseutil.TableInfo) []lsp.CompletionItem {
	candidates := []lsp.CompletionItem{}
	if c.DBCache == nil {
		return candidates
	}

	for _, targetTable := range targetTables {
		includeTables := []string{}
		for _, table := range c.DBCache.SortedTables() {
			if table == targetTable.Name {
				includeTables = append(includeTables, table)
			}
		}
		candidates = append(candidates, generateTableCandidates(includeTables, c.DBCache, "referenced table")...)
	}
	return candidates
}

func (c *Completer) TableCandidates(parent *completionParent, targetTables []*parseutil.TableInfo) []lsp.CompletionItem {
	candidates := []lsp.CompletionItem{}
	if c.DBCache == nil {
		return candidates
	}

	switch parent.Type {
	case ParentTypeNone:
		excludeTables := []string{}
		for _, table := range c.DBCache.SortedTables() {
			isExclude := false
			for _, targetTable := range targetTables {
				if table == targetTable.Name {
					isExclude = true
				}
			}
			if isExclude {
				continue
			}
			excludeTables = append(excludeTables, table)
		}
		candidates = append(candidates, generateTableCandidates(excludeTables, c.DBCache, "table")...)
	case ParentTypeSchema:
		tables, ok := c.DBCache.SortedTablesByDBName(parent.Name)
		if ok {
			candidates = append(candidates, generateTableCandidates(tables, c.DBCache, "table")...)
		} else {
			tables := c.DBCache.SortedTables()
			candidates = append(candidates, generateTableCandidates(tables, c.DBCache, "table")...)
		}
	case ParentTypeTable:
	}
	return candidates
}

func generateTableCandidates(tables []string, dbCache *database.DatabaseCache, detail string) []lsp.CompletionItem {
	candidates := []lsp.CompletionItem{}
	for _, tableName := range tables {
		candidate := lsp.CompletionItem{
			Label:  tableName,
			Kind:   lsp.FieldCompletion,
			Detail: detail,
		}
		cols, ok := dbCache.ColumnDescs(tableName)
		if ok {
			candidate.Documentation = lsp.MarkupContent{
				Kind:  lsp.Markdown,
				Value: database.TableDoc(tableName, cols),
			}
		}
		candidates = append(candidates, candidate)
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
