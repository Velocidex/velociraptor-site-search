package api

import (
	"sort"
	"time"
)

type IndexStat struct {
	Path        string
	Fields      []string
	Stats       map[string]interface{}
	DocCount    uint64
	LastUsedAgo string
	RefCount    int
}

func GetStats() []IndexStat {
	return cache.GetStats()
}

func (self *IndexCache) GetStats() (res []IndexStat) {
	self.mu.Lock()
	defer self.mu.Unlock()

	now := time.Now()

	for _, k := range self.cache {
		k.mu.Lock()
		item := IndexStat{
			Path:        k.idx.Name(),
			Stats:       k.idx.StatsMap(),
			LastUsedAgo: now.Sub(k.last_used).Round(time.Second).String(),
			RefCount:    k.refs,
		}
		item.DocCount, _ = k.idx.DocCount()
		item.Fields, _ = k.Fields()
		k.mu.Unlock()

		res = append(res, item)
	}

	sort.Slice(res, func(i, j int) bool {
		return res[i].Path < res[j].Path
	})

	return res
}
