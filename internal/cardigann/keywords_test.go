package cardigann

import "testing"

func TestSearchRequestAppliesKeywordFilters(t *testing.T) {
	d := &Definition{Config: map[string]string{}, Raw: map[string]any{"search": map[string]any{
		"path":            "/search?q={{ .Keywords }}&encoded={{ .Query }}",
		"keywordsfilters": []any{map[string]any{"name": "trim"}},
	}}}
	got := SearchRequest(d, " dune two ")
	if got.Path != "/search?q=dune two&encoded=dune+two" {
		t.Fatalf("Path = %q", got.Path)
	}
}
