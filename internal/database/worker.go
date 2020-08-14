package database

import (
	"context"
	"database/sql"
	"log"
)

type Worker struct {
	dbCfg  *DBConfig
	dbConn *sql.DB

	dbCache *DatabaseCache

	done   chan struct{}
	update chan struct{}
}

func NewWorker() *Worker {
	return &Worker{
		done:   make(chan struct{}, 1),
		update: make(chan struct{}, 1),
	}
}

func (w *Worker) Cache() *DatabaseCache {
	return w.dbCache
}

func (w *Worker) setCache(c *DatabaseCache) {
	w.dbCache = c
}

func (w *Worker) setColumnCache(col map[string][]*ColumnDesc) {
	if w.dbCache != nil {
		w.dbCache.ColumnsWithParent = col
	}
}

func (w *Worker) Start() {
	go func() {
		log.Println("db worker: start")
		for {
			select {
			case <-w.done:
				log.Println("db worker: done")
				return
			case <-w.update:
				generator, err := w.newDBCacheGenerator()
				if err != nil {
					log.Println(err)
				}
				col, err := generator.GenerateDBCacheSecondary(context.Background())
				if err != nil {
					log.Println(err)
				}
				w.setColumnCache(col)
				log.Println("db worker: Update db chache secondary complete")
			}
		}
	}()
}

func (w *Worker) Stop() {
	close(w.done)
}

func (w *Worker) Update(ctx context.Context, dbCfg *DBConfig, dbConn *sql.DB) error {
	w.dbCfg = dbCfg
	w.dbConn = dbConn

	generator, err := w.newDBCacheGenerator()
	if err != nil {
		return err
	}
	if err := generator.GenerateDBCachePrimary(ctx); err != nil {
		return err
	}
	w.setCache(generator.Cache)
	log.Println("db worker: Update db chache primary complete")
	return nil
}

func (w *Worker) UpdateAsync(dbCfg *DBConfig, dbConn *sql.DB) {
	w.dbCfg = dbCfg
	w.dbConn = dbConn

	w.update <- struct{}{}
}

func (w *Worker) newDBCacheGenerator() (*DBCacheGenerator, error) {
	repo, err := CreateRepository(w.dbCfg.Driver, w.dbConn)
	if err != nil {
		return nil, err
	}
	generator := NewDBCacheUpdater(repo)
	return generator, nil
}
