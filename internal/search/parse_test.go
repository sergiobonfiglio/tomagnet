package search

import (
	"testing"

	"github.com/sergiobonfiglio/tomagnet/internal/cardigann"
)

func TestParseJSONRowsAttributeAndFieldDefaults(t *testing.T) {
	d := &cardigann.Definition{BaseURL: "https://idx.test", Config: map[string]string{}, Raw: map[string]any{"search": map[string]any{
		"rows": map[string]any{"selector": "data", "attribute": "torrents"},
		"fields": map[string]any{
			"title":   map[string]any{"selector": "name"},
			"seeders": map[string]any{"selector": "seeders", "optional": true, "default": "0"},
			"details": map[string]any{"selector": "url"},
		},
	}}}
	got := parseJSON("idx", d, `{"data":{"torrents":[{"name":"A","url":"/a"},{"name":"B"}]}}`, 10)
	if len(got) != 2 || *got[0].Title != "A" || *got[0].DetailsURL != "https://idx.test/a" || *got[1].Seeders != 0 {
		t.Fatalf("unexpected results: %#v", got)
	}
}

func TestParseHTMLRemoveAndCaseMapping(t *testing.T) {
	d := &cardigann.Definition{BaseURL: "https://idx.test", Config: map[string]string{}, Raw: map[string]any{"search": map[string]any{
		"rows": map[string]any{"selector": "tr.item", "remove": ".ad"},
		"fields": map[string]any{
			"title":    map[string]any{"selector": ".title"},
			"category": map[string]any{"selector": ".cat", "case": map[string]any{"Movies": "2000", "TV": "5000"}},
		},
	}}}
	got, err := parseHTML("idx", d, `<table><tr class="item"><td class="title">Dune<span class="ad">AD</span></td><td class="cat">Movies</td></tr></table>`, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || *got[0].Title != "Dune" || *got[0].Category != "2000" {
		t.Fatalf("unexpected results: %#v", got)
	}
}

func TestParseJSONDefaultRequiresOptional(t *testing.T) {
	d := &cardigann.Definition{BaseURL: "https://idx.test", Config: map[string]string{}, Raw: map[string]any{"search": map[string]any{
		"rows": map[string]any{"selector": "data"},
		"fields": map[string]any{
			"title":    map[string]any{"selector": "name"},
			"seeders":  map[string]any{"selector": "seeders", "default": "7"},
			"leechers": map[string]any{"selector": "leechers", "optional": true, "default": "3"},
		},
	}}}
	got := parseJSON("idx", d, `{"data":[{"name":"A"}]}`, 10)
	if len(got) != 1 {
		t.Fatalf("unexpected results: %#v", got)
	}
	if got[0].Seeders != nil {
		t.Fatalf("seeders should stay nil for non-optional default: %#v", got[0])
	}
	if got[0].Leechers == nil || *got[0].Leechers != 3 {
		t.Fatalf("optional default should apply: %#v", got[0])
	}
}

func TestParseHTMLDecodesDefinitionEncoding(t *testing.T) {
	d := &cardigann.Definition{BaseURL: "https://idx.test", Config: map[string]string{}, Raw: map[string]any{
		"encoding": "iso-8859-1",
		"search": map[string]any{
			"rows": map[string]any{"selector": "tr.item"},
			"fields": map[string]any{
				"title": map[string]any{"selector": ".title"},
			},
		},
	}}
	body := []byte("<table><tr class='item'><td class='title'>Caf\xe9</td></tr></table>")
	got, err := Parse("idx", d, body, "text/html", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Title == nil || *got[0].Title != "Café" {
		t.Fatalf("unexpected results: %#v", got)
	}
}
