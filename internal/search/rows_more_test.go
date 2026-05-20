package search

import (
	"testing"

	"github.com/sergiobonfiglio/tomagnet/internal/cardigann"
)

func TestParseJSONMissingAttributeEqualsNoResults(t *testing.T) {
	d := &cardigann.Definition{Config: map[string]string{}, Raw: map[string]any{"search": map[string]any{
		"rows":   map[string]any{"selector": "data", "attribute": "items", "missingAttributeEqualsNoResults": true},
		"fields": map[string]any{"title": map[string]any{"selector": "name"}},
	}}}
	got := parseJSON("idx", d, `{"data":{"other":[{"name":"bad"}]}}`, 10)
	if len(got) != 0 {
		t.Fatalf("got %#v", got)
	}
}

func TestParseHTMLRowFilters(t *testing.T) {
	d := &cardigann.Definition{Config: map[string]string{}, Raw: map[string]any{"search": map[string]any{
		"rows":   map[string]any{"selector": ".item", "filters": []any{map[string]any{"name": "replace", "args": []any{"old", "new"}}}},
		"fields": map[string]any{"title": map[string]any{"selector": ".title"}},
	}}}
	got, err := parseHTML("idx", d, `<div class="item"><span class="title">old</span></div>`, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || *got[0].Title != "new" {
		t.Fatalf("got %#v", got)
	}
}
