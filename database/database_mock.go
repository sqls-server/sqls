package database

type MockDB struct {
	MockOpen          func() error
	MockClose         func() error
	MockDatabases     func() ([]string, error)
	MockTables        func() ([]string, error)
	MockDescribeTable func(tableName string) ([]*ColumnDesc, error)
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
