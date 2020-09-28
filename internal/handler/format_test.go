package handler

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/lighttiger2505/sqls/internal/lsp"
)

// ## case3 statement

// select a,
// case when a = 0
// then 1
// when bb = 1 then 1
// when c = 2 then 2
// else 0 end as d,
// extra_col
// from table
// where c is true
// and b between 3 and 4

// select a,
//        case when a = 0  then 1
//             when bb = 1 then 1
//             when c = 2  then 2
//             else 0
//              end as d,
//        extra_col
//   from table
//  where c is true
//    and b between 3 and 4

// ## case statement_with_between

// select a,
// case when a = 0
// then 1
// when bb = 1 then 1
// when c = 2 then 2
// when d between 3 and 5 then 3
// else 0 end as d,
// extra_col
// from table
// where c is true
// and b between 3 and 4

// select a,
//        case when a = 0             then 1
//             when bb = 1            then 1
//             when c = 2             then 2
//             when d between 3 and 5 then 3
//             else 0
//              end as d,
//        extra_col
//   from table
//  where c is true
//    and b between 3 and 4

// ## case group_by

// select a, b, c, sum(x) as sum_x, count(y) as cnt_y
// from table
// group by a,b,c
// having sum(x) > 1
// and count(y) > 5
// order by 3,2,1

// select a,
//        b,
//        c,
//        sum(x) as sum_x,
//        count(y) as cnt_y
//   from table
//  group by a,
//           b,
//           c
// having sum(x) > 1
//    and count(y) > 5
//  order by 3,
//           2,
//           1

// ## case group_by_subquery

// select *, sum_b + 2 as mod_sum
// from (
//     select a, sum(b) as sum_b
//     from table
//     group by a,z)
// order by 1,2

// select *,
//        sum_b + 2 as mod_sum
//   from (
//         select a,
//                sum(b) as sum_b
//           from table
//          group by a,
//                   z
//        )
//  order by 1,
//           2

func TestFormatting(t *testing.T) {
	tx := newTestContext()
	tx.initServer(t)
	defer tx.tearDown()

	uri := "file:///Users/octref/Code/css-test/test.sql"

	type formattingTestCase struct {
		name  string
		input string
		line  int
		col   int
		want  string
	}
	testCase := []formattingTestCase{
		// 		{
		// 			name:  "select",
		// 			input: "SELECT ID , Name FROM city;",
		// 			want: `SELECT
		// 	ID,
		// 	Name
		// FROM
		// 	city;`,
		// 		},
		// 		{
		// 			name:  "member ident",
		// 			input: "select ci.ID, ci.Name, co.Code, co.Name from city ci join country co on ci.CountryCode = co.Code;",
		// 			want: `select
		// 	ci.ID,
		// 	ci.Name,
		// 	co.Code,
		// 	co.Name
		// from
		// 	city ci
		// join country co
		// 	on ci.CountryCode = co.Code;`,
		// 		},
		{
			name: "select",
			input: `select a, b as bb,c from table
join (select a * 2 as a from new_table) other
on table.a = other.a
where c is true
and b between 3 and 4
or d is 'blue'
limit 10`,
			want: `select
	a,
	b as bb,
	c
from
	table
join (
	select
		a * 2 as a
	from
		new_table
) other
	on table.a = other.a
where
	c is true
	and b between 3
	and 4
	or d is 'blue'
limit 10`,
		},
		{
			name: "joins",
			input: `select * from a
join b on a.one = b.one
left join c on c.two = a.two and c.three = a.three
right outer join d on d.three = a.three
cross join e on e.four = a.four
join f using (one, two, three)`,
			want: `select
	*
from
	a
join b
	on a.one = b.one
left join c
	on c.two = a.two
	and c.three = a.three
right outer join d
	on d.three = a.three
cross join e
	on e.four = a.four
join f using (
	one,
	two,
	three
)`,
		},
	}

	for _, tt := range testCase {
		t.Run(tt.name, func(t *testing.T) {
			// Open dummy file
			didOpenParams := lsp.DidOpenTextDocumentParams{
				TextDocument: lsp.TextDocumentItem{
					URI:        uri,
					LanguageID: "sql",
					Version:    0,
					Text:       tt.input,
				},
			}
			if err := tx.conn.Call(tx.ctx, "textDocument/didOpen", didOpenParams, nil); err != nil {
				t.Fatal("conn.Call textDocument/didOpen:", err)
			}
			tx.testFile(t, didOpenParams.TextDocument.URI, didOpenParams.TextDocument.Text)
			// Create completion params
			formattingParams := lsp.DocumentFormattingParams{
				TextDocument: lsp.TextDocumentIdentifier{
					URI: uri,
				},
				Options: lsp.FormattingOptions{
					TabSize:                0.0,
					InsertSpaces:           false,
					TrimTrailingWhitespace: false,
					InsertFinalNewline:     false,
					TrimFinalNewlines:      false,
				},
				WorkDoneProgressParams: lsp.WorkDoneProgressParams{
					WorkDoneToken: nil,
				},
			}

			var got []lsp.TextEdit
			if err := tx.conn.Call(tx.ctx, "textDocument/formatting", formattingParams, &got); err != nil {
				t.Fatal("conn.Call textDocument/formatting:", err)
			}
			if diff := cmp.Diff(tt.want, got[0].NewText); diff != "" {
				t.Errorf("unmatch (- want, + got):\n%s", diff)
			}
		})
	}
}
