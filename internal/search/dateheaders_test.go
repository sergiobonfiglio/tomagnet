package search

import (
	"testing"

	"github.com/sergiobonfiglio/tomagnet/internal/cardigann"
)

func TestParseHTMLDateHeaders(t *testing.T) {
	d := &cardigann.Definition{Config: map[string]string{}, Raw: map[string]any{"search": map[string]any{
		"rows":   map[string]any{"selector": "tr.item", "dateheaders": map[string]any{"selector": "tr.date"}},
		"fields": map[string]any{"title": map[string]any{"selector": ".title"}, "date": map[string]any{"selector": ".date"}},
	}}}
	got, err := parseHTML("idx", d, `<table><tr class="date"><td>2024-01-02</td></tr><tr class="item"><td class="title">Dune</td></tr></table>`, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].PublishDate == nil {
		t.Fatalf("got %#v", got)
	}
}
