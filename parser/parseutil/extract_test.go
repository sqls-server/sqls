package parseutil

import (
	"testing"

	"github.com/sqls-server/sqls/token"
)

func TestExtractSelectExpr(t *testing.T) {
	testcases := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "",
			input: "SELECT abc",
			want:  "abc",
		},
		{
			name:  "",
			input: "SELECT abc, def",
			want:  "abc, def",
		},
		{
			name:  "",
			input: "SELECT ALL abc",
			want:  "abc",
		},
		{
			name:  "",
			input: "SELECT DISTINCT abc",
			want:  "abc",
		},
	}
	for _, tt := range testcases {
		t.Run(tt.name, func(t *testing.T) {
			query := initExtractTable(t, tt.input)
			got := ExtractSelectExpr(query)
			if len(got) == 0 {
				t.Fatalf("not found filtered node")
			}
			if tt.want != got[0].String() {
				t.Errorf("expected %q, got %q", tt.want, got[0].String())
			}
		})
	}
}

func TestExtractTableReferences(t *testing.T) {
	testcases := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "",
			input: "FROM abc",
			want:  "abc",
		},
		{
			name:  "",
			input: "FROM abc, def",
			want:  "abc, def",
		},
		{
			name:  "",
			input: "UPDATE abc",
			want:  "abc",
		},
		{
			name:  "",
			input: "UPDATE abc, def",
			want:  "abc, def",
		},
	}
	for _, tt := range testcases {
		t.Run(tt.name, func(t *testing.T) {
			query := initExtractTable(t, tt.input)
			got := ExtractTableReferences(query)
			if len(got) == 0 {
				t.Fatalf("not found filtered node")
			}
			if tt.want != got[0].String() {
				t.Errorf("expected %q, got %q", tt.want, got[0].String())
			}
		})
	}
}

func TestExtractTableReference(t *testing.T) {
	testcases := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "INSERT INTO",
			input: "INSERT INTO abc",
			want:  "abc",
		},
		// FIXME
		// {
		// 	name:  "DELETE FROM",
		// 	input: "DELETE FROM abc",
		// 	want:  "abc",
		// },
	}
	for _, tt := range testcases {
		t.Run(tt.name, func(t *testing.T) {
			query := initExtractTable(t, tt.input)
			got := ExtractTableReference(query)
			if len(got) == 0 {
				t.Fatalf("not found filtered node")
			}
			if tt.want != got[0].String() {
				t.Errorf("expected %q, got %q", tt.want, got[0].String())
			}
		})
	}
}

func TestExtractTableFactor(t *testing.T) {
	testcases := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "only",
			input: "JOIN abc",
			want:  "abc",
		},
	}
	for _, tt := range testcases {
		t.Run(tt.name, func(t *testing.T) {
			query := initExtractTable(t, tt.input)
			got := ExtractTableFactor(query)
			if len(got) == 0 {
				t.Fatalf("not found filtered node")
			}
			if tt.want != got[0].String() {
				t.Errorf("expected %q, got %q", tt.want, got[0].String())
			}
		})
	}
}
func TestExtractWhereCondition(t *testing.T) {
	testcases := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "only",
			input: "WHERE id = 123",
			want:  "id = 123",
		},
		{
			name:  "SELECT",
			input: "SELECT id, name FROM abc WHERE id = 123",
			want:  "id = 123",
		},
		{
			name:  "UPDATE",
			input: "UPDATE abc SET name = 'foo' WHERE id = 123",
			want:  "id = 123",
		},
		{
			name:  "DELETE",
			input: "DELETE FROM abc WHERE id = 123",
			want:  "id = 123",
		},
	}
	for _, tt := range testcases {
		t.Run(tt.name, func(t *testing.T) {
			query := initExtractTable(t, tt.input)
			got := ExtractWhereCondition(query)
			if len(got) == 0 {
				t.Fatalf("not found filtered node")
			}
			if tt.want != got[0].String() {
				t.Errorf("expected %q, got %q", tt.want, got[0].String())
			}
		})
	}
}

func TestExtractAliasedIdentifier(t *testing.T) {
	testcases := []struct {
		name  string
		input string
		want  []string
	}{
		{
			name:  "one alias",
			input: "abc as a",
			want: []string{
				"abc as a",
			},
		},
		{
			name:  "two alias",
			input: "abc as a, def as d",
			want: []string{
				"abc as a",
				"def as d",
			},
		},
		{
			name:  "through sub query",
			input: "(select * from abc) as t",
			want:  []string{},
		},
	}
	for _, tt := range testcases {
		t.Run(tt.name, func(t *testing.T) {
			query := initExtractTable(t, tt.input)
			gots := ExtractAliasedIdentifier(query)

			if len(gots) != len(tt.want) {
				t.Errorf("contain nodes %d, got %d", len(tt.want), len(gots))
				return
			}
			for i, got := range gots {
				if tt.want[i] != got.String() {
					t.Errorf("expected %q, got %q", tt.want[i], got.String())
				}
			}
		})
	}
}

func TestExtractInsertColumns(t *testing.T) {
	testcases := []struct {
		name  string
		input string
		want  []string
	}{
		{
			name:  "full",
			input: "insert into city (ID, Name, CountryCode) VALUES (123, 'aaa', '2020')",
			want: []string{
				"ID, Name, CountryCode",
			},
		},
		{
			name:  "with out statement",
			input: "city (ID, Name, CountryCode) VALUES (123, 'aaa', '2020')",
			want: []string{
				"ID, Name, CountryCode",
			},
		},
		{
			name:  "minimum",
			input: "city (ID, Name, CountryCode)",
			want: []string{
				"ID, Name, CountryCode",
			},
		},
	}
	for _, tt := range testcases {
		t.Run(tt.name, func(t *testing.T) {
			query := initExtractTable(t, tt.input)
			gots := ExtractInsertColumns(query)

			if len(gots) != len(tt.want) {
				t.Errorf("contain nodes %d, got %d (%v)", len(tt.want), len(gots), gots)
				return
			}
			for i, got := range gots {
				if tt.want[i] != got.String() {
					t.Errorf("expected %q, got %q", tt.want[i], got.String())
				}
			}
		})
	}
}

func TestExtractInsertValues(t *testing.T) {
	testcases := []struct {
		name  string
		input string
		pos   token.Pos
		want  []string
	}{
		{
			name:  "full",
			input: "insert into city (ID, Name, CountryCode) VALUES (123, 'aaa', '2020')",
			pos: token.Pos{
				Line: 0,
				Col:  50,
			},
			want: []string{
				"123, 'aaa', '2020'",
			},
		},
		{
			name:  "multi value",
			input: "insert into city (ID, Name, CountryCode) VALUES (123, 'aaa', '2020'), (456, 'bbb', '2021')",
			pos: token.Pos{
				Line: 0,
				Col:  72,
			},
			want: []string{
				"456, 'bbb', '2021'",
			},
		},
		{
			name:  "with out statement",
			input: "city (ID, Name, CountryCode) VALUES (123, 'aaa', '2020')",
			pos: token.Pos{
				Line: 0,
				Col:  38,
			},
			want: []string{
				"123, 'aaa', '2020'",
			},
		},
		{
			name:  "minimum",
			input: "VALUES (123, 'aaa', '2020')",
			pos: token.Pos{
				Line: 0,
				Col:  9,
			},
			want: []string{
				"123, 'aaa', '2020'",
			},
		},
	}
	for _, tt := range testcases {
		t.Run(tt.name, func(t *testing.T) {
			query := initExtractTable(t, tt.input)
			gots := ExtractInsertValues(query, tt.pos)

			if len(gots) != len(tt.want) {
				t.Errorf("contain nodes %d, got %d (%v)", len(tt.want), len(gots), gots)
				return
			}
			for i, got := range gots {
				if tt.want[i] != got.String() {
					t.Errorf("expected %q, got %q", tt.want[i], got.String())
				}
			}
		})
	}
}
