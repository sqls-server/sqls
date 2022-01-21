package completer

import (
	"strings"

	"github.com/lighttiger2505/sqls/internal/database"
	"github.com/lighttiger2505/sqls/internal/lsp"
	"github.com/lighttiger2505/sqls/parser/parseutil"
)

func (c *Completer) keywordCandidates(lower bool, keywords []string) []lsp.CompletionItem {
	candidates := []lsp.CompletionItem{}
	for _, k := range keywords {
		candidate := lsp.CompletionItem{
			Label:  k,
			Kind:   lsp.KeywordCompletion,
			Detail: "keyword",
		}
		if lower {
			candidate.Label = strings.ToLower(candidate.Label)
		}
		candidates = append(candidates, candidate)
	}
	return candidates
}

func (c *Completer) functionCandidates(lower bool, keywords []string) []lsp.CompletionItem {
	candidates := []lsp.CompletionItem{}
	for _, k := range keywords {
		candidate := lsp.CompletionItem{
			Label:  k,
			Kind:   lsp.FunctionCompletion,
			Detail: "Function",
		}
		if lower {
			candidate.Label = strings.ToLower(candidate.Label)
		}
		candidates = append(candidates, candidate)
	}
	return candidates
}

func (c *Completer) columnCandidates(targetTables []*parseutil.TableInfo, parent *completionParent) []lsp.CompletionItem {
	candidates := []lsp.CompletionItem{}

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
		// pass
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
	case ParentTypeSubQuery:
		// pass
	}
	return candidates
}

func generateColumnCandidates(tableName string, columns []*database.ColumnDesc) []lsp.CompletionItem {
	candidates := []lsp.CompletionItem{}
	for _, column := range columns {
		candidate := lsp.CompletionItem{
			Label:  column.Name,
			Kind:   lsp.FieldCompletion,
			Detail: columnDetail(tableName),
			Documentation: lsp.MarkupContent{
				Kind:  lsp.Markdown,
				Value: database.ColumnDoc(tableName, column),
			},
		}
		candidates = append(candidates, candidate)
	}
	return candidates
}

func columnDetail(tableName string) string {
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

	for _, targetTable := range targetTables {
		includeTables := []*parseutil.TableInfo{}
		for _, table := range c.DBCache.SortedTables() {
			if table == targetTable.Name {
				includeTables = append(includeTables, targetTable)
			}
		}
		genCands := generateTableCandidatesByInfos(includeTables, c.DBCache)
		candidates = append(candidates, genCands...)
	}
	return candidates
}

func (c *Completer) TableCandidates(parent *completionParent, targetTables []*parseutil.TableInfo) []lsp.CompletionItem {
	candidates := []lsp.CompletionItem{}

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
		candidates = append(candidates, generateTableCandidates(excludeTables, c.DBCache)...)
	case ParentTypeSchema:
		tables, ok := c.DBCache.SortedTablesByDBName(parent.Name)
		if ok {
			candidates = append(candidates, generateTableCandidates(tables, c.DBCache)...)
		}
	case ParentTypeTable:
		// pass
	case ParentTypeSubQuery:
		// pass
	}
	return candidates
}

func generateTableCandidates(tables []string, dbCache *database.DBCache) []lsp.CompletionItem {
	candidates := []lsp.CompletionItem{}
	for _, tableName := range tables {
		candidate := lsp.CompletionItem{
			Label:  tableName,
			Kind:   lsp.ClassCompletion,
			Detail: "table",
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

func generateTableCandidatesByInfos(tables []*parseutil.TableInfo, dbCache *database.DBCache) []lsp.CompletionItem {
	candidates := []lsp.CompletionItem{}
	for _, table := range tables {
		name := table.Name
		detail := "referenced table"
		if table.Alias != "" {
			name = table.Alias
			detail = "aliased table"
		}
		candidate := lsp.CompletionItem{
			Label:  name,
			Kind:   lsp.ClassCompletion,
			Detail: detail,
		}
		cols, ok := dbCache.ColumnDescs(table.Name)
		if ok {
			candidate.Documentation = lsp.MarkupContent{
				Kind:  lsp.Markdown,
				Value: database.TableDoc(table.Name, cols),
			}
		}
		candidates = append(candidates, candidate)
	}
	return candidates
}

func (c *Completer) SubQueryCandidates(infos []*parseutil.SubQueryInfo) []lsp.CompletionItem {
	candidates := []lsp.CompletionItem{}
	for _, info := range infos {
		candidate := lsp.CompletionItem{
			Label:  info.Name,
			Kind:   lsp.FieldCompletion,
			Detail: "subquery",
			Documentation: lsp.MarkupContent{
				Kind:  lsp.Markdown,
				Value: database.SubqueryDoc(info.Name, info.Views, c.DBCache),
			},
		}
		candidates = append(candidates, candidate)
	}
	return candidates
}

func (c *Completer) SubQueryColumnCandidates(infos []*parseutil.SubQueryInfo) []lsp.CompletionItem {
	candidates := []lsp.CompletionItem{}
	for _, info := range infos {
		for _, view := range info.Views {
			for _, col := range view.SubQueryColumns {
				if col.ColumnName == "*" {
					tableCols, ok := c.DBCache.ColumnDescs(col.ParentTable.Name)
					if !ok {
						continue
					}
					for _, tableCol := range tableCols {
						candidate := lsp.CompletionItem{
							Label:  tableCol.Name,
							Kind:   lsp.FieldCompletion,
							Detail: subQueryColumnDetail(info.Name),
							Documentation: lsp.MarkupContent{
								Kind:  lsp.Markdown,
								Value: database.SubqueryColumnDoc(tableCol.Name, info.Views, c.DBCache),
							},
						}
						candidates = append(candidates, candidate)
					}
				} else {
					candidate := lsp.CompletionItem{
						Label:  col.DisplayName(),
						Kind:   lsp.FieldCompletion,
						Detail: subQueryColumnDetail(info.Name),
						Documentation: lsp.MarkupContent{
							Kind:  lsp.Markdown,
							Value: database.SubqueryColumnDoc(col.DisplayName(), info.Views, c.DBCache),
						},
					}
					candidates = append(candidates, candidate)
				}
			}
		}
	}
	return candidates
}

func subQueryColumnDetail(subQueryAliasName string) string {
	detail := strings.Join(
		[]string{
			"subquery column from",
			"\"" + subQueryAliasName + "\"",
		},
		" ",
	)
	return detail
}

func (c *Completer) SchemaCandidates() []lsp.CompletionItem {
	candidates := []lsp.CompletionItem{}
	dbs := c.DBCache.SortedSchemas()
	for _, db := range dbs {
		candidate := lsp.CompletionItem{
			Label:  db,
			Kind:   lsp.ModuleCompletion,
			Detail: "schema",
		}
		candidates = append(candidates, candidate)
	}
	return candidates
}
