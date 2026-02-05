package parser

import (
	"strings"
	"testing"

	"github.com/Velocidex/velociraptor-site-search/api"
	"github.com/alecthomas/assert"
)

type testCase struct {
	// Path to the page that has the link in it.
	path string

	// The link URL
	url      string
	expected string
}

var (
	urlTestCases = []testCase{{
		// Absolute ref.
		path:     "../velociraptor-docs/content/announcements/2021-artifact-contest/_index.md",
		url:      "/foo/bar/",
		expected: "https://docs.velociraptor.app/foo/bar/",
	}, {
		// Absolute ref to other.md.
		path:     "../velociraptor-docs/content/announcements/2021-artifact-contest/other.md",
		url:      "foo.png",
		expected: "https://docs.velociraptor.app/announcements/2021-artifact-contest/other/foo.png",
	}, {
		// Relative link
		path:     "../velociraptor-docs/content/announcements/2021-artifact-contest/_index.md",
		url:      "foo.png",
		expected: "https://docs.velociraptor.app/announcements/2021-artifact-contest/foo.png",
	}, {
		// Relative link with traversal
		path:     "../velociraptor-docs/content/announcements/2021-artifact-contest/_index.md",
		url:      "../../foo.png",
		expected: "https://docs.velociraptor.app/foo.png",
	}, {
		// External URL
		path:     "../velociraptor-docs/content/announcements/2021-artifact-contest/_index.md",
		url:      "https://www.google.com",
		expected: "https://www.google.com",
	}, {
		// Hash URL
		path:     "../velociraptor-docs/content/announcements/2021-artifact-contest/_index.md",
		url:      "#foobar",
		expected: "https://docs.velociraptor.app/announcements/2021-artifact-contest/#foobar",
	}}
)

// Resolves a URL on the
func TestCalculateURL(t *testing.T) {
	for _, tc := range urlTestCases {
		url := CalculateURL(tc.path, tc.url)
		assert.Equal(t, url, tc.expected)
	}
}

func TestCalculateURLFromPath(t *testing.T) {
	// Two types of markdown URLs
	assert.Equal(t, "https://docs.velociraptor.app/foo/",
		CalculateURLFromPath("../velociraptor-docs/content/foo/_index.md"))

	assert.Equal(t, "https://docs.velociraptor.app/foo/",
		CalculateURLFromPath("../velociraptor-docs/content/foo.md"))

	assert.Equal(t, "https://docs.velociraptor.app/exchange/artifacts/pages/exchange.windows.eventlogs.hayabusa.takajo/",
		CalculateURLFromPath("../velociraptor-docs/content/exchange/artifacts/pages/Exchange.Windows.EventLogs.Hayabusa.Takajo.md"))

}

func TestNormalizeText(t *testing.T) {
	for _, tc := range []struct {
		in, expected string
		path         string
	}{{
		in:       `Hello {{< ref "/images/image.png" >}}`,
		expected: `Hello https://docs.velociraptor.app/images/image.png`,
	}, {
		in: `or [VQL functions]({{< ref
"/vql_reference/" >}}).
`,
		expected: `or [VQL functions](https://docs.velociraptor.app/vql_reference/).`,
	}, {
		in:       `![image caption](image.png)`,
		expected: `![image caption](https://docs.velociraptor.app/foo/image.png)`,
	}, {
		in:       `![](../img/image.png)`,
		expected: `![](https://docs.velociraptor.app/img/image.png)`,
	}, {
		in:       "![](../img/image.png)\nxxx\n![](../img/image.png)",
		expected: "![](https://docs.velociraptor.app/img/image.png)\nxxx\n![](https://docs.velociraptor.app/img/image.png)",
	}, {
		in:       `![](../../img/1__rsKWeCDPrO9AffAuG2k__rA.png)`,
		expected: `![](https://docs.velociraptor.app/blog/img/1__rsKWeCDPrO9AffAuG2k__rA.png)`,
		path:     `../velociraptor-docs/content/blog/2019/2019-10-08_triage-with-velociraptor-pt-3-d6f63215f579/_index.md`,
	}, {
		in:       `<img src="image.png">`,
		expected: `<img src="https://docs.velociraptor.app/foo/image.png">`,
	}, {
		in:       `{{% notice note "This is important" %}}`,
		expected: `<velo-admonition adtype="note" caption="This is important">`,
	}, {
		in:       `{{% notice note %}}`,
		expected: `<velo-admonition adtype="note">`,
	}, {
		in:       `{{% /notice %}}`,
		expected: "</velo-admonition>",
	}, {
		in:       `{{% carousel %}}`,
		expected: ``,
	}, {
		// Broken across lines
		in: `![Server artifacts run with the permissions
of the launching user](server_artifacts_permissions.svg)`,
		expected: `![Server artifacts run with the permissions
of the launching user](https://docs.velociraptor.app/docs/artifacts/security/server_artifacts_permissions.svg)`,
		path: "../velociraptor-docs/content/docs/artifacts/security/_index.md",
	}, {
		// Hash URLs in MD links
		in:       `[Step 1: Download the Velociraptor binaries](#step-1-download-the-velociraptor-binaries)`,
		expected: `[Step 1: Download the Velociraptor binaries](https://docs.velociraptor.app/docs/artifacts/security/#step-1-download-the-velociraptor-binaries)`,
		path:     "../velociraptor-docs/content/docs/artifacts/security/_index.md",
	}, {
		in: `<!-- This is a comment

with several lines
 -->`,
		expected: "",
	}, {
		in: `
<!--
See this [blog post]({{< ref "/blog/html/2019/03/02/agentless_hunting_with_velociraptor.html" >}}) for details of how to deploy Velociraptor in agentless mode.
-->
`,
		expected: "",
	}} {
		path := tc.path
		if path == "" {
			path = "../velociraptor-docs/content/foo.md"
		}

		assert.Equal(t, tc.expected, NormalizeText(path, tc.in))
	}
}

func TestGetTags(t *testing.T) {
	for _, tc := range []struct {
		in, expected string
	}{{
		in:       "../velociraptor-docs/content/docs/foo.md",
		expected: "Docs",
	}, {
		in:       "../velociraptor-docs/content/vql_reference/foo.md",
		expected: "VQLReference",
	}, {
		in:       "../velociraptor-docs/content/presentations/foo.md",
		expected: "Presentations",
	}, {
		in:       "../velociraptor-docs/content/blog/foo.md",
		expected: "BlogPost",
	}, {
		in:       "../velociraptor-docs/content/knowledge_base/foo.md",
		expected: "KB",
	}, {
		in:       "../velociraptor-docs/content/artifact_references/foo.md",
		expected: "Artifacts",
	}, {
		in:       "../velociraptor-docs/content/exchange/artifacts/pages/foo.md",
		expected: "Exchange",
	}, {
		in:       "../velociraptor-docs/content/training/playbooks/foo.md",
		expected: "Playbooks",
	}} {
		tags, _ := GetTags(tc.in, "")
		assert.Equal(t, tc.expected, strings.Join(tags, ","))
	}
}

func TestBreadcrumbs(t *testing.T) {
	for _, tc := range []struct {
		in       string
		expected []api.BreadCrumb
	}{{
		in: "../velociraptor-docs/content/docs/troubleshooting/debugging/internals/go_profile/foo.md",
		expected: []api.BreadCrumb{
			{
				Url:  "https://docs.velociraptor.app/docs/",
				Name: "Docs",
			},
			{
				Url:  "https://docs.velociraptor.app/docs/troubleshooting/",
				Name: "Troubleshooting",
			},
			{
				Url:  "https://docs.velociraptor.app/docs/troubleshooting/debugging/",
				Name: "Debugging",
			},
			{
				Url:  "https://docs.velociraptor.app/docs/troubleshooting/debugging/internals/",
				Name: "Internals",
			},
			{
				Url:  "https://docs.velociraptor.app/docs/troubleshooting/debugging/internals/go_profile/",
				Name: "Go Profile",
			},
		},
	}} {
		tags := GetBreadCrumbs(tc.in)
		assert.Equal(t, tc.expected, tags)
	}
}
