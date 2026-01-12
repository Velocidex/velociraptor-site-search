package api

import "github.com/blevesearch/bleve"

func SearchPage(index Index, query_str string, start, len int) (
	*bleve.SearchResult, error) {
	query := bleve.NewQueryStringQuery(query_str)
	searchRequest := bleve.NewSearchRequest(query)
	searchRequest.Fields = []string{"title", "text", "url", "tags", "rank"}
	searchRequest.Highlight = bleve.NewHighlight()
	searchRequest.From = start
	searchRequest.Size = len
	searchRequest.SortBy([]string{"-rank", "-_score"})

	return index.Search(searchRequest)
}
