package search

import (
	"testing"

	"github.com/sergiobonfiglio/tomagnet/internal/cardigann"
)

func TestQueryValueUsesModeSpecificPrimaryField(t *testing.T) {
	opt := Options{Mode: "movie-search", IMDBID: "tt123", Query: "fallback"}
	d := &cardigann.Definition{Raw: map[string]any{"caps": map[string]any{"modes": map[string]any{
		"movie-search": map[string]any{"params": []any{map[string]any{"name": "imdbid"}, map[string]any{"name": "q"}}},
	}}}}
	if got := queryValue(d, opt); got != "tt123" {
		t.Fatalf("got %q", got)
	}
}
