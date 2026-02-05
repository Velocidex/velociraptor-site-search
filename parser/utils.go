package parser

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"regexp"

	"github.com/Velocidex/velociraptor-site-search/api"
	"github.com/goccy/go-yaml"
)

var (
	preambleRegex = regexp.MustCompile("(?ms)^---\n(.+?)\n---\n(.+)$")
)

func ParsePageFromFile(path string) (*api.Page, error) {
	res := api.NewPage()

	var tags []string
	tags, res.Rank = GetTags(path, res.Text)

	// We skip unranked pages.
	if res.Rank == 0 {
		return res, nil
	}

	fd, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	data, err := ioutil.ReadAll(fd)
	if err != nil {
		return nil, err
	}

	m := preambleRegex.FindStringSubmatch(string(data))
	if len(m) > 0 {
		yaml.Unmarshal([]byte(m[1]), res)
		res.Text = NormalizeText(path, m[2])
	} else {
		res.Text = NormalizeText(path, string(data))
	}

	if res.Draft {
		res.Rank = 0
		res.Text = ""
		return res, nil
	}

	// Ignore empty pages
	if res.Text == "" {
		res.Rank = 0
		return res, nil
	}

	res.Url = CalculateURLFromPath(path)
	if res.Title == "" {
		res.Title = res.Menutitle
	}
	res.Menutitle = ""
	res.Tags = append(tags, res.Tags...)
	res.Type = "page"

	serialized, err := json.Marshal(GetBreadCrumbs(path))
	if err == nil {
		res.BreadCrumbs = string(serialized)
	}

	//fmt.Printf("Url: %#v\nTags: %v\n", res.Url, res.Tags)

	return res, nil
}
