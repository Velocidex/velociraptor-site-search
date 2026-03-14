package api

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/blevesearch/bleve/v2"
	"github.com/blevesearch/bleve/v2/mapping"
)

var (
	cache *IndexCache = NewIndexCache(time.Minute)
)

type IndexCache struct {
	mu sync.Mutex

	ctx             context.Context
	cancel          func()
	cache           map[string]*Index
	houseKeepPeriod time.Duration
}

func (self *IndexCache) Purge() {
	self.mu.Lock()
	defer self.mu.Unlock()

	for _, idx := range self.cache {
		idx.Purge()
	}
	self.cache = make(map[string]*Index)
}

func (self *IndexCache) Close() {
	self.cancel()
}

func (self *IndexCache) removeIdx(path string) {
	self.mu.Lock()
	defer self.mu.Unlock()

	delete(self.cache, self.getKeyFromPath(path))
}

func (self *IndexCache) getKeyFromPath(path string) string {
	path = strings.TrimPrefix(path, `\\?\`)
	path, _ = filepath.Abs(path)
	return strings.ToLower(path)
}

func (self *IndexCache) newIndex(
	idx bleve.Index, path string) *Index {
	res := &Index{
		path:      path,
		idx:       idx,
		owner:     self,
		last_used: time.Now(),
		refs:      1,
	}

	go res.HouseKeep(self.ctx, self.houseKeepPeriod)
	return res
}

func (self *IndexCache) OpenIndex(path string) (*Index, error) {
	self.mu.Lock()
	defer self.mu.Unlock()

	key := self.getKeyFromPath(path)
	fmt.Printf("Path %v key %v\n", path, key)
	existing, pres := self.cache[key]
	if pres {
		existing.IncRef()
		return existing, nil
	}

	res_idx, err := bleve.Open(path)
	if err != nil {
		return nil, err
	}

	res := self.newIndex(res_idx, res_idx.Name())
	self.cache[key] = res
	return res, nil

}

func (self *IndexCache) NewIndex(
	path string, mapping mapping.IndexMapping) (*Index, error) {

	self.mu.Lock()
	defer self.mu.Unlock()

	key := self.getKeyFromPath(path)

	existing, pres := self.cache[key]
	if pres {
		existing.IncRef()
		return existing, nil
	}

	res_idx, err := bleve.New(key, mapping)
	if err != nil {
		return nil, err
	}

	res := self.newIndex(res_idx, res_idx.Name())
	self.cache[key] = res
	return res, nil

}

func NewIndexCache(period time.Duration) *IndexCache {
	ctx, cancel := context.WithCancel(context.Background())

	res := &IndexCache{
		cancel:          cancel,
		ctx:             ctx,
		cache:           make(map[string]*Index),
		houseKeepPeriod: period,
	}
	return res
}

// Force the cache to purge immediately. Only used rarely.
func PurgeCache() {
	cache.Purge()
}
