package database

import (
	"sort"
	"strings"
)

func GenerateDBCache(db Database, defaultSchema string) (*DatabaseCache, error) {
	if err := db.Open(); err != nil {
		return nil, err
	}
	defer db.Close()

	// Create caches
	var err error
	dbCache := &DatabaseCache{}
	dbCache.defaultSchema, err = db.CurrentSchema()
	if err != nil {
		return nil, err
	}
	dbCache.Schemas, err = genSchmeaCache(db)
	if err != nil {
		return nil, err
	}
	dbCache.SchemaTables, err = db.SchemaTables()
	if err != nil {
		return nil, err
	}
	dbCache.ColumnsWithParent, err = genColumnsWithParentCache(db)
	if err != nil {
		return nil, err
	}
	return dbCache, nil
}

func genSchmeaCache(db Database) (map[string]string, error) {
	dbs, err := db.Schemas()
	if err != nil {
		return nil, err
	}
	databaseMap := map[string]string{}
	for _, db := range dbs {
		databaseMap[strings.ToUpper(db)] = db
	}
	return databaseMap, nil
}

func genTableCache(db Database) (map[string]string, error) {
	tbls, err := db.Tables()
	if err != nil {
		return nil, err
	}
	tableMap := map[string]string{}
	for _, tbl := range tbls {
		tableMap[strings.ToUpper(tbl)] = tbl
	}
	return tableMap, nil
}

func genColumnCache(db Database, tbls map[string]string) (map[string][]*ColumnDesc, error) {
	columnMap := map[string][]*ColumnDesc{}
	for _, tbl := range tbls {
		columnDescs, err := db.DescribeTable(tbl)
		if err != nil {
			return nil, err
		}
		columnMap[strings.ToUpper(tbl)] = columnDescs
	}
	return columnMap, nil
}

func genColumnsWithParentCache(db Database) (map[string][]*ColumnDesc, error) {
	columnMap := map[string][]*ColumnDesc{}
	columnDescs, err := db.DescribeDatabaseTable()
	if err != nil {
		return nil, err
	}
	for _, desc := range columnDescs {
		key := desc.Schema + "\t" + desc.Table
		if _, ok := columnMap[key]; ok {
			columnMap[key] = append(columnMap[key], desc)
		} else {
			arr := []*ColumnDesc{desc}
			columnMap[key] = arr
		}
	}
	return columnMap, nil
}

type DatabaseCache struct {
	defaultSchema     string
	Schemas           map[string]string
	SchemaTables      map[string][]string
	ColumnsWithParent map[string][]*ColumnDesc
}

func (dc *DatabaseCache) Database(dbName string) (db string, ok bool) {
	db, ok = dc.Schemas[strings.ToUpper(dbName)]
	return
}

func (dc *DatabaseCache) SortedSchemas() []string {
	dbs := []string{}
	for _, db := range dc.Schemas {
		dbs = append(dbs, db)
	}
	sort.Strings(dbs)
	return dbs
}

func (dc *DatabaseCache) SortedTablesByDBName(dbName string) (tbls []string, ok bool) {
	tbls, ok = dc.SchemaTables[dbName]
	sort.Strings(tbls)
	return
}

func (dc *DatabaseCache) SortedTables() []string {
	tbls, _ := dc.SortedTablesByDBName(dc.defaultSchema)
	return tbls
}

func (dc *DatabaseCache) ColumnDescs(tableName string) (cols []*ColumnDesc, ok bool) {
	cols, ok = dc.ColumnsWithParent[columnDatabaseKey(dc.defaultSchema, tableName)]
	return
}

func (dc *DatabaseCache) ColumnDatabase(dbName, tableName string) (cols []*ColumnDesc, ok bool) {
	cols, ok = dc.ColumnsWithParent[columnDatabaseKey(dbName, tableName)]
	return
}

func (dc *DatabaseCache) Column(tableName, colName string) (*ColumnDesc, bool) {
	cols, ok := dc.ColumnsWithParent[columnDatabaseKey(dc.defaultSchema, tableName)]
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

func columnDatabaseKey(dbName, tableName string) string {
	return dbName + "\t" + tableName
}
