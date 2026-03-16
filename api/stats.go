package api

import (
	"sort"

	"github.com/Velocidex/ordereddict"
)

type IndexStat struct {
	Path        string
	Fields      []string
	Stats       *ordereddict.Dict
	DocCount    uint64
	LastUsedAgo string
	OpenedAgo   string
	RefCount    int
}

func GetStats() []IndexStat {
	return cache.GetStats()
}

func (self *IndexCache) GetStats() (res []IndexStat) {
	self.mu.Lock()
	defer self.mu.Unlock()

	for _, item := range self.cache {
		res = append(res, item.Stats())
	}

	sort.Slice(res, func(i, j int) bool {
		return res[i].Path < res[j].Path
	})

	return res
}
