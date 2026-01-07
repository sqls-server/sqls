package completer

import (
	"fmt"
	"strings"

	"github.com/sqls-server/sqls/internal/database"
	"github.com/sqls-server/sqls/internal/lsp"
	"github.com/sqls-server/sqls/parser/parseutil"
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
			Documentation: &lsp.MarkupContent{
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

func (c *Completer) joinCandidates(lastTable *parseutil.TableInfo,
	targetTables, allTables []*parseutil.TableInfo,
	joinOn, lowercaseKeywords bool) []lsp.CompletionItem {
	var candidates []lsp.CompletionItem
	if len(c.DBCache.ForeignKeys) == 0 {
		return candidates
	}

	tMap := make(map[string]*parseutil.TableInfo)
	for _, t := range targetTables {
		tMap[t.Name] = t
	}
	fkMap := make(map[string][][]*database.ForeignKey)
	if lastTable == nil {
		for t := range tMap {
			for k, v := range c.DBCache.ForeignKeys[t] {
				fkMap[k] = append(fkMap[k], v)
			}
		}
	} else {
		delete(tMap, lastTable.Name)
		rTab := []*parseutil.TableInfo{lastTable}
		if !joinOn {
			rTab = resolveTables(lastTable, c.DBCache)
		}
		for _, lt := range rTab {
			for k, v := range c.DBCache.ForeignKeys[lt.Name] {
				if _, ok := tMap[k]; ok {
					fkMap[lt.Name] = append(fkMap[lt.Name], v)
				}
			}
		}

		for _, t := range rTab {
			if _, ok := tMap[t.Name]; !ok {
				tMap[t.Name] = t
			}
		}
	}

	aliases := make(map[string]interface{})
	for _, t := range allTables {
		if t.Alias != "" {
			aliases[t.Alias] = true
		}
	}

	for k, v := range fkMap {
		for _, fks := range v {
			for _, fk := range fks {
				candidates = append(candidates, generateForeignKeyCandidate(k, tMap, aliases,
					fk, joinOn, lowercaseKeywords))
			}
		}
	}
	return candidates
}

func resolveTables(t *parseutil.TableInfo, cache *database.DBCache) []*parseutil.TableInfo {
	if _, ok := cache.ColumnDescs(t.Name); ok {
		return []*parseutil.TableInfo{t}
	}
	var rv []*parseutil.TableInfo
	targetName := strings.ToLower(t.Name)
	for _, cond := range cache.SortedTables() {
		if strings.Contains(strings.ToLower(cond), targetName) {
			rv = append(rv, &parseutil.TableInfo{
				Name: cond,
			})
		}
	}
	return rv
}

func generateTableAlias(target string,
	aliases map[string]interface{}) string {
	ch := []rune(target)[0]
	i := 1
	var rv string
	for {
		rv = fmt.Sprintf("%c%d", ch, i)
		if _, ok := aliases[rv]; ok {
			i++
			continue
		}
		break
	}
	return rv
}

func generateForeignKeyCandidate(target string,
	tMap map[string]*parseutil.TableInfo,
	aliases map[string]interface{},
	fk *database.ForeignKey,
	joinOn, lowercaseKeywords bool) lsp.CompletionItem {
	var tAlias string
	if joinOn {
		tAlias = tMap[target].Alias
		if tAlias == "" {
			tAlias = tMap[target].Name
		}
	} else {
		tAlias = generateTableAlias(target, aliases)
	}
	builder := []struct {
		sb    *strings.Builder
		alias string
	}{
		{
			sb:    &strings.Builder{},
			alias: tAlias,
		},
		{
			sb:    &strings.Builder{},
			alias: tAlias,
		},
	}
	if !joinOn {
		builder[1].alias = fmt.Sprintf("${1:%s}", tAlias)
		onKw := "ON"
		if lowercaseKeywords {
			onKw = "on"
		}
		for _, b := range builder {
			fmt.Fprintf(b.sb, "%s %s %s ", target, b.alias, onKw)
		}
	}
	andKw := " AND "
	if lowercaseKeywords {
		andKw = " and "
	}
	prefix := ""
	for _, cur := range *fk {
		tIdx, rIdx := 0, 1
		if cur[rIdx].Table == target {
			tIdx, rIdx = rIdx, tIdx
		}
		for _, b := range builder {
			b.sb.WriteString(prefix)
		}
		prefix = andKw
		for _, b := range builder {
			b.sb.WriteString(strings.Join([]string{b.alias, cur[tIdx].Name}, "."))
			b.sb.WriteString(" = ")
		}
		rAlias := tMap[cur[rIdx].Table].Alias
		if rAlias == "" {
			rAlias = cur[rIdx].Table
		}
		for _, b := range builder {
			b.sb.WriteString(strings.Join([]string{rAlias, cur[rIdx].Name}, "."))
		}
	}
	builder[1].sb.WriteString("$0")
	return lsp.CompletionItem{
		Label:            builder[0].sb.String(),
		Kind:             lsp.SnippetCompletion,
		Detail:           "Join generator for foreign key",
		InsertText:       builder[1].sb.String(),
		InsertTextFormat: lsp.SnippetTextFormat,
	}
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
			candidate.Documentation = &lsp.MarkupContent{
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
			candidate.Documentation = &lsp.MarkupContent{
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
			Documentation: &lsp.MarkupContent{
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
							Documentation: &lsp.MarkupContent{
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
						Documentation: &lsp.MarkupContent{
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
