package database

import (
	"context"
	"database/sql"
)

type MockDB struct {
	MockOpen           func() error
	MockClose          func() error
	MockDatabases      func() ([]string, error)
	MockDatabaseTables func() (map[string][]string, error)
	MockTables         func() ([]string, error)
	MockDescribeTable  func(string) ([]*ColumnDesc, error)
	MockExec           func(context.Context, string) (sql.Result, error)
	MockQuery          func(context.Context, string) (*sql.Rows, error)
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

func (m *MockDB) DatabaseTables() (map[string][]string, error) {
	return m.MockDatabaseTables()
}

func (m *MockDB) Tables() ([]string, error) {
	return m.MockTables()
}

func (m *MockDB) DescribeTable(tableName string) ([]*ColumnDesc, error) {
	return m.MockDescribeTable(tableName)
}

func (m *MockDB) Exec(ctx context.Context, query string) (sql.Result, error) {
	return m.MockExec(ctx, query)
}

func (m *MockDB) Query(ctx context.Context, query string) (*sql.Rows, error) {
	return m.MockQuery(ctx, query)
}

func (m *MockDB) SwitchDB(dbName string) error {
	return ErrNotImplementation
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
	{
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
	{
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
	{
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
	{
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
	{
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
	{
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
	{
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
	{
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
	{
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
	{
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
	{
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
	{
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
	{
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
	{
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
	{
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
	{
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
	{
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
	{
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
	{
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
	{
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
	{
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
	{
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
	{
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
	{
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

type MockResult struct {
	MockLastInsertID func() (int64, error)
	MockRowsAffected func() (int64, error)
}

func (m *MockResult) LastInsertId() (int64, error) {
	return m.MockLastInsertID()
}
func (m *MockResult) RowsAffected() (int64, error) {
	return m.MockRowsAffected()
}

func init() {
	Register("mock", func(cfg *Config) Database {
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
			MockExec: func(ctx context.Context, query string) (sql.Result, error) {
				return &MockResult{
					MockLastInsertID: func() (int64, error) { return 11, nil },
					MockRowsAffected: func() (int64, error) { return 22, nil },
				}, nil
			},
			MockQuery: func(ctx context.Context, query string) (*sql.Rows, error) {
				return &sql.Rows{}, nil
			},
		}
	})
}
