package api

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/blevesearch/bleve/v2"
	"github.com/blevesearch/bleve/v2/analysis/analyzer/custom"
	"github.com/blevesearch/bleve/v2/analysis/lang/en"
	"github.com/blevesearch/bleve/v2/analysis/token/lowercase"
	"github.com/blevesearch/bleve/v2/analysis/token/porter"
	"github.com/blevesearch/bleve/v2/analysis/tokenizer/unicode"
	"github.com/blevesearch/bleve/v2/mapping"
)

type Index struct {
	idx bleve.Index

	mu        sync.Mutex
	path      string
	last_used time.Time
	refs      int

	owner *IndexCache
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
		// Remove ourselves from the cache
		self.owner.removeIdx(self.path)

		// Close the underlying index
		self.idx.Close()
	}
}

func (self *Index) HouseKeep(
	ctx context.Context, period time.Duration) {
	for {
		select {
		case <-ctx.Done():
			return

		case <-time.After(period):
			self.houseKeepOnce(period)
		}
	}

}

func (self *Index) Index(id string, data interface{}) error {
	return self.idx.Index(id, data)
}

func (self *Index) Fields() ([]string, error) {
	return self.idx.Fields()
}

func (self *Index) Search(req *bleve.SearchRequest) (*bleve.SearchResult, error) {
	return self.idx.Search(req)
}

func (self *Index) IncRef() {
	self.mu.Lock()
	defer self.mu.Unlock()

	self.refs++
	self.last_used = time.Now()
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
	fmt.Printf("OpenIndex %v\n", path)

	return cache.OpenIndex(path)
}
