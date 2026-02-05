package api

import (
	"encoding/json"
)

type BreadCrumb struct {
	Url  string `json:"url"`
	Name string `json:"name"`
}

type Page struct {
	Title       string   `json:"title"`
	Menutitle   string   `json:"menutitle"`
	Url         string   `json:"url"`
	Text        string   `json:"text"`
	Tags        []string `json:"tags"`
	BreadCrumbs string   `json:"crumbs"`
	Rank        int      `json:"rank"`
	Type        string   `json:"type"`
	Draft       bool     `json:"draft"`
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
