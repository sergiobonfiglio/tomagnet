package search

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sergiobonfiglio/tomagnet/internal/cardigann"
	"github.com/sergiobonfiglio/tomagnet/internal/config"
)

func TestRunResolvesRelativeResultURLAndAppliesDownloadBefore(t *testing.T) {
	thanked := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/forum/search.php":
			_, _ = w.Write([]byte(`<div class="result"><a class="title" href="./viewtopic.php?id=1">Result</a></div>`))
		case "/forum/viewtopic.php":
			if thanked {
				_, _ = w.Write([]byte(`<script>addLinkToDocument("ABC123")</script>`))
				return
			}
			_, _ = w.Write([]byte(`<a class="thanks" href="./thanks.php?id=1">Thanks</a>`))
		case "/forum/thanks.php":
			thanked = true
			_, _ = w.Write([]byte(`<html></html>`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	d := &cardigann.Definition{BaseURL: server.URL, Config: map[string]string{}, Raw: map[string]any{
		"search": map[string]any{
			"path": "/forum/search.php",
			"rows": map[string]any{"selector": ".result"},
			"fields": map[string]any{
				"title":    map[string]any{"selector": ".title"},
				"details":  map[string]any{"selector": ".title", "attribute": "href"},
				"download": map[string]any{"selector": ".title", "attribute": "href"},
			},
		},
		"download": map[string]any{
			"before": map[string]any{"pathselector": map[string]any{
				"selector":  ".thanks",
				"attribute": "href",
				"filters": []any{map[string]any{
					"name": "re_replace", "args": []any{"^.", "forum"},
				}},
			}},
			"selectors": []any{map[string]any{
				"selector": "script:contains(addLinkToDocument)",
				"filters": []any{
					map[string]any{"name": "regexp", "args": `addLinkToDocument\("(.*?)"`},
					map[string]any{"name": "prepend", "args": "magnet:?xt=urn:btih:"},
				},
			}},
		},
	}}

	results, err := runOneBase(context.Background(), d, config.Indexer{ID: "nested", TimeoutSeconds: 5}, Options{Query: "test", Mode: "search"})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 || results[0].MagnetURL == nil || *results[0].MagnetURL != "magnet:?xt=urn:btih:ABC123" {
		t.Fatalf("results = %#v", results)
	}
}
