package parseutil

import (
	"testing"

	"github.com/lighttiger2505/sqls/parser"
	"github.com/lighttiger2505/sqls/token"
)

func TestCheckSyntaxPosition(t *testing.T) {
	tests := []struct {
		name string
		text string
		pos  token.Pos
		want SyntaxPosition
	}{
		{
			name: "insert column",
			text: "insert into city (ID, Name, CountryCode) VALUES (123, 'aaa', '2020')",
			pos: token.Pos{
				Line: 0,
				Col:  57,
			},
			want: InsertValue,
		},
		{
			name: "on lparen",
			text: "insert into city (ID, Name, CountryCode) VALUES (",
			pos: token.Pos{
				Line: 0,
				Col:  49,
			},
			want: InsertValue,
		},
		{
			name: "with space first param",
			text: "insert into city (ID, Name, CountryCode) VALUES (   ",
			pos: token.Pos{
				Line: 0,
				Col:  52,
			},
			want: InsertValue,
		},
		{
			name: "second param",
			text: "insert into city (ID, Name, CountryCode) VALUES (123, ",
			pos: token.Pos{
				Line: 0,
				Col:  54,
			},
			want: InsertValue,
		},
		{
			name: "white space with second param",
			text: "insert into city (ID, Name, CountryCode) VALUES (123,   ",
			pos: token.Pos{
				Line: 0,
				Col:  56,
			},
			want: InsertValue,
		},
		{
			name: "third param",
			text: "insert into city (ID, Name, CountryCode) VALUES (123, 'aaa', '2020' ",
			pos: token.Pos{
				Line: 0,
				Col:  68,
			},
			want: InsertValue,
		},
		{
			name: "white space with third param",
			text: "insert into city (ID, Name, CountryCode) VALUES (123, 'aaa', '2020'   ",
			pos: token.Pos{
				Line: 0,
				Col:  70,
			},
			want: InsertValue,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parsed, err := parser.Parse(tt.text)
			if err != nil {
				t.Errorf("parse error, %v", err)
				return
			}

			nodeWalker := NewNodeWalker(parsed, tt.pos)
			if got := CheckSyntaxPosition(nodeWalker); got != tt.want {
				t.Errorf("unmatch syntax position got %v, want %v", got, tt.want)
			}
		})
	}
}
