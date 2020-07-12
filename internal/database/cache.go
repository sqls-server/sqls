package database

import (
	"context"
	"sort"
	"strings"
)

func GenerateDBCache(ctx context.Context, db Database, defaultSchema string) (*DatabaseCache, error) {
	if err := db.Open(); err != nil {
		return nil, err
	}
	defer db.Close()

	// Create caches
	var err error
	dbCache := &DatabaseCache{}
	dbCache.defaultSchema, err = db.CurrentSchema(ctx)
	if err != nil {
		return nil, err
	}
	dbCache.Schemas, err = genSchmeaCache(ctx, db)
	if err != nil {
		return nil, err
	}
	dbCache.SchemaTables, err = db.SchemaTables(ctx)
	if err != nil {
		return nil, err
	}
	dbCache.ColumnsWithParent, err = genColumnsWithParentCache(ctx, db)
	if err != nil {
		return nil, err
	}
	return dbCache, nil
}

func genSchmeaCache(ctx context.Context, db Database) (map[string]string, error) {
	dbs, err := db.Schemas(ctx)
	if err != nil {
		return nil, err
	}
	databaseMap := map[string]string{}
	for _, db := range dbs {
		databaseMap[strings.ToUpper(db)] = db
	}
	return databaseMap, nil
}

func genColumnsWithParentCache(ctx context.Context, db Database) (map[string][]*ColumnDesc, error) {
	columnMap := map[string][]*ColumnDesc{}
	columnDescs, err := db.DescribeDatabaseTable(ctx)
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
