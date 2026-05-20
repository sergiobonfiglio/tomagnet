package search

import (
	"testing"

	"github.com/sergiobonfiglio/tomagnet/internal/cardigann"
)

func TestParseHTMLRowsAfterMergesFollowingRows(t *testing.T) {
	d := &cardigann.Definition{Config: map[string]string{}, Raw: map[string]any{"search": map[string]any{
		"rows":   map[string]any{"selector": "tr.item", "after": 1},
		"fields": map[string]any{"title": map[string]any{"selector": ".title"}},
	}}}
	got, err := parseHTML("idx", d, `<table><tr class="item"><td class="title">Dune</td></tr><tr><td class="title"> Part Two</td></tr></table>`, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Title == nil || *got[0].Title != "Dune Part Two" {
		t.Fatalf("title=%q got %#v", deref(got[0].Title), got)
	}
}

func TestParseHTMLFieldRemoveAppliesBeforeTextExtraction(t *testing.T) {
	d := &cardigann.Definition{Config: map[string]string{}, Raw: map[string]any{"search": map[string]any{
		"rows":   map[string]any{"selector": ".item"},
		"fields": map[string]any{"title": map[string]any{"selector": ".title", "remove": ".badge"}},
	}}}
	got, err := parseHTML("idx", d, `<div class="item"><div class="title">Dune <span class="badge">NEW</span></div></div>`, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || *got[0].Title != "Dune" {
		t.Fatalf("got %#v", got)
	}
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
