package search

import (
	"testing"

	"github.com/sergiobonfiglio/tomagnet/internal/cardigann"
)

func TestParseJSONRowsCountZeroReturnsNoResults(t *testing.T) {
	d := &cardigann.Definition{Config: map[string]string{}, Raw: map[string]any{"search": map[string]any{
		"rows":   map[string]any{"selector": "$", "count": map[string]any{"selector": "$[0].id"}},
		"fields": map[string]any{"title": map[string]any{"selector": "name"}},
	}}}
	got := parseJSON("idx", d, `[{"id":"0","name":"No results returned"}]`, 10)
	if len(got) != 0 {
		t.Fatalf("got %#v", got)
	}
}

func TestParseJSONRowsCountPositiveKeepsResults(t *testing.T) {
	d := &cardigann.Definition{Config: map[string]string{}, Raw: map[string]any{"search": map[string]any{
		"rows":   map[string]any{"selector": "data.items", "count": map[string]any{"selector": "data.total"}},
		"fields": map[string]any{"title": map[string]any{"selector": "name"}},
	}}}
	got := parseJSON("idx", d, `{"data":{"total":1,"items":[{"name":"Dune"}]}}`, 10)
	if len(got) != 1 || got[0].Title == nil || *got[0].Title != "Dune" {
		t.Fatalf("got %#v", got)
	}
}
