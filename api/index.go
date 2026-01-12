package api

import (
	"github.com/blevesearch/bleve"
	"github.com/blevesearch/bleve/analysis/analyzer/custom"
	"github.com/blevesearch/bleve/analysis/lang/en"
	"github.com/blevesearch/bleve/analysis/token/lowercase"
	"github.com/blevesearch/bleve/analysis/token/porter"
	"github.com/blevesearch/bleve/analysis/tokenizer/unicode"
	"github.com/blevesearch/bleve/mapping"
)

type Index bleve.Index

func BuildIndexMapping() (mapping.IndexMapping, error) {
	englishTextFieldMapping := bleve.NewTextFieldMapping()
	englishTextFieldMapping.Analyzer = "markdown"

	pageMapping := bleve.NewDocumentMapping()
	pageMapping.AddFieldMappingsAt("text", englishTextFieldMapping)
	pageMapping.AddFieldMappingsAt("title", englishTextFieldMapping)

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

func NewIndex(path string) (Index, error) {
	analyzer, err := BuildIndexMapping()
	if err != nil {
		return nil, err
	}
	return bleve.New(path, analyzer)
}

func OpenIndex(path string) (Index, error) {
	return bleve.Open(path)
}
