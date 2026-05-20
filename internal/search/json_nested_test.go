package search

import (
	"testing"

	"github.com/sergiobonfiglio/tomagnet/internal/cardigann"
)

func TestParseJSONFlattensRowsAttributeFromEachParent(t *testing.T) {
	d := &cardigann.Definition{Config: map[string]string{}, Raw: map[string]any{"search": map[string]any{
		"rows":   map[string]any{"selector": "groups", "attribute": "torrents"},
		"fields": map[string]any{"title": map[string]any{"selector": "name"}},
	}}}
	got := parseJSON("idx", d, `{"groups":[{"torrents":[{"name":"A"}]},{"torrents":[{"name":"B"}]}]}`, 10)
	if len(got) != 2 || *got[0].Title != "A" || *got[1].Title != "B" {
		t.Fatalf("got %#v", got)
	}
}
