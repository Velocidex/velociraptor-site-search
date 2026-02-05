package parser

import (
	"strings"

	"github.com/Velocidex/velociraptor-site-search/api"
)

// Give the filename to the target file generate a set of breadcrumb
// links to the pages.
func GetBreadCrumbs(in string) (res []api.BreadCrumb) {
	base_path := strings.TrimPrefix(getBaseDir(in), "/")
	parts := strings.Split(base_path, "/")

	url := TopLevelUrl

	for idx, part := range parts {
		// Drop empty components
		if part == "" || part == "pages" || part == "tips" {
			continue
		}

		// Drop the final link
		if idx == len(parts)-2 {
			break
		}

		url += part + "/"
		res = append(res, api.BreadCrumb{
			Url:  url,
			Name: strings.Title(strings.ReplaceAll(part, "_", " ")),
		})

		if part == "blog" || part == "exchange" {
			break
		}

	}

	return res
}
