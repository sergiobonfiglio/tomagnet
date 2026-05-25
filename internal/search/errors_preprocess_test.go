package search

import (
	"strings"
	"testing"

	"github.com/sergiobonfiglio/tomagnet/internal/cardigann"
)

func TestParseAppliesPreprocessingFilters(t *testing.T) {
	d := &cardigann.Definition{Config: map[string]string{}, Raw: map[string]any{"search": map[string]any{
		"preprocessingfilters": []any{map[string]any{"name": "replace", "args": []any{"bad", "Dune"}}},
		"rows":                 map[string]any{"selector": ".item"},
		"fields":               map[string]any{"title": map[string]any{"selector": ".title"}},
	}}}
	got, err := Parse("idx", d, []byte(`<div class="item"><span class="title">bad</span></div>`), "text/html", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || *got[0].Title != "Dune" {
		t.Fatalf("unexpected: %#v", got)
	}
}

func TestParseReturnsCardigannErrorSelector(t *testing.T) {
	d := &cardigann.Definition{Config: map[string]string{}, Raw: map[string]any{"search": map[string]any{
		"error": []any{map[string]any{"selector": ".err", "message": map[string]any{"selector": ".err"}}},
		"rows":  map[string]any{"selector": ".item"},
	}}}
	_, err := Parse("idx", d, []byte(`<div class="err">blocked</div>`), "text/html", 10)
	if err == nil || !strings.Contains(err.Error(), "blocked") {
		t.Fatalf("err = %v", err)
	}
}

func TestParseReturnsChallengeErrorInsteadOfEmptyResults(t *testing.T) {
	d := &cardigann.Definition{Config: map[string]string{}, Raw: map[string]any{
		"settings": []any{map[string]any{"type": "info_flaresolverr"}},
		"search":   map[string]any{"rows": map[string]any{"selector": ".item"}},
	}}
	cases := []string{
		`<html><script src="/js/fingerprint/iife.min.js"></script><script>FingerprintJS.load({monitoring:false})</script></html>`,
		`<html><head><title>Loading...</title></head><body><script type='text/javascript'>window.location.replace('https://idx.test/search?ch=1&js=token&sid=abc');</script></body></html>`,
	}
	for _, body := range cases {
		_, err := Parse("idx", d, []byte(body), "text/html", 10)
		if err == nil || !strings.Contains(err.Error(), "browser challenge") {
			t.Fatalf("err = %v", err)
		}
	}
}
