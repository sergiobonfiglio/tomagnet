package search

import (
	"testing"

	"github.com/sergiobonfiglio/tomagnet/internal/cardigann"
)

func TestParseJSONSupportsParentSelectors(t *testing.T) {
	d := &cardigann.Definition{BaseURL: "https://idx.test", Config: map[string]string{}, Raw: map[string]any{"search": map[string]any{
		"rows": map[string]any{"selector": "data.movies", "attribute": "torrents", "multiple": true},
		"fields": map[string]any{
			"year":     map[string]any{"selector": "..year"},
			"title":    map[string]any{"text": "{{ .Result.year }} {{ .Result._quality }}"},
			"_quality": map[string]any{"selector": "quality"},
		},
	}}}
	got := parseJSON("idx", d, `{"data":{"movies":[{"year":"2024","torrents":[{"quality":"1080p"}]}]}}`, 10)
	if len(got) != 1 || *got[0].Title != "2024 1080p" {
		t.Fatalf("got %#v", got)
	}
}
