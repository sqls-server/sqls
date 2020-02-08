package parser

import (
	"bytes"
	"reflect"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/lighttiger2505/sqls/ast"
	"github.com/lighttiger2505/sqls/dialect"
	"github.com/lighttiger2505/sqls/token"
)

func TestExtractTable(t *testing.T) {
	testcases := []struct {
		name  string
		input string
		want  []*TableInfo
	}{
		{
			name:  "one table",
			input: "select * from abc",
			want: []*TableInfo{
				&TableInfo{
					Name: "abc",
				},
			},
		},
		{
			name:  "multiple table",
			input: "select * from abc, def",
			want: []*TableInfo{
				&TableInfo{
					Name: "abc",
				},
				&TableInfo{
					Name: "def",
				},
			},
		},
		{
			name:  "with database schema",
			input: "select * from abc.def",
			want: []*TableInfo{
				&TableInfo{
					DatabaseSchema: "abc",
					Name:           "def",
				},
			},
		},
		{
			name:  "with database schema and alias",
			input: "select * from abc.def as ghi",
			want: []*TableInfo{
				&TableInfo{
					DatabaseSchema: "abc",
					Name:           "def",
					Alias:          "ghi",
				},
			},
		},
	}

	for _, tt := range testcases {
		t.Run(tt.name, func(t *testing.T) {
			stmt := initExtractTable(t, tt.input)
			got := ExtractTable(stmt)
			if d := cmp.Diff(tt.want, got); d != "" {
				t.Errorf("unmatched value: %s", d)
			}
		})
	}
}

func initExtractTable(t *testing.T, input string) ast.TokenList {
	t.Helper()
	src := bytes.NewBuffer([]byte(input))
	parser, err := NewParser(src, &dialect.GenericSQLDialect{})
	if err != nil {
		t.Fatalf("error %+v\n", err)
	}

	parsed, err := parser.Parse()
	if err != nil {
		t.Fatalf("error %+v\n", err)
	}
	return parsed
}

// func TestPathEnclosingInterval(t *testing.T) {
// 	tests := []struct {
// 		name      string
// 		input     string
// 		start     token.Pos
// 		end       token.Pos
// 		wantPath  []ast.Node
// 		wantExact bool
// 	}{
// 		{
// 			name:      "",
// 			input:     "SELECT  FROM tabl",
// 			start:     genPosOneline(7),
// 			end:       genPosOneline(7),
// 			wantPath:  nil,
// 			wantExact: false,
// 		},
// 		{
// 			name:      "",
// 			input:     "SELECT  FROM sch.tabl",
// 			start:     genPosOneline(7),
// 			end:       genPosOneline(7),
// 			wantPath:  nil,
// 			wantExact: false,
// 		},
// 		{
// 			name:      "",
// 			input:     "SELECT * FROM tabl WHERE ",
// 			start:     genPosOneline(7),
// 			end:       genPosOneline(7),
// 			wantPath:  nil,
// 			wantExact: false,
// 		},
// 		{
// 			name:      "",
// 			input:     "select 1;select 2;select",
// 			start:     genPosOneline(7),
// 			end:       genPosOneline(7),
// 			wantPath:  nil,
// 			wantExact: false,
// 		},
// 	}
// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			// initialize
// 			src := bytes.NewBuffer([]byte(tt.input))
// 			parser, err := NewParser(src, &dialect.GenericSQLDialect{})
// 			if err != nil {
// 				t.Fatalf("error %+v\n", err)
// 			}
// 			parsed, err := parser.Parse()
// 			if err != nil {
// 				t.Fatalf("error %+v\n", err)
// 			}
//
// 			// execute
// 			gotPath, gotExact := PathEnclosingInterval(parsed, tt.start, tt.end)
// 			if !reflect.DeepEqual(gotPath, tt.wantPath) {
// 				t.Errorf("PathEnclosingInterval() gotPath = %v, want %v", gotPath, tt.wantPath)
// 			}
// 			if gotExact != tt.wantExact {
// 				t.Errorf("PathEnclosingInterval() gotExact = %v, want %v", gotExact, tt.wantExact)
// 			}
// 		})
// 	}
// }
//
// func genPosOneline(char int) token.Pos {
// 	return token.Pos{Line: 0, Col: char}
// }
