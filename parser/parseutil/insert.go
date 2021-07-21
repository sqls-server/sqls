package parseutil

import (
	"github.com/lighttiger2505/sqls/ast"
	"github.com/lighttiger2505/sqls/token"
)

type Insert struct {
	Tables  []*TableInfo
	Columns []*ast.IdentiferList
	Values  []*ast.IdentiferList
}

func (i *Insert) Enable() bool {
	if len(i.Tables) == 0 {
		return false
	}
	if len(i.Columns) == 0 {
		return false
	}
	if len(i.Values) == 0 {
		return false
	}
	return true
}

func (i *Insert) GetTable() *TableInfo {
	if len(i.Tables) == 0 {
		return nil
	}
	return i.Tables[0]
}

func (i *Insert) GetColumns() *ast.IdentiferList {
	if len(i.Columns) == 0 {
		return nil
	}
	return i.Columns[0]
}

func (i *Insert) GetValues() *ast.IdentiferList {
	if len(i.Values) == 0 {
		return nil
	}
	return i.Values[0]
}

func ExtractInsert(parsed ast.TokenList, pos token.Pos) (*Insert, error) {
	stmt, err := extractFocusedStatement(parsed, pos)
	if err != nil {
		return nil, err
	}

	tables, err := ExtractTable(parsed, pos)
	if err != nil {
		return nil, err
	}

	columns := []*ast.IdentiferList{}
	columnsNodes := ExtractInsertColumns(stmt)
	for _, n := range columnsNodes {
		c, ok := n.(*ast.IdentiferList)
		if ok {
			columns = append(columns, c)
		}
	}

	values := []*ast.IdentiferList{}
	valuesNodes := ExtractInsertValues(stmt, pos)
	for _, n := range valuesNodes {
		n, ok := n.(*ast.IdentiferList)
		if ok {
			values = append(values, n)
		}
	}

	res := &Insert{
		Tables:  tables,
		Columns: columns,
		Values:  values,
	}
	return res, nil
}
