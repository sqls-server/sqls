package parseutil

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/sqls-server/sqls/token"
)

func TestExtractInsert(t *testing.T) {
	testcases := []struct {
		name  string
		input string
		pos   token.Pos
		tbl   *TableInfo
		cols  string
		vals  string
	}{
		{
			name:  "single",
			input: "insert into city (ID, Name, CountryCode) VALUES (123, 'aaa', '2020')",
			pos: token.Pos{
				Line: 0,
				Col:  50,
			},
			tbl: &TableInfo{
				Name: "city",
			},
			cols: "ID, Name, CountryCode",
			vals: "123, 'aaa', '2020'",
		},
		{
			name:  "multi forcus first",
			input: "insert into city (ID, Name, CountryCode) VALUES (123, 'aaa', '2020'), (456, 'bbb', '2021')",
			pos: token.Pos{
				Line: 0,
				Col:  50,
			},
			tbl: &TableInfo{
				Name: "city",
			},
			cols: "ID, Name, CountryCode",
			vals: "123, 'aaa', '2020'",
		},
		{
			name:  "multi forcus second",
			input: "insert into city (ID, Name, CountryCode) VALUES (123, 'aaa', '2020'), (456, 'bbb', '2021')",
			pos: token.Pos{
				Line: 0,
				Col:  72,
			},
			tbl: &TableInfo{
				Name: "city",
			},
			cols: "ID, Name, CountryCode",
			vals: "456, 'bbb', '2021'",
		},
	}

	for _, tt := range testcases {
		t.Run(tt.name, func(t *testing.T) {
			stmt := initExtractTable(t, tt.input)
			got, err := ExtractInsert(stmt, tt.pos)
			if err != nil {
				t.Fatalf("error: %+v", err)
			}
			if d := cmp.Diff(tt.tbl, got.GetTable()); d != "" {
				t.Errorf("unmatched table info(-want, +got): %s", d)
			}
			if d := cmp.Diff(tt.cols, got.GetColumns().String()); d != "" {
				t.Errorf("unmatched columns info(-want, +got): %s", d)
			}
			if d := cmp.Diff(tt.vals, got.GetValues().String()); d != "" {
				t.Errorf("unmatched values info(-want, +got): %s", d)
			}
		})
	}
}
