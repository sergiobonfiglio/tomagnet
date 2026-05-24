package tomagnet

import (
	"strings"
	"testing"
)

func TestDecodeDefinition(t *testing.T) {
	def, err := DecodeDefinition(strings.NewReader(`id: custom
name: Custom
base_url: https://idx.test
search:
  path: /
  rows:
    selector: results
  fields:
    title:
      selector: title
`))
	if err != nil {
		t.Fatal(err)
	}
	if def.ID != "custom" || def.BaseURL != "https://idx.test" || def.Search.Rows.Selector != "results" || def.Search.Fields["title"].Selector != "title" {
		t.Fatalf("definition = %#v", def)
	}
}
