package database

import (
	"context"
	"log"
	"sync"
)

type Worker struct {
	dbRepo  DBRepository
	dbCache *DBCache

	done   chan struct{}
	update chan struct{}
	lock   sync.Mutex
}

func NewWorker() *Worker {
	return &Worker{
		done:   make(chan struct{}, 1),
		update: make(chan struct{}, 1),
	}
}

func (w *Worker) Cache() *DBCache {
	return w.dbCache
}

func (w *Worker) setCache(c *DBCache) {
	w.lock.Lock()
	defer w.lock.Unlock()
	w.dbCache = c
}

func (w *Worker) setColumnCache(col map[string][]*ColumnDesc) {
	w.lock.Lock()
	defer w.lock.Unlock()
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
				generator := NewDBCacheUpdater(w.dbRepo)
				col, err := generator.GenerateDBCacheSecondary(context.Background())
				if err != nil {
					log.Println(err)
				}
				w.setColumnCache(col)
				log.Println("db worker: Update db cache secondary complete")
			}
		}
	}()
}

func (w *Worker) Stop() {
	close(w.done)
}

func (w *Worker) ReCache(ctx context.Context, repo DBRepository) error {
	w.dbRepo = repo
	if err := w.updateAllCache(ctx); err != nil {
		return err
	}
	w.updateAdditionalCache()
	return nil
}

func (w *Worker) updateAllCache(ctx context.Context) error {
	generator := NewDBCacheUpdater(w.dbRepo)
	cache, err := generator.GenerateDBCachePrimary(ctx)
	if err != nil {
		return err
	}
	w.setCache(cache)
	log.Println("db worker: Update db cache primary complete")
	return nil
}

func (w *Worker) updateAdditionalCache() {
	w.update <- struct{}{}
}
