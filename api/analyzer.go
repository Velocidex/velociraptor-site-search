package api

import (
	"bytes"
	"regexp"

	"github.com/blevesearch/bleve/v2/analysis"
	"github.com/blevesearch/bleve/v2/registry"
)

const MD_CharFilter_Name = "markdown"

var (

	// Do not index in things that will be marked up by the MD
	// renderer. This causes problems with inserting the highlight
	// markers.
	markDownFilters = []*regexp.Regexp{
		// Markdown link: [text in link](http://www.example.com)
		regexp.MustCompile(`(?sm)\[[^\]]*\]\([^\(]+\)`),

		// Remove from code block
		regexp.MustCompile("(?sm)^```[^`]+```\n"),

		// Backtick: `Special Term`
		regexp.MustCompile("`[^`]+`"),

		// Remove HTML tags from indexing
		regexp.MustCompile("(?sm)<[^>]+>"),
	}
)

type MarkdownFilter struct {
	res []*regexp.Regexp
}

func (self *MarkdownFilter) Filter(input []byte) []byte {
	for _, r := range self.res {
		input = r.ReplaceAllFunc(input, func(in []byte) []byte {
			return bytes.Repeat([]byte(" "), len(in))
		})
	}
	return input
}

func CharFilterConstructor(
	config map[string]interface{}, cache *registry.Cache) (
	analysis.CharFilter, error) {

	return &MarkdownFilter{
		res: markDownFilters,
	}, nil
}

func init() {
	registry.RegisterCharFilter(MD_CharFilter_Name, CharFilterConstructor)
}
