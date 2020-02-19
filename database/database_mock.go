package database

import (
	"context"
	"database/sql"
)

type MockDB struct {
	MockOpen          func() error
	MockClose         func() error
	MockDatabases     func() ([]string, error)
	MockTables        func() ([]string, error)
	MockDescribeTable func(tableName string) ([]*ColumnDesc, error)
	MockExecuteQuery  func(context.Context, string) (interface{}, error)
}

func (m *MockDB) Open() error {
	return m.MockOpen()
}

func (m *MockDB) Close() error {
	return m.MockClose()
}

func (m *MockDB) Databases() ([]string, error) {
	return m.MockDatabases()
}

func (m *MockDB) Tables() ([]string, error) {
	return m.MockTables()
}

func (m *MockDB) DescribeTable(tableName string) ([]*ColumnDesc, error) {
	return m.MockDescribeTable(tableName)
}

func (m *MockDB) ExecuteQuery(ctx context.Context, query string) (interface{}, error) {
	return m.MockExecuteQuery(ctx, query)
}

var dummyDatabases = []string{
	"information_schema",
	"mysql",
	"performance_schema",
	"sys",
	"world",
}
var dummyTables = []string{
	"city",
	"country",
	"countrylanguage",
}
var dummyCityColumns = []*ColumnDesc{
	&ColumnDesc{
		Name: "ID",
		Type: "int(11)",
		Null: "NO",
		Key:  "PRI",
		Default: sql.NullString{
			String: "<null>",
			Valid:  false,
		},
		Extra: "auto_increment",
	},
	&ColumnDesc{
		Name: "Name",
		Type: "char(35)",
		Null: "NO",
		Key:  "",
		Default: sql.NullString{
			String: "",
			Valid:  false,
		},
		Extra: "",
	},
	&ColumnDesc{
		Name: "CountryCode",
		Type: "char(3)",
		Null: "NO",
		Key:  "MUL",
		Default: sql.NullString{
			String: "",
			Valid:  false,
		},
		Extra: "",
	},
	&ColumnDesc{
		Name: "District",
		Type: "char(20)",
		Null: "NO",
		Key:  "",
		Default: sql.NullString{
			String: "",
			Valid:  false,
		},
		Extra: "",
	},
	&ColumnDesc{
		Name: "Population",
		Type: "int(11)",
		Null: "NO",
		Key:  "",
		Default: sql.NullString{
			String: "",
			Valid:  false,
		},
		Extra: "",
	},
}
var dummyCountryColumns = []*ColumnDesc{
	&ColumnDesc{
		Name: "Code",
		Type: "char(3)",
		Null: "NO",
		Key:  "PRI",
		Default: sql.NullString{
			String: "<null>",
			Valid:  false,
		},
		Extra: "auto_increment",
	},
	&ColumnDesc{
		Name: "Name",
		Type: "char(52)",
		Null: "NO",
		Key:  "",
		Default: sql.NullString{
			String: "",
			Valid:  false,
		},
		Extra: "",
	},
	&ColumnDesc{
		Name: "CountryCode",
		Type: "char(3)",
		Null: "NO",
		Key:  "",
		Default: sql.NullString{
			String: "",
			Valid:  false,
		},
		Extra: "",
	},
	&ColumnDesc{
		Name: "Continent",
		Type: "enum('Asia','Europe','North America','Africa','Oceania','Antarctica','South America')",
		Null: "NO",
		Key:  "",
		Default: sql.NullString{
			String: "Asia",
			Valid:  false,
		},
		Extra: "",
	},
	&ColumnDesc{
		Name: "Region",
		Type: "char(26)",
		Null: "NO",
		Key:  "",
		Default: sql.NullString{
			String: "",
			Valid:  false,
		},
		Extra: "",
	},
	&ColumnDesc{
		Name: "SurfaceArea",
		Type: "decimal(10,2)",
		Null: "NO",
		Key:  "",
		Default: sql.NullString{
			String: "0.00",
			Valid:  false,
		},
		Extra: "auto_increment",
	},
	&ColumnDesc{
		Name: "IndepYear",
		Type: "smallint(6)",
		Null: "YES",
		Key:  "",
		Default: sql.NullString{
			String: "0",
			Valid:  false,
		},
		Extra: "",
	},
	&ColumnDesc{
		Name: "LifeExpectancy",
		Type: "decimal(3,1)",
		Null: "YES",
		Key:  "",
		Default: sql.NullString{
			String: "<null>",
			Valid:  false,
		},
		Extra: "",
	},
	&ColumnDesc{
		Name: "GNP",
		Type: "decimal(10,2)",
		Null: "YES",
		Key:  "",
		Default: sql.NullString{
			String: "<null>",
			Valid:  false,
		},
		Extra: "",
	},
	&ColumnDesc{
		Name: "GNPOld",
		Type: "decimal(10,2)",
		Null: "YES",
		Key:  "",
		Default: sql.NullString{
			String: "<null>",
			Valid:  false,
		},
		Extra: "",
	},
	&ColumnDesc{
		Name: "LocalName",
		Type: "char(45)",
		Null: "NO",
		Key:  "",
		Default: sql.NullString{
			String: "",
			Valid:  false,
		},
		Extra: "",
	},
	&ColumnDesc{
		Name: "GovernmentForm",
		Type: "char(45)",
		Null: "NO",
		Key:  "",
		Default: sql.NullString{
			String: "",
			Valid:  false,
		},
		Extra: "",
	},
	&ColumnDesc{
		Name: "HeadOfState",
		Type: "char(60)",
		Null: "YES",
		Key:  "",
		Default: sql.NullString{
			String: "<null>",
			Valid:  false,
		},
		Extra: "",
	},
	&ColumnDesc{
		Name: "Capital",
		Type: "int(11)",
		Null: "YES",
		Key:  "",
		Default: sql.NullString{
			String: "<null>",
			Valid:  false,
		},
		Extra: "",
	},
	&ColumnDesc{
		Name: "Code2",
		Type: "char(2)",
		Null: "NO",
		Key:  "",
		Default: sql.NullString{
			String: "",
			Valid:  false,
		},
		Extra: "",
	},
}
var dummyCountryLanguageColumns = []*ColumnDesc{
	&ColumnDesc{
		Name: "CountryCode",
		Type: "char(3)",
		Null: "NO",
		Key:  "PRI",
		Default: sql.NullString{
			String: "",
			Valid:  false,
		},
		Extra: "",
	},
	&ColumnDesc{
		Name: "Language",
		Type: "char(30)",
		Null: "NO",
		Key:  "PRI",
		Default: sql.NullString{
			String: "",
			Valid:  false,
		},
		Extra: "",
	},
	&ColumnDesc{
		Name: "IsOfficial",
		Type: "enum('T','F')",
		Null: "NO",
		Key:  "F",
		Default: sql.NullString{
			String: "",
			Valid:  false,
		},
		Extra: "",
	},
	&ColumnDesc{
		Name: "Percentage",
		Type: "decimal(4,1)",
		Null: "NO",
		Key:  "",
		Default: sql.NullString{
			String: "0.0",
			Valid:  false,
		},
		Extra: "",
	},
}

func init() {
	Register("mock", func(connString string) Database {
		return &MockDB{
			MockOpen:      func() error { return nil },
			MockClose:     func() error { return nil },
			MockDatabases: func() ([]string, error) { return dummyDatabases, nil },
			MockTables:    func() ([]string, error) { return dummyTables, nil },
			MockDescribeTable: func(tableName string) ([]*ColumnDesc, error) {
				switch tableName {
				case "city":
					return dummyCityColumns, nil
				case "country":
					return dummyCountryColumns, nil
				case "countrylanguage":
					return dummyCountryLanguageColumns, nil
				}
				return nil, nil
			},
			MockExecuteQuery: func(ctx context.Context, query string) (interface{}, error) {
				return "dummy result", nil
			},
		}
	})
}
