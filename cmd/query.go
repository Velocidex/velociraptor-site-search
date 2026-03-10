package main

import (
	"encoding/json"
	"fmt"

	"github.com/Velocidex/velociraptor-site-search/api"
	"github.com/alecthomas/kingpin"
	"github.com/blevesearch/bleve/v2"
)

var (
	query_command = app.Command(
		"query", "Query index.")

	query_command_path = query_command.Arg(
		"path", "The path for the index file",
	).Required().String()

	query_command_query = query_command.Arg(
		"query", "The query string",
	).Required().String()
)

func doQuery() {
	index, err := api.OpenIndex(*query_command_path)
	kingpin.FatalIfError(err, "Opening index")

	query := bleve.NewQueryStringQuery(*query_command_query)
	searchRequest := bleve.NewSearchRequest(query)
	searchRequest.Fields = []string{"title", "text", "url", "tags", "rank", "crumbs"}
	searchRequest.Highlight = bleve.NewHighlight()

	searchResult, err := index.Search(searchRequest)
	kingpin.FatalIfError(err, "Searching")

	serialized, err := json.MarshalIndent(searchResult, " ", " ")
	kingpin.FatalIfError(err, "Marshal")

	fmt.Println(string(serialized))
}

func init() {
	command_handlers = append(command_handlers, func(command string) bool {
		switch command {
		case query_command.FullCommand():
			doQuery()
		default:
			return false
		}
		return true
	})
}
