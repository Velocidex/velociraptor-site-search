package api

import (
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	"github.com/alecthomas/assert"
	"github.com/sebdah/goldie"
)

func TestIndexing(t *testing.T) {
	dir, err := ioutil.TempDir("", "index")
	assert.NoError(t, err)

	defer os.RemoveAll(dir)

	idx, err := NewIndex(dir)
	assert.NoError(t, err)

	page := NewPage()

	// Markdown page with some hits that should be excluded.
	page.Text = `
# Hello in heading

![No Hello in caption](http://no_ Hello _here)

` + "No `Hello` in backticks\n```yaml\n" + `
No Hello in code fences

` + "```\n\nBut hello in paragraph. No `tools:` No <Hello> in HTML"

	err = idx.Index("1", page)
	assert.NoError(t, err)

	resp, err := SearchPage(idx, "Hello tool", 0, 10)
	assert.NoError(t, err)

	assert.Equal(t, len(resp.Hits), 1)

	golden := ""
	for _, h := range resp.Hits[0].Locations["text"]["hello"] {
		highlight := page.Text[h.Start:h.End]
		golden += fmt.Sprintf("%d-%d : Highlight %s\n",
			h.Start, h.End, highlight)
	}

	for _, f := range resp.Hits[0].Fragments["text"] {
		golden += fmt.Sprintf("\nFragment:\n*********\n%v\n**********", f)
	}

	goldie.Assert(t, "TestIndexing", []byte(golden))
}
