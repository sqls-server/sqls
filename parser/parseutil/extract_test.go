package parseutil

import (
	"testing"
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
				t.Fatalf("not found filterd node")
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
				t.Fatalf("not found filterd node")
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
		{
			name:  "UPDATE",
			input: "UPDATE abc",
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
				t.Fatalf("not found filterd node")
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
				t.Fatalf("not found filterd node")
			}
			if tt.want != got[0].String() {
				t.Errorf("expected %q, got %q", tt.want, got[0].String())
			}
		})
	}
}
func TestExtractWhereConditon(t *testing.T) {
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
				t.Fatalf("not found filterd node")
			}
			if tt.want != got[0].String() {
				t.Errorf("expected %q, got %q", tt.want, got[0].String())
			}
		})
	}
}
