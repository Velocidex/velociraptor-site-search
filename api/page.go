package api

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"regexp"

	"github.com/Velocidex/velociraptor-site-search/parser"
	"github.com/goccy/go-yaml"
)

var (
	preambleRegex = regexp.MustCompile("(?ms)^---\n(.+?)\n---\n(.+)$")
)

type Page struct {
	Title     string   `json:"title"`
	Menutitle string   `json:"menutitle"`
	Url       string   `json:"url"`
	Text      string   `json:"text"`
	Tags      []string `json:"tags"`
	Rank      int      `json:"rank"`
	Type      string   `json:"type"`
	Draft     bool     `json:"draft"`
}

func (self *Page) ParsePageFromFile(path string) error {
	var tags []string
	tags, self.Rank = parser.GetTags(path, self.Text)

	// We skip unranked pages.
	if self.Rank == 0 {
		return nil
	}

	fd, err := os.Open(path)
	if err != nil {
		return err
	}

	data, err := ioutil.ReadAll(fd)
	if err != nil {
		return err
	}

	if self.Draft {
		self.Rank = 0
		return nil
	}

	m := preambleRegex.FindStringSubmatch(string(data))
	if len(m) > 0 {
		yaml.Unmarshal([]byte(m[1]), self)
		self.Text = parser.NormalizeText(path, m[2])
	} else {
		self.Text = parser.NormalizeText(path, string(data))
	}

	self.Url = parser.CalculateURLFromPath(path)
	if self.Title == "" {
		self.Title = self.Menutitle
	}
	self.Menutitle = ""
	self.Tags = append(tags, self.Tags...)
	self.Type = "page"

	fmt.Printf("Url: %#v\nTags: %v\n", self.Url, self.Tags)

	return nil
}

func NewPage() *Page {
	return &Page{}
}

func PageFromFields(fields map[string]interface{}) (*Page, error) {
	tags_any, pres := fields["tags"]
	if pres {
		tags_string, ok := tags_any.(string)
		if ok {
			fields["tags"] = []interface{}{tags_string}
		}
	}

	serialized, err := json.Marshal(fields)
	if err != nil {
		return nil, err
	}

	res := &Page{}
	err = json.Unmarshal(serialized, res)
	return res, err
}
