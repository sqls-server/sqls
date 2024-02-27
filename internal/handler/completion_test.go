package handler

import (
	"testing"

	"github.com/sqls-server/sqls/internal/config"
	"github.com/sqls-server/sqls/internal/database"
	"github.com/sqls-server/sqls/internal/lsp"
)

type completionTestCase struct {
	name  string
	input string
	line  int
	col   int
	want  []string
	bad   []string
}

var statementCase = []completionTestCase{
	{
		name:  "columns on multiple statement forcused first",
		input: "SELECT c. FROM city as c;SELECT c. FROM country as c;",
		line:  0,
		col:   9,
		want: []string{
			"ID",
			"Name",
			"CountryCode",
			"District",
			"Population",
		},
	},
	{
		name:  "columns with multiple statement forcused second",
		input: "SELECT c. FROM city as c;SELECT c. FROM country as c;",
		line:  0,
		col:   34,
		want: []string{
			"Code",
			"Name",
			"CountryCode",
			"Region",
			"SurfaceArea",
			"IndepYear",
			"LifeExpectancy",
			"GNP",
			"GNPOld",
			"LocalName",
			"GovernmentForm",
			"HeadOfState",
			"Capital",
			"Code2",
		},
	},
}

var selectExprCase = []completionTestCase{
	{
		name:  "table columns",
		input: "select  from city",
		line:  0,
		col:   7,
		want: []string{
			"ID",
			"Name",
			"CountryCode",
			"District",
			"Population",
			"city",
			"country",
			"countrylanguage",
		},
	},
	{
		name:  "quoted table columns",
		input: "select  from `city`",
		line:  0,
		col:   7,
		want: []string{
			"ID",
			"Name",
			"CountryCode",
			"District",
			"Population",
			"city",
			"country",
			"countrylanguage",
		},
	},
	{
		name:  "quoted table alias columns",
		input: "select c. from `city` as c",
		line:  0,
		col:   9,
		want: []string{
			"ID",
			"Name",
			"CountryCode",
			"District",
			"Population",
		},
	},
	{
		name:  "quoted table quoted alias columns",
		input: "select c. from `city` as `c`",
		line:  0,
		col:   9,
		want: []string{
			"ID",
			"Name",
			"CountryCode",
			"District",
			"Population",
		},
	},
	{
		name:  "quoted table quoted alias specified table columns",
		input: "select `c`. from `city` as `c`",
		line:  0,
		col:   11,
		want: []string{
			"ID",
			"Name",
			"CountryCode",
			"District",
			"Population",
		},
	},
	{
		name:  "quoted child columns",
		input: "select city.`Na from city",
		line:  0,
		col:   15,
		want: []string{
			"`Name`",
		},
	},
	{
		name:  "filtered table columns",
		input: "select Cou from city",
		line:  0,
		col:   10,
		want: []string{
			"CountryCode",
			"country",
			"countrylanguage",
		},
	},
	{
		name:  "filtered single quote table columns",
		input: "select `Cou from city",
		line:  0,
		col:   10,
		want: []string{
			"`CountryCode`",
			"`country`",
			"`countrylanguage`",
		},
	},
	{
		name:  "columns of table specifies database",
		input: "select  from world.city",
		line:  0,
		col:   7,
		want: []string{
			"ID",
			"Name",
			"CountryCode",
			"District",
			"Population",
			"city",
			"country",
			"countrylanguage",
		},
	},
	{
		name:  "columns of aliased table",
		input: "select  from city as c",
		line:  0,
		col:   7,
		want: []string{
			"ID",
			"Name",
			"CountryCode",
			"District",
			"Population",
			"c",
			"country",
			"countrylanguage",
		},
	},
	{
		name:  "columns of aliased table specifies database",
		input: "select  from world.city as c",
		line:  0,
		col:   7,
		want: []string{
			"ID",
			"Name",
			"CountryCode",
			"District",
			"Population",
			"c",
			"country",
			"countrylanguage",
		},
	},
	{
		name:  "columns of aliased table without as",
		input: "select  from city c",
		line:  0,
		col:   7,
		want: []string{
			"ID",
			"Name",
			"CountryCode",
			"District",
			"Population",
			"c",
			"country",
			"countrylanguage",
		},
	},
	{
		name:  "columns of before period table",
		input: "select c. from city as c",
		line:  0,
		col:   9,
		want: []string{
			"ID",
			"Name",
			"CountryCode",
			"District",
			"Population",
		},
	},
	{
		name:  "filtered columns of before period table",
		input: "select c.C from city as c",
		line:  0,
		col:   10,
		want: []string{
			"CountryCode",
		},
	},
	{
		name:  "columns of before period table closed",
		input: "select `c`. from city as c",
		line:  0,
		col:   11,
		want: []string{
			"ID",
			"Name",
			"CountryCode",
			"District",
			"Population",
		},
	},
	{
		name:  "identifier list",
		input: "select id,  from city",
		line:  0,
		col:   11,
		want: []string{
			"ID",
			"Name",
			"CountryCode",
			"District",
			"Population",
			"city",
			"country",
			"countrylanguage",
		},
	},
	{
		name:  "filtered identifier list",
		input: "select id, cou from city",
		line:  0,
		col:   14,
		want: []string{
			"CountryCode",
			"country",
			"countrylanguage",
		},
	},
	{
		name:  "comparison",
		input: "select 1 = cou from city",
		line:  0,
		col:   14,
		want: []string{
			"CountryCode",
			"country",
			"countrylanguage",
		},
	},
	{
		name:  "operator",
		input: "select 1 + cou from city",
		line:  0,
		col:   14,
		want: []string{
			"CountryCode",
			"country",
			"countrylanguage",
		},
	},
}

var tableReferenceCase = []completionTestCase{
	{
		name:  "from tables",
		input: "select CountryCode from ",
		line:  0,
		col:   24,
		want: []string{
			"city",
			"country",
			"countrylanguage",
			"information_schema",
			"mysql",
			"performance_schema",
			"sys",
			"world",
		},
	},
	{
		name:  "from quoted tables",
		input: "select CountryCode from `",
		line:  0,
		col:   25,
		want: []string{
			"`city`",
			"`country`",
			"`countrylanguage`",
			"`information_schema`",
			"`mysql`",
			"`performance_schema`",
			"`sys`",
			"`world`",
		},
	},
	{
		name:  "from filtered tables",
		input: "select CountryCode from co",
		line:  0,
		col:   26,
		want: []string{
			"country",
			"countrylanguage",
		},
	},
	{
		name:  "from quoted filtered tables",
		input: "select CountryCode from `co",
		line:  0,
		col:   26,
		want: []string{
			"`country`",
			"`countrylanguage`",
		},
	},
	{
		name:  "join tables",
		input: "select CountryCode from city join ",
		line:  0,
		col:   34,
		want: []string{
			"city",
			"country",
			"countrylanguage",
		},
	},
	{
		name:  "join filtered tables",
		input: "select CountryCode from city join co",
		line:  0,
		col:   36,
		want: []string{
			"country",
			"countrylanguage",
		},
	},
	{
		name:  "left join tables",
		input: "select CountryCode from city left join ",
		line:  0,
		col:   39,
		want: []string{
			"city",
			"country",
			"countrylanguage",
		},
	},
	{
		name:  "left outer join tables",
		input: "select CountryCode from city left outer join ",
		line:  0,
		col:   45,
		want: []string{
			"city",
			"country",
			"countrylanguage",
		},
	},
	{
		name:  "insert tables",
		input: "INSERT INTO ",
		line:  0,
		col:   12,
		want: []string{
			"city",
			"country",
			"countrylanguage",
		},
	},
	{
		name:  "insert quoted tables",
		input: "INSERT INTO `",
		line:  0,
		col:   13,
		want: []string{
			"`city`",
			"`country`",
			"`countrylanguage`",
		},
	},
	{
		name:  "insert filtered tables",
		input: "INSERT INTO co",
		line:  0,
		col:   12,
		want: []string{
			"country",
			"countrylanguage",
		},
	},
	{
		name:  "insert columns",
		input: "INSERT INTO city (",
		line:  0,
		col:   18,
		want: []string{
			"ID",
			"Name",
			"CountryCode",
			"District",
			"Population",
		},
		bad: []string{
			"city",
			"country",
			"countrylanguage",
		},
	},
	{
		name:  "insert filtered columns",
		input: "INSERT INTO city (cou",
		line:  0,
		col:   21,
		want: []string{
			"CountryCode",
		},
	},
	{
		name:  "insert identifier list",
		input: "INSERT INTO city (id, cou",
		line:  0,
		col:   25,
		want: []string{
			"CountryCode",
		},
	},
	{
		name:  "update tables",
		input: "UPDATE ",
		line:  0,
		col:   7,
		want: []string{
			"city",
			"country",
			"countrylanguage",
		},
	},
	{
		name:  "update filtered tables",
		input: "UPDATE co",
		line:  0,
		col:   9,
		want: []string{
			"country",
			"countrylanguage",
		},
	},
	{
		name:  "update columns",
		input: "UPDATE city SET ",
		line:  0,
		col:   16,
		want: []string{
			"ID",
			"Name",
			"CountryCode",
			"District",
			"Population",
		},
	},
	{
		name:  "update filtered columns",
		input: "UPDATE city SET cou",
		line:  0,
		col:   19,
		want: []string{
			"CountryCode",
		},
	},
	{
		name:  "update identiger list",
		input: "UPDATE city SET CountryCode=12, Na",
		line:  0,
		col:   34,
		want: []string{
			"Name",
		},
	},
	{
		name:  "delete tables",
		input: "DELETE FROM ",
		line:  0,
		col:   12,
		want: []string{
			"city",
			"country",
			"countrylanguage",
		},
	},
	{
		name:  "delete filtered tables",
		input: "DELETE FROM co",
		line:  0,
		col:   14,
		want: []string{
			"country",
			"countrylanguage",
		},
	},
}

var whereCondition = []completionTestCase{
	{
		name:  "where columns",
		input: "select * from city where ",
		line:  0,
		col:   25,
		want: []string{
			"ID",
			"Name",
			"CountryCode",
			"District",
			"Population",
		},
	},
	{
		name:  "where columns of specified table",
		input: "select * from city where city.",
		line:  0,
		col:   30,
		want: []string{
			"ID",
			"Name",
			"CountryCode",
			"District",
			"Population",
		},
	},
	{
		name:  "where columns in left of comparison",
		input: "select * from city where  = ID",
		line:  0,
		col:   25,
		want: []string{
			"ID",
			"Name",
			"CountryCode",
			"District",
			"Population",
		},
	},
	{
		name:  "where columns in right of comparison",
		input: "select * from city where ID = ",
		line:  0,
		col:   30,
		want: []string{
			"ID",
			"Name",
			"CountryCode",
			"District",
			"Population",
		},
	},
	{
		name:  "where columns of specified table in left of comparison",
		input: "select * from city where city. = city.ID",
		line:  0,
		col:   30,
		want: []string{
			"ID",
			"Name",
			"CountryCode",
			"District",
			"Population",
		},
	},
	{
		name:  "where columns of specified table in right of comparison",
		input: "select * from city where city.ID = city.",
		line:  0,
		col:   40,
		want: []string{
			"ID",
			"Name",
			"CountryCode",
			"District",
			"Population",
		},
	},
	{
		name:  "join on columns",
		input: "select * from city left join country on ",
		line:  0,
		col:   40,
		want: []string{
			"Code",
			"Name",
			"CountryCode",
			"Continent",
			"Region",
			"SurfaceArea",
			"IndepYear",
			"LifeExpectancy",
			"GNP",
			"GNPOld",
			"LocalName",
			"GovernmentForm",
			"HeadOfState",
			"Capital",
			"Code2",
		},
	},
	{
		name:  "join on filtered columns",
		input: "select * from city left join country on co",
		line:  0,
		col:   52,
		want: []string{
			"Code",
			"Continent",
			"Code2",
		},
	},
}

var colNameCase = []completionTestCase{
	{
		name:  "ORDER BY columns",
		input: "SELECT ID, Name FROM city ORDER BY ",
		line:  0,
		col:   35,
		want: []string{
			"ID",
			"Name",
			"CountryCode",
			"District",
			"Population",
			"city",
			"country",
			"countrylanguage",
		},
	},
	{
		name:  "GROUP BY columns",
		input: "SELECT CountryCode, COUNT(*) FROM city GROUP BY ",
		line:  0,
		col:   48,
		want: []string{
			"ID",
			"Name",
			"CountryCode",
			"District",
			"Population",
			"city",
			"country",
			"countrylanguage",
		},
	},
}

var caseValueCase = []completionTestCase{
	{
		name:  "select case after case",
		input: "SELECT `Language`, CASE Is WHEN 'T' THEN 'official' WHEN 'F' THEN 'unofficial' END AS is_official FROM countrylanguage;",
		line:  0,
		col:   26,
		want: []string{
			"IsOfficial",
		},
	},
	{
		name:  "select case after when",
		input: "SELECT `Language`, CASE IsOfficial WHEN Is THEN 'official' WHEN 'F' THEN 'unofficial' END AS is_official FROM countrylanguage;",
		line:  0,
		col:   42,
		want: []string{
			"IsOfficial",
		},
	},
	{
		name:  "select case after then",
		input: "SELECT `Language`, CASE IsOfficial WHEN 'T' THEN Is WHEN 'F' THEN 'unofficial' END AS is_official FROM countrylanguage;",
		line:  0,
		col:   51,
		want: []string{
			"IsOfficial",
		},
	},
	{
		name:  "select case identifier list",
		input: "SELECT `Language`, CASE IsOfficial WHEN 'T' THEN Is WHEN 'F' THEN 'unofficial' END AS is_official, P FROM countrylanguage;",
		line:  0,
		col:   100,
		want: []string{
			"Percentage",
		},
	},
}

var subQueryCase = []completionTestCase{
	{
		name:  "in subquery table columns",
		input: "SELECT * FROM (SELECT Cou FROM city)",
		line:  0,
		col:   25,
		want: []string{
			"CountryCode",
			"country",
			"countrylanguage",
		},
	},
	{
		name:  "in subquery table references",
		input: "SELECT * FROM (SELECT * FROM ",
		line:  0,
		col:   29,
		want: []string{
			"city",
			"country",
			"countrylanguage",
		},
	},
	{
		name:  "in subquery filtered table references",
		input: "SELECT * FROM (SELECT * FROM co",
		line:  0,
		col:   29,
		want: []string{
			"country",
			"countrylanguage",
		},
	},
	{
		name:  "subquery columns",
		input: "SELECT  FROM (SELECT ID as city_id, Name as city_name FROM city) as t",
		line:  0,
		col:   7,
		want: []string{
			"t",
			"city_id",
			"city_name",
		},
	},
	{
		name:  "quoted subquery parent",
		input: "SELECT ` FROM (SELECT ID as city_id, Name as city_name FROM city) as t",
		line:  0,
		col:   8,
		want: []string{
			"`t`",
		},
	},
	{
		name:  "subquery parent table columns",
		input: "SELECT t. FROM (SELECT ID as city_id, Name as city_name FROM city) as t",
		line:  0,
		col:   9,
		want: []string{
			"city_id",
			"city_name",
		},
	},
	{
		name:  "subquery parent table quoted columns",
		input: "SELECT t.` FROM (SELECT ID as city_id, Name as city_name FROM city) as t",
		line:  0,
		col:   10,
		want: []string{
			"`city_id`",
			"`city_name`",
		},
	},
	{
		name:  "columns of multiple subquery",
		input: "SELECT  FROM (SELECT Name as city_name FROM city) AS sub1, (SELECT LocalName as country_name FROM country) AS sub2 limit 1",
		line:  0,
		col:   7,
		want: []string{
			"sub1",
			"sub2",
			"city_name",
			"country_name",
		},
	},
	{
		name:  "subquery asterisk columns",
		input: "SELECT  FROM (SELECT * FROM city) as t",
		line:  0,
		col:   7,
		want: []string{
			"ID",
			"Name",
			"CountryCode",
			"District",
			"Population",
		},
	},
	{
		name:  "subquery parent asterisk table columns",
		input: "SELECT t. FROM (SELECT * FROM city) as t",
		line:  0,
		col:   9,
		want: []string{
			"ID",
			"Name",
			"CountryCode",
			"District",
			"Population",
		},
	},
}
var joinClauseCase = []completionTestCase{
	{
		name:  "join tables",
		input: "select CountryCode from city join ",
		line:  0,
		col:   34,
		want: []string{
			"country c1 ON c1.Code = city.CountryCode",
			"city",
			"country",
			"countrylanguage",
		},
	},
	{
		name:  "join tables with reference",
		input: "select c.CountryCode from city c join ",
		line:  0,
		col:   38,
		want: []string{
			"country c1 ON c1.Code = c.CountryCode",
			"city",
			"country",
			"countrylanguage",
		},
	},
	{
		name:  "join filtered tables",
		input: "select CountryCode from city join co",
		line:  0,
		col:   36,
		want: []string{
			"country c1 ON c1.Code = city.CountryCode",
			"country",
			"countrylanguage",
		},
	},
	{
		name:  "join filtered tables with reference",
		input: "select c.CountryCode from city c join co",
		line:  0,
		col:   40,
		want: []string{
			"country c1 ON c1.Code = c.CountryCode",
			"country",
			"countrylanguage",
		},
	},
	{
		name:  "left join tables",
		input: "select CountryCode from city left join ",
		line:  0,
		col:   39,
		want: []string{
			"country c1 ON c1.Code = city.CountryCode",
			"city",
			"country",
			"countrylanguage",
		},
	},
	{
		name:  "left join tables with reference",
		input: "select c.CountryCode from city c left join ",
		line:  0,
		col:   43,
		want: []string{
			"country c1 ON c1.Code = c.CountryCode",
			"city",
			"country",
			"countrylanguage",
		},
	},
	{
		name:  "left outer join tables",
		input: "select CountryCode from city left outer join ",
		line:  0,
		col:   45,
		want: []string{
			"country c1 ON c1.Code = city.CountryCode",
			"city",
			"country",
			"countrylanguage",
		},
	},
	{
		name:  "left outer join tables with reference",
		input: "select c.CountryCode from city c left outer join ",
		line:  0,
		col:   49,
		want: []string{
			"country c1 ON c1.Code = c.CountryCode",
			"city",
			"country",
			"countrylanguage",
		},
	},
}

var joinConditionCase = []completionTestCase{
	{
		name:  "join on columns",
		input: "select * from city left join country on ",
		line:  0,
		col:   40,
		want: []string{
			"country.Code = city.CountryCode",
			"Code",
			"Name",
			"CountryCode",
			"Continent",
			"Region",
			"SurfaceArea",
			"IndepYear",
			"LifeExpectancy",
			"GNP",
			"GNPOld",
			"LocalName",
			"GovernmentForm",
			"HeadOfState",
			"Capital",
			"Code2",
		},
	},
	{
		name:  "join on columns with reference",
		input: "select * from city c left join country co on ",
		line:  0,
		col:   45,
		want: []string{
			"co.Code = c.CountryCode",
			"Code",
			"Name",
			"CountryCode",
			"Continent",
			"Region",
			"SurfaceArea",
			"IndepYear",
			"LifeExpectancy",
			"GNP",
			"GNPOld",
			"LocalName",
			"GovernmentForm",
			"HeadOfState",
			"Capital",
			"Code2",
		},
	},
}

var multiJoin = []completionTestCase{
	{
		name: "join tables",
		input: `select * from city c
                      join country c1 on c1.Code = c.CountryCode
                      join `,
		line: 2,
		col:  27,
		want: []string{
			"countrylanguage c2 ON c2.CountryCode = c1.Code",
			"city",
			"country",
			"countrylanguage",
		},
	},
	{
		name: "join tables start",
		input: `select * from city c
                     join country c1 on c1.Code = c.CountryCode
                     join co`,
		line: 2,
		col:  28,
		want: []string{
			"countrylanguage c2 ON c2.CountryCode = c1.Code",
			"country",
			"countrylanguage",
		},
	},
	{
		name: "join tables on",
		input: `select * from city c
                     join country c1 on c1.Code = c.CountryCode
                     join countrylanguage c2 ON `,
		line: 2,
		col:  48,
		want: []string{
			"c2.CountryCode = c1.Code",
		},
	},
	{
		name: "join tables prev line",
		input: `select * from city c
                      join 
                      join countrylanguage c2 ON c2.CountryCode = c1.Code`,
		line: 1,
		col:  27,
		want: []string{
			"country c1 ON c1.Code = c.CountryCode",
		},
		bad: []string{
			"country c1 ON c1.Code = c2.CountryCode",
		},
	},
}

var joinSnippetCompletionCase = []completionTestCase{
	{
		name:  "join snippet alias",
		input: "select CountryCode from city c join country c1 on c1.Code = c.CountryCode",
		line:  0,
		col:   44,
		bad: []string{
			"country c1 ON c1.Code = city.CountryCode",
			"country",
		},
	},
	{
		name: "join snippet alias multi",
		input: `select CountryCode from city c join country c1 on c1.Code = c.CountryCode
				join countrylanguage c2 on c2.CountryCode = c1.Code`,
		line: 1,
		col:  25,
		bad: []string{
			"country c1 ON c1.Code = city.CountryCode",
			"country",
			"countrylanguage",
		},
	},
}

func TestCompleteMain(t *testing.T) {
	tx := newTestContext()
	tx.initServer(t)
	defer tx.tearDown()

	cfg := &config.Config{
		Connections: []*database.DBConfig{
			{Driver: "mock"},
		},
	}
	tx.addWorkspaceConfig(t, cfg)

	testcaseMap := map[string][]completionTestCase{
		"statement":       statementCase,
		"select expr":     selectExprCase,
		"table reference": tableReferenceCase,
		"col name":        colNameCase,
		"case value":      caseValueCase,
		"subquery":        subQueryCase,
	}

	for k, v := range testcaseMap {
		for _, tt := range v {
			t.Run(k+" "+tt.name, func(t *testing.T) {
				tx.textDocumentDidOpen(t, testFileURI, tt.input)

				completionParams := lsp.CompletionParams{
					TextDocumentPositionParams: lsp.TextDocumentPositionParams{
						TextDocument: lsp.TextDocumentIdentifier{
							URI: testFileURI,
						},
						Position: lsp.Position{
							Line:      tt.line,
							Character: tt.col,
						},
					},
					CompletionContext: lsp.CompletionContext{
						TriggerKind:      0,
						TriggerCharacter: nil,
					},
				}

				var got []lsp.CompletionItem
				if err := tx.conn.Call(tx.ctx, "textDocument/completion", completionParams, &got); err != nil {
					t.Fatal("conn.Call textDocument/completion:", err)
				}
				testCompletionItem(t, tt.want, tt.bad, got)
			})
		}
	}
}

func TestCompleteJoin(t *testing.T) {
	tx := newTestContext()
	tx.initServer(t)
	defer tx.tearDown()

	cfg := &config.Config{
		Connections: []*database.DBConfig{
			{Driver: "mock"},
		},
	}
	tx.addWorkspaceConfig(t, cfg)

	testcaseMap := map[string][]completionTestCase{
		"join clause":    joinClauseCase,
		"join condition": joinConditionCase,
		"multi-join":     multiJoin,
		"snippet":        joinSnippetCompletionCase,
	}

	for k, v := range testcaseMap {
		for _, tt := range v {
			t.Run(k+" "+tt.name, func(t *testing.T) {
				tx.textDocumentDidOpen(t, testFileURI, tt.input)

				completionParams := lsp.CompletionParams{
					TextDocumentPositionParams: lsp.TextDocumentPositionParams{
						TextDocument: lsp.TextDocumentIdentifier{
							URI: testFileURI,
						},
						Position: lsp.Position{
							Line:      tt.line,
							Character: tt.col,
						},
					},
					CompletionContext: lsp.CompletionContext{
						TriggerKind:      0,
						TriggerCharacter: nil,
					},
				}

				var got []lsp.CompletionItem
				if err := tx.conn.Call(tx.ctx, "textDocument/completion", completionParams, &got); err != nil {
					t.Fatal("conn.Call textDocument/completion:", err)
				}
				testCompletionItem(t, tt.want, tt.bad, got)
			})
		}
	}
}

func TestCompleteNoneDBConnection(t *testing.T) {
	tx := newTestContext()
	tx.initServer(t)
	defer tx.tearDown()

	cfg := &config.Config{
		Connections: []*database.DBConfig{},
	}
	tx.addWorkspaceConfig(t, cfg)

	testcaseMap := map[string][]completionTestCase{
		"statement":       statementCase,
		"select expr":     selectExprCase,
		"table reference": tableReferenceCase,
		"col name":        colNameCase,
		"case value":      caseValueCase,
		"subquery":        subQueryCase,
	}

	for k, v := range testcaseMap {
		for _, tt := range v {
			t.Run(k+" "+tt.name, func(t *testing.T) {
				tx.textDocumentDidOpen(t, testFileURI, tt.input)

				completionParams := lsp.CompletionParams{
					TextDocumentPositionParams: lsp.TextDocumentPositionParams{
						TextDocument: lsp.TextDocumentIdentifier{
							URI: testFileURI,
						},
						Position: lsp.Position{
							Line:      tt.line,
							Character: tt.col,
						},
					},
					CompletionContext: lsp.CompletionContext{
						TriggerKind:      0,
						TriggerCharacter: nil,
					},
				}

				// Without a DB connection, it is not possible to provide functions using the DB connection, so just make sure that no errors occur.
				var got []lsp.CompletionItem
				if err := tx.conn.Call(tx.ctx, "textDocument/completion", completionParams, &got); err != nil {
					t.Fatal("conn.Call textDocument/completion:", err)
				}
			})
		}
	}
}

func testCompletionItem(t *testing.T, expectLabels []string, badLabels []string, gotItems []lsp.CompletionItem) {
	t.Helper()

	itemMap := map[string]struct{}{}
	for _, item := range gotItems {
		itemMap[item.Label] = struct{}{}
	}

	for _, el := range expectLabels {
		_, ok := itemMap[el]
		if !ok {
			t.Errorf("expected to be included in the results, expect candidate %q", el)
		}
	}

	for _, el := range badLabels {
		_, ok := itemMap[el]
		if ok {
			t.Errorf("should not be included in the results, got candidate %q", el)
		}
	}
}
