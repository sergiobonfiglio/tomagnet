package tomagnet

import (
	"context"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"
)

func TestSearchOptionsExposeIntentBasedQueryFields(t *testing.T) {
	opt := SearchOptions{
		Query:      Query{Movie: &MovieQuery{Title: "Dune", Year: 2024, IMDBID: "tt1", TMDBID: "tm1"}},
		Categories: []string{"5000"},
	}

	if opt.Query.Movie == nil || opt.Query.Movie.Title != "Dune" || opt.Query.Movie.Year != 2024 || opt.Query.Movie.IMDBID != "tt1" || opt.Query.Movie.TMDBID != "tm1" || len(opt.Categories) != 1 {
		t.Fatalf("opt=%#v", opt)
	}
}

func TestDefaultIndexersExposeCurrentDefaults(t *testing.T) {
	indexers := DefaultIndexers()
	ids := make([]string, 0, len(indexers))
	for _, idx := range indexers {
		ids = append(ids, idx.ID)
		if idx.BaseURL == "" {
			t.Fatalf("default indexer %q missing base url", idx.ID)
		}
		if idx.TimeoutSeconds <= 0 {
			t.Fatalf("default indexer %q missing timeout", idx.ID)
		}
	}
	if !slices.Equal(ids, []string{"btdig", "yts", "limetorrents", "thepiratebay"}) {
		t.Fatalf("ids=%v", ids)
	}
}

func TestSearchUsesInMemoryDefinition(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"results":[{"title":"Dune","magnet":"magnet:?xt=urn:btih:abc"}]}`))
	}))
	t.Cleanup(srv.Close)

	definition := &Definition{
		ID:      "custom",
		Name:    "Custom",
		BaseURL: srv.URL,
		Search: SearchDefinition{
			Path: "/",
			Rows: RowsDefinition{Selector: "results"},
			Fields: map[string]FieldDefinition{
				"title":  {Selector: "title"},
				"magnet": {Selector: "magnet"},
			},
		},
	}

	resp := Search(context.Background(), SearchOptions{
		Query:    Query{Text: "dune"},
		Indexers: []Indexer{{ID: "custom", TimeoutSeconds: 5, Definition: definition}},
	})
	if len(resp.Errors) > 0 {
		t.Fatalf("errors: %#v", resp.Errors)
	}
	if len(resp.Results) != 1 || resp.Results[0].Title != "Dune" {
		t.Fatalf("results: %#v", resp.Results)
	}
}

func TestSearchMovieQueryPrefersMovieModeWhenSupported(t *testing.T) {
	var gotType, gotQuery, gotTitle, gotYear, gotTMDBID string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotType = r.URL.Query().Get("t")
		gotQuery = r.URL.Query().Get("q")
		gotTitle = r.URL.Query().Get("title")
		gotYear = r.URL.Query().Get("year")
		gotTMDBID = r.URL.Query().Get("tmdbid")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"results":[{"title":"Dune","magnet":"magnet:?xt=urn:btih:abc"}]}`))
	}))
	t.Cleanup(srv.Close)

	definition := &Definition{
		ID:      "custom",
		Name:    "Custom",
		BaseURL: srv.URL,
		Caps: Caps{Modes: map[string]SearchMode{
			"search":       {Params: []SearchParam{{Name: "q"}}},
			"movie-search": {Params: []SearchParam{{Name: "q"}, {Name: "imdbid"}}},
		}},
		Search: SearchDefinition{
			Path:   "/",
			Inputs: StringMap{"t": "{{ .Query.Type }}", "q": "{{ .Keywords }}", "title": "{{ .Query.Title }}", "year": "{{ .Query.Year }}", "tmdbid": "{{ .Query.TMDBID }}"},
			Rows:   RowsDefinition{Selector: "results"},
			Fields: map[string]FieldDefinition{
				"title":  {Selector: "title"},
				"magnet": {Selector: "magnet"},
			},
		},
	}

	resp := Search(context.Background(), SearchOptions{
		Query:    Query{Movie: &MovieQuery{Title: "Dune", Year: 2024}},
		Indexers: []Indexer{{ID: "custom", BaseURL: srv.URL, TimeoutSeconds: 5, Definition: definition}},
	})
	if len(resp.Errors) > 0 {
		t.Fatalf("errors: %#v", resp.Errors)
	}
	if gotType != "movie-search" {
		t.Fatalf("type=%q", gotType)
	}
	if gotQuery != "Dune 2024" || gotTitle != "Dune" || gotYear != "2024" {
		t.Fatalf("query=%q title=%q year=%q", gotQuery, gotTitle, gotYear)
	}
	if gotTMDBID != "" {
		t.Fatalf("tmdbid=%q, want empty", gotTMDBID)
	}
}

func TestSearchMovieQueryFallsBackToGenericSearchWhenSpecificModeNeedsMissingParams(t *testing.T) {
	var gotType, gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotType = r.URL.Query().Get("t")
		gotQuery = r.URL.Query().Get("q")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"results":[{"title":"Dune","magnet":"magnet:?xt=urn:btih:abc"}]}`))
	}))
	t.Cleanup(srv.Close)

	definition := &Definition{
		ID:      "custom",
		Name:    "Custom",
		BaseURL: srv.URL,
		Caps: Caps{Modes: map[string]SearchMode{
			"search":       {Params: []SearchParam{{Name: "q"}}},
			"movie-search": {Params: []SearchParam{{Name: "imdbid"}}},
		}},
		Search: SearchDefinition{
			Path:   "/",
			Inputs: StringMap{"t": "{{ .Query.Type }}", "q": "{{ .Keywords }}"},
			Rows:   RowsDefinition{Selector: "results"},
			Fields: map[string]FieldDefinition{
				"title":  {Selector: "title"},
				"magnet": {Selector: "magnet"},
			},
		},
	}

	resp := Search(context.Background(), SearchOptions{
		Query:    Query{Movie: &MovieQuery{Title: "Dune", Year: 2024}},
		Indexers: []Indexer{{ID: "custom", BaseURL: srv.URL, TimeoutSeconds: 5, Definition: definition}},
	})
	if len(resp.Errors) > 0 {
		t.Fatalf("errors: %#v", resp.Errors)
	}
	if gotType != "search" {
		t.Fatalf("type=%q", gotType)
	}
	if gotQuery != "Dune 2024" {
		t.Fatalf("query=%q", gotQuery)
	}
}
