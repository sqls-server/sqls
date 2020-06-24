package database

import (
	"sort"
	"strings"
)

func GenerateDBCache(db Database) (*DatabaseCache, error) {
	dbCache := &DatabaseCache{}
	if err := db.Open(); err != nil {
		return nil, err
	}
	defer db.Close()

	dbs, err := db.Databases()
	if err != nil {
		return nil, err
	}
	databaseMap := map[string]string{}
	for _, db := range dbs {
		databaseMap[strings.ToUpper(db)] = db
	}
	dbCache.Databases = databaseMap

	dbTables, err := db.DatabaseTables()
	if err != nil {
		return nil, err
	}
	dbCache.DatabaseTables = dbTables

	tbls, err := db.Tables()
	if err != nil {
		return nil, err
	}
	tableMap := map[string]string{}
	for _, tbl := range tbls {
		tableMap[strings.ToUpper(tbl)] = tbl
	}
	dbCache.Tables = tableMap

	columnMap := map[string][]*ColumnDesc{}
	for _, tbl := range tbls {
		columnDescs, err := db.DescribeTable(tbl)
		if err != nil {
			return nil, err
		}
		columnMap[strings.ToUpper(tbl)] = columnDescs
	}
	dbCache.Columns = columnMap

	return dbCache, nil
}

type DatabaseCache struct {
	Databases      map[string]string
	DatabaseTables map[string][]string
	Tables         map[string]string
	Columns        map[string][]*ColumnDesc
}

func (dc *DatabaseCache) Database(dbName string) (db string, ok bool) {
	db, ok = dc.Databases[strings.ToUpper(dbName)]
	return
}

func (dc *DatabaseCache) SortedDatabases() []string {
	dbs := []string{}
	for _, db := range dc.Databases {
		dbs = append(dbs, db)
	}
	sort.Strings(dbs)
	return dbs
}

func (dc *DatabaseCache) SortedTablesByDBName(dbName string) (tbls []string, ok bool) {
	tbls, ok = dc.DatabaseTables[dbName]
	sort.Strings(tbls)
	return
}

func (dc *DatabaseCache) SortedTables() []string {
	tbls := []string{}
	for _, tbl := range dc.Tables {
		tbls = append(tbls, tbl)
	}
	sort.Strings(tbls)
	return tbls
}

func (dc *DatabaseCache) ColumnDescs(tableName string) (cols []*ColumnDesc, ok bool) {
	cols, ok = dc.Columns[strings.ToUpper(tableName)]
	return
}

func (dc *DatabaseCache) Column(dbName, colName string) (*ColumnDesc, bool) {
	cols, ok := dc.Columns[strings.ToUpper(dbName)]
	if !ok {
		return nil, false
	}
	for _, col := range cols {
		if strings.EqualFold(col.Name, colName) {
			return col, true
		}
	}
	return nil, false
}
