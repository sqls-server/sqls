package database

import (
	"context"
	"sort"
	"strings"
)

type DBCacheGenerator struct {
	Cache *DatabaseCache
	repo  DBRepository
}

func NewDBCacheUpdater(repo DBRepository) *DBCacheGenerator {
	return &DBCacheGenerator{
		repo: repo,
	}
}

func (u *DBCacheGenerator) GenerateDBCache(ctx context.Context, defaultSchema string) error {
	var err error
	dbCache := &DatabaseCache{}
	dbCache.defaultSchema, err = u.repo.CurrentSchema(ctx)
	if err != nil {
		return err
	}
	dbCache.Schemas, err = u.genSchmeaCache(ctx)
	if err != nil {
		return err
	}
	dbCache.SchemaTables, err = u.repo.SchemaTables(ctx)
	if err != nil {
		return err
	}
	dbCache.ColumnsWithParent, err = u.genColumnsWithParentCache(ctx)
	if err != nil {
		return err
	}
	u.Cache = dbCache
	return nil
}

func (u *DBCacheGenerator) genSchmeaCache(ctx context.Context) (map[string]string, error) {
	dbs, err := u.repo.Schemas(ctx)
	if err != nil {
		return nil, err
	}
	databaseMap := map[string]string{}
	for _, db := range dbs {
		databaseMap[strings.ToUpper(db)] = db
	}
	return databaseMap, nil
}

func (u *DBCacheGenerator) genColumnsWithParentCache(ctx context.Context) (map[string][]*ColumnDesc, error) {
	columnMap := map[string][]*ColumnDesc{}
	columnDescs, err := u.repo.DescribeDatabaseTable(ctx)
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
