package parseutil

import (
	"testing"

	"github.com/sqls-server/sqls/parser"
	"github.com/sqls-server/sqls/token"
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
		{
			name: "join tables",
			text: "select CountryCode from city join ",
			pos: token.Pos{
				Line: 0,
				Col:  34,
			},
			want: JoinClause,
		},
		{
			name: "join filtered tables",
			text: "select CountryCode from city join co",
			pos: token.Pos{
				Line: 0,
				Col:  36,
			},
			want: JoinClause,
		},
		{
			name: "left join tables",
			text: "select CountryCode from city left join ",
			pos: token.Pos{
				Line: 0,
				Col:  39,
			},
			want: JoinClause,
		},
		{
			name: "left outer join tables",
			text: "select CountryCode from city left outer join ",
			pos: token.Pos{
				Line: 0,
				Col:  45,
			},
			want: JoinClause,
		},
		{
			name: "join on columns",
			text: "select * from city left join country on ",
			pos: token.Pos{
				Line: 0,
				Col:  40,
			},
			want: JoinOn,
		},
		{
			name: "join on filtered columns",
			text: "select * from city left join country on co",
			pos: token.Pos{
				Line: 0,
				Col:  42,
			},
			want: WhereCondition,
		},
		{
			name: "join on table<Period>",
			text: "select * from city left join country on country.",
			pos: token.Pos{
				Line: 0,
				Col:  48,
			},
			want: ColName,
		},
		{
			name: "join on <Eq>",
			text: "select * from city left join country on country.Code =",
			pos: token.Pos{
				Line: 0,
				Col:  54,
			},
			want: WhereCondition,
		},
		{
			name: "join on <Eq><WhiteSpace>",
			text: "select * from city left join country on country.Code = ",
			pos: token.Pos{
				Line: 0,
				Col:  55,
			},
			want: WhereCondition,
		},
		{
			name: "join on ref tables filtered",
			text: "select * from city left join country on country.Code = ci",
			pos: token.Pos{
				Line: 0,
				Col:  57,
			},
			want: WhereCondition,
		},
		{
			name: "join on ref table<Period>",
			text: "select * from city left join country on country.Code = city.",
			pos: token.Pos{
				Line: 0,
				Col:  60,
			},
			want: ColName,
		},
		{
			name: "join alias snippet",
			text: "select * from city c left join country c1 on c1.Code",
			pos: token.Pos{
				Line: 0,
				Col:  39,
			},
			want: TableReference,
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
