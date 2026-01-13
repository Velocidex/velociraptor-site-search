package parser

import (
	"fmt"
	"path"
	"regexp"
	"strings"
)

const (
	TopLevelUrl = "https://docs.velociraptor.app/"
)

type ReplacementTable struct {
	in      string
	inRegex *regexp.Regexp
	out     string
}

type TagMap struct {
	components []string
	tag        string
	rank       int
}

func (self *TagMap) Match(components []string) bool {
	for idx, c := range components {
		if idx >= len(self.components) {
			return true
		}

		if c != self.components[idx] {
			return false
		}
	}
	return false
}

var (
	// Handle split lines and ref/relref
	refRegex = regexp.MustCompile(`(?ms){{<\s*(?:ref|relref)\s+"([^"]+)"\s*>}}`)

	noticeRegex = regexp.MustCompile(`{{% +notice +([a-z]+) +("[^"]+")? *%}}`)

	// Markdown img tag (handle split lines)
	imgRegex = regexp.MustCompile(`(?ms)(\!?\[[^]]*\]\()([^\)]+)\)`)

	// Raw img tag
	imgRegex2    = regexp.MustCompile(`(src=")([^"]+)"`)
	replacements = []ReplacementTable{
		{
			in:  `<pre><code class="language-yaml">`,
			out: "```yaml",
		}, {
			in:  `</code></pre>`,
			out: "```",
		}, {
			// Remove random shortcodes
			in:  `{{% [^%]+? %}}`,
			out: "",
		}, {
			// Remove random shortcodes
			in:  `{{< [^>]+? >}}`,
			out: "",
		}, {
			in:  "```vql",
			out: "```sql",
		},
	}

	tags = []TagMap{{
		components: []string{"docs"},
		tag:        "Docs",
		rank:       100,
	}, {
		components: []string{"vql_reference"},
		tag:        "VQLReference",
		rank:       50,
	}, {
		components: []string{"presentations"},
		tag:        "Presentations",
		rank:       20,
	}, {
		components: []string{"blog"},
		tag:        "BlogPost",
		rank:       40,
	}, {
		components: []string{"knowledge_base"},
		tag:        "KB",
		rank:       50,
	}, {
		components: []string{"artifact_references"},
		tag:        "Artifacts",
		rank:       20,
	}, {
		components: []string{"exchange", "artifacts"},
		tag:        "Exchange",
		rank:       60,
	}, {
		components: []string{"training", "playbooks"},
		tag:        "Playbooks",
		rank:       100,
	}}
)

// Take the MD file and remove hugo specific codes, rebase URLs,
// insert images etc. Returns a canonical MarkDown file.
func NormalizeText(path, in string) string {
	in = refRegex.ReplaceAllStringFunc(in, func(in string) string {
		m := refRegex.FindStringSubmatch(in)
		return CalculateURL(path, m[1])
	})

	in = imgRegex.ReplaceAllStringFunc(in, func(in string) string {
		m := imgRegex.FindStringSubmatch(in)
		return m[1] + CalculateURL(path, m[2]) + ")"
	})

	in = imgRegex2.ReplaceAllStringFunc(in, func(in string) string {
		m := imgRegex2.FindStringSubmatch(in)
		return m[1] + CalculateURL(path, m[2]) + `"`
	})

	in = noticeRegex.ReplaceAllStringFunc(in, func(in string) string {
		m := noticeRegex.FindStringSubmatch(in)
		if m[2] == "" {
			return fmt.Sprintf("### %s", strings.Trim(m[1], `"`))
		}
		return fmt.Sprintf("### %s: %s", m[1], strings.Trim(m[2], `"`))
	})

	in = replace(in)

	return strings.TrimSpace(in)
}

func replace(in string) string {
	for _, r := range replacements {
		if r.inRegex == nil {
			r.inRegex = regexp.MustCompile(r.in)
		}

		in = r.inRegex.ReplaceAllString(in, r.out)
	}

	return in
}

// Given a possibly relative link and a page path, calculate a fully
// qualified URL to access the source.
func CalculateURL(page_path, in string) string {
	if strings.Contains(in, "#") {
		DlvBreak()
	}

	// External URL
	if strings.HasPrefix(in, "http") {
		return in
	}

	// Absolute URL
	if strings.HasPrefix(in, "/") {
		return TopLevelUrl + strings.TrimPrefix(in, "/")
	}

	// Relative URL
	return CalculateURLFromPath(path.Join(getBaseDir(page_path), in))
}

func getBaseDir(in string) string {
	in = path.Clean(in)

	parts := strings.SplitN(in, "/content/", 2)
	if len(parts) >= 2 {
		in = "/" + parts[1]
	}

	if strings.HasSuffix(in, "_index.md") {
		return strings.ToLower(path.Dir(in)) + "/"
	}

	if strings.HasSuffix(in, ".md") {
		return strings.ToLower(strings.TrimSuffix(in, ".md")) + "/"
	}

	return in
}

// Given the path to the doc site's top level, return a full qualified
// URL to access the page.
func CalculateURLFromPath(in string) string {
	base_path := getBaseDir(in)
	return TopLevelUrl + strings.TrimPrefix(base_path, "/")
}

// Calculate a set of tags from the path. This embeds knowledge of the
// site structure by path to derive a set of tags. Tags are used to
// scope the search to more relevant data.
func GetTags(path, text string) (page_tags []string, rank int) {
	url := CalculateURLFromPath(path)
	components := strings.Split(strings.TrimPrefix(url, TopLevelUrl), "/")
	for _, tag := range tags {
		if tag.Match(components) {
			page_tags = append(page_tags, tag.tag)
			rank += tag.rank
		}
	}

	return page_tags, rank
}
