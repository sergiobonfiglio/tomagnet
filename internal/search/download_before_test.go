package search

import (
	"testing"

	"github.com/sergiobonfiglio/tomagnet/internal/cardigann"
)

func TestBuildDownloadBeforeRequestUsesPathSelector(t *testing.T) {
	d := &cardigann.Definition{BaseURL: "https://idx.test", Config: map[string]string{}, Raw: map[string]any{"download": map[string]any{"before": map[string]any{
		"pathselector": map[string]any{
			"selector":  "a.thanks",
			"attribute": "href",
			"filters":   []any{map[string]any{"name": "re_replace", "args": []any{"^\\.", "forum"}}},
		},
	}}}}
	html := `<a class="thanks" href="./viewtopic.php?thanks=1"></a>`
	r := buildDownloadBeforeRequest(d, "https://idx.test/download.php?id=1", html)
	if r.Path != "forum/viewtopic.php?thanks=1" || r.Method != "get" {
		t.Fatalf("got %#v", r)
	}
}

func TestBuildDownloadBeforeRequestUsesPathSelectorFilters(t *testing.T) {
	d := &cardigann.Definition{BaseURL: "https://idx.test", Config: map[string]string{}, Raw: map[string]any{"download": map[string]any{"before": map[string]any{
		"pathselector": map[string]any{
			"selector":  "a.dl",
			"attribute": "href",
			"filters":   []any{map[string]any{"name": "replace", "args": []any{"/download?", "/download/?"}}},
		},
	}}}}
	html := `<a class="dl" href="/download?id=1"></a>`
	r := buildDownloadBeforeRequest(d, "https://idx.test/download?id=1", html)
	if r.Path != "/download/?id=1" {
		t.Fatalf("got %#v", r)
	}
}
