package cardigann

import "testing"

func TestSearchRequestSelectsPathByCategory(t *testing.T) {
	d := &Definition{Config: map[string]string{}, Raw: map[string]any{"search": map[string]any{"paths": []any{
		map[string]any{"path": "/movies", "categories": []any{"2000"}},
		map[string]any{"path": "/tv", "categories": []any{"5000"}},
	}}}}
	got := SearchRequestWithOptions(d, SearchOptions{Keywords: "dune", Categories: []string{"5000"}})
	if got.Path != "/tv" {
		t.Fatalf("path = %q, want /tv", got.Path)
	}
}

func TestCapsDefaultCategories(t *testing.T) {
	d := &Definition{Raw: map[string]any{"caps": map[string]any{"categorymappings": []any{
		map[string]any{"id": "2000", "cat": "2000", "default": true},
		map[string]any{"id": "5000", "cat": "5000"},
	}}}}
	got := DefaultCategories(d)
	if len(got) != 1 || got[0] != "2000" {
		t.Fatalf("DefaultCategories() = %#v", got)
	}
}

func TestSearchRequestWithOptionsRendersStructuredQueryFields(t *testing.T) {
	d := &Definition{Config: map[string]string{}, Raw: map[string]any{
		"search": map[string]any{
			"inputs": map[string]any{
				"search":       "{{ .Keywords }}",
				"imdb":         "{{ .Query.IMDBID }}",
				"season":       "{{ .Query.Season }}",
				"ep":           "{{ .Query.Ep }}",
				"douban":       "{{ .Query.DoubanID }}",
				"missingFalse": "{{ if eq .Query.TMDBID .False }}none{{ else }}have{{ end }}",
			},
		},
	}}
	got := SearchRequestWithOptions(d, SearchOptions{Keywords: "dune", IMDBID: "tt123", Season: "2", Episode: "3", DoubanID: "db1"})
	if got.Inputs["search"] != "dune" || got.Inputs["imdb"] != "tt123" || got.Inputs["season"] != "2" || got.Inputs["ep"] != "3" || got.Inputs["douban"] != "db1" || got.Inputs["missingFalse"] != "none" {
		t.Fatalf("inputs=%#v", got.Inputs)
	}
}

func TestQueryParamUsesModeSpecificCapsParam(t *testing.T) {
	d := &Definition{Raw: map[string]any{"caps": map[string]any{"modes": map[string]any{
		"search":       map[string]any{"params": []any{map[string]any{"name": "q"}}},
		"movie-search": map[string]any{"params": []any{map[string]any{"name": "imdbid"}}},
	}}}}
	if got := QueryParamForMode(d, "movie-search"); got != "imdbid" {
		t.Fatalf("got %q", got)
	}
}

func TestSearchRequestWithOptionsRendersIMDBIDShort(t *testing.T) {
	d := &Definition{Config: map[string]string{}, Raw: map[string]any{
		"search": map[string]any{"inputs": map[string]any{"imdb": "{{ .Query.IMDBIDShort }}"}},
	}}
	got := SearchRequestWithOptions(d, SearchOptions{IMDBID: "tt1234567"})
	if got.Inputs["imdb"] != "1234567" {
		t.Fatalf("inputs=%#v", got.Inputs)
	}
}
