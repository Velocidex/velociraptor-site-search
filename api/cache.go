package api

import (
	"context"
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

func (self *IndexCache) Purge() error {
	self.mu.Lock()
	defer self.mu.Unlock()

	for k, idx := range self.cache {
		err := idx.Purge()
		if err != nil {
			return err
		}
		delete(self.cache, k)
	}
	return nil
}

func (self *IndexCache) Close() {
	self.cancel()
}

func (self *IndexCache) removeIdx(key string) {
	self.mu.Lock()
	defer self.mu.Unlock()

	delete(self.cache, key)
}

func (self *IndexCache) getKeyFromPath(path string) string {
	path = strings.TrimPrefix(path, `\\?\`)
	path, _ = filepath.Abs(path)
	return strings.ToLower(path)
}

func (self *IndexCache) newIndex(
	idx bleve.Index, key, path string) *Index {

	now := time.Now()
	res := &Index{
		path:      path,
		key:       key,
		idx:       idx,
		owner:     self,
		open_time: now,
		last_used: now,

		// Start off with one user.
		refs: 1,
	}

	// Keep checking if we can close this index - after sufficient
	// inactivity.
	go res.HouseKeep(self.ctx, self.houseKeepPeriod)
	return res
}

func (self *IndexCache) OpenIndex(path string) (*Index, error) {
	self.mu.Lock()
	defer self.mu.Unlock()

	key := self.getKeyFromPath(path)
	existing, pres := self.cache[key]
	if pres {
		existing.IncRef()
		return existing, nil
	}

	opts := make(map[string]interface{})
	opts["bolt_timeout"] = "10s"
	res_idx, err := bleve.OpenUsing(path, opts)
	if err != nil {
		return nil, err
	}

	res := self.newIndex(res_idx, key, res_idx.Name())
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

	res := self.newIndex(res_idx, key, res_idx.Name())
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
func PurgeCache() error {
	return cache.Purge()
}
