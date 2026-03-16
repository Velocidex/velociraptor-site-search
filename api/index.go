package api

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/Velocidex/ordereddict"
	"github.com/blevesearch/bleve/v2"
	"github.com/blevesearch/bleve/v2/analysis/analyzer/custom"
	"github.com/blevesearch/bleve/v2/analysis/lang/en"
	"github.com/blevesearch/bleve/v2/analysis/token/lowercase"
	"github.com/blevesearch/bleve/v2/analysis/token/porter"
	"github.com/blevesearch/bleve/v2/analysis/tokenizer/unicode"
	"github.com/blevesearch/bleve/v2/mapping"
)

var (
	alreadyClosedErorr = errors.New("Index Already closed")
)

type Index struct {
	idx bleve.Index

	mu        sync.Mutex
	path      string
	key       string
	open_time time.Time
	last_used time.Time
	refs      int

	house_keep_cancel func()

	// Mark this index as already closed. This will ensure we dont
	// close it multiple times.
	closed bool

	owner *IndexCache
}

// Force close of the underlying index - rarely happens.
func (self *Index) Purge() error {
	self.mu.Lock()
	defer self.mu.Unlock()

	if !self.closed {
		err := self.idx.Close()
		if err != nil {
			return fmt.Errorf("While closing %v: %w",
				self.idx.Name(), err)
		}
		self.closed = true
	}
	return nil
}

func (self *Index) houseKeepOnce(period time.Duration) {
	self.mu.Lock()
	defer self.mu.Unlock()

	// Still in use
	if self.refs > 0 {
		return
	}

	expired := time.Now().Add(-period)
	if self.last_used.Before(expired) {
		// Close the underlying index if needed.
		if !self.closed {
			err := self.idx.Close()
			if err != nil {
				fmt.Printf("While closing %v: %v", self.idx.Name(), err)
				return
			}

			self.closed = true

			// Remove ourselves from the cache
			self.owner.removeIdx(self.key)

			// Stop the housekeep loop
			self.house_keep_cancel()
		}
	}
}

func (self *Index) HouseKeep(
	ctx context.Context, period time.Duration) {

	sub_ctx, cancel := context.WithCancel(ctx)
	self.house_keep_cancel = cancel

	for {
		select {
		case <-sub_ctx.Done():
			return

		case <-time.After(period):
			self.houseKeepOnce(period)
		}
	}

}

func (self *Index) Index(id string, data interface{}) error {
	self.mu.Lock()
	defer self.mu.Unlock()

	if self.closed {
		return alreadyClosedErorr
	}

	self.last_used = time.Now()

	return self.idx.Index(id, data)
}

func (self *Index) Fields() ([]string, error) {
	self.mu.Lock()
	defer self.mu.Unlock()

	if self.closed {
		return nil, alreadyClosedErorr
	}

	res, err := self.idx.Fields()
	if err != nil {
		return nil, err
	}

	sort.Strings(res)
	return res, nil
}

func (self *Index) Search(req *bleve.SearchRequest) (*bleve.SearchResult, error) {
	self.mu.Lock()
	defer self.mu.Unlock()

	if self.closed {
		return nil, alreadyClosedErorr
	}

	self.last_used = time.Now()
	return self.idx.Search(req)
}

func (self *Index) IncRef() {
	self.mu.Lock()
	defer self.mu.Unlock()

	self.refs++
	self.last_used = time.Now()
}

func (self *Index) Stats() IndexStat {
	self.mu.Lock()
	defer self.mu.Unlock()

	doc_count, _ := self.idx.DocCount()
	fields, _ := self.idx.Fields()
	sort.Strings(fields)

	now := time.Now()

	// This contains too much info for us.
	s := self.idx.StatsMap()
	item := ordereddict.NewDict()
	index_stats := s["index"]
	if index_stats != nil {
		idx_map, ok := index_stats.(map[string]interface{})
		if ok {
			item.Set("CurOnDiskBytes", idx_map["CurOnDiskBytes"])
		}
	}
	item.Set("Searches", s["searches"])
	item.Set("SearchTime", s["search_time"])

	return IndexStat{
		Path:        self.idx.Name(),
		Fields:      fields,
		Stats:       item,
		DocCount:    doc_count,
		LastUsedAgo: now.Sub(self.last_used).Round(time.Second).String(),
		OpenedAgo:   now.Sub(self.open_time).Round(time.Second).String(),
		RefCount:    self.refs,
	}
}

func (self *Index) Close() {
	self.mu.Lock()
	defer self.mu.Unlock()

	self.refs--
	self.last_used = time.Now()

	// The real closing happens in the cache reap cycle.
}

func BuildIndexMapping() (mapping.IndexMapping, error) {
	englishTextFieldMapping := bleve.NewTextFieldMapping()
	englishTextFieldMapping.Analyzer = "markdown"

	disabledMapping := bleve.NewDocumentDisabledMapping()

	pageMapping := bleve.NewDocumentMapping()
	pageMapping.AddFieldMappingsAt("text", englishTextFieldMapping)
	pageMapping.AddFieldMappingsAt("title", englishTextFieldMapping)
	pageMapping.AddSubDocumentMapping("crumb", disabledMapping)

	indexMapping := bleve.NewIndexMapping()
	indexMapping.AddDocumentMapping("page", pageMapping)
	indexMapping.TypeField = "type"
	indexMapping.DefaultAnalyzer = "markdown"

	// Same as en analyzer but includes markdown filter and html
	err := indexMapping.AddCustomAnalyzer("markdown",
		map[string]interface{}{
			"type":      custom.Name,
			"tokenizer": unicode.Name,
			"token_filters": []string{
				en.PossessiveName,
				lowercase.Name,
				en.StopName,
				porter.Name,
			},
			"char_filters": []string{
				MD_CharFilter_Name,
			},
		})
	return indexMapping, err
}

func NewIndex(path string, mapping mapping.IndexMapping) (*Index, error) {
	if mapping == nil {
		analyzer, err := BuildIndexMapping()
		if err != nil {
			return nil, err
		}
		return cache.NewIndex(path, analyzer)
	}
	return cache.NewIndex(path, mapping)
}

func OpenIndex(path string) (*Index, error) {
	return cache.OpenIndex(path)
}
