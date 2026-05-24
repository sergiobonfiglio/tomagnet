package tomagnet

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSearchOptionsExposeModeAwareFields(t *testing.T) {
	opt := SearchOptions{
		Mode:       "movie-search",
		Season:     "1",
		Episode:    "2",
		IMDBID:     "tt1",
		TMDBID:     "tm1",
		TVDBID:     "tv1",
		DoubanID:   "db1",
		TVMazeID:   "mz1",
		Artist:     "art",
		Album:      "alb",
		Author:     "auth",
		Title:      "ttl",
		Genre:      "gen",
		Year:       "2024",
		Categories: []string{"5000"},
	}

	if opt.Mode != "movie-search" || opt.Season != "1" || opt.Episode != "2" || opt.IMDBID != "tt1" || opt.TMDBID != "tm1" || opt.TVDBID != "tv1" || opt.DoubanID != "db1" || opt.TVMazeID != "mz1" || opt.Artist != "art" || opt.Album != "alb" || opt.Author != "auth" || opt.Title != "ttl" || opt.Genre != "gen" || opt.Year != "2024" || len(opt.Categories) != 1 {
		t.Fatalf("opt=%#v", opt)
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
		Query:    "dune",
		Indexers: []Indexer{{ID: "custom", TimeoutSeconds: 5, Definition: definition}},
	})
	if len(resp.Errors) > 0 {
		t.Fatalf("errors: %#v", resp.Errors)
	}
	if len(resp.Results) != 1 || resp.Results[0].Title != "Dune" {
		t.Fatalf("results: %#v", resp.Results)
	}
}
