package search

import (
	"testing"

	"github.com/sergiobonfiglio/tomagnet/internal/cardigann"
)

func TestParseHTMLXPathSelectors(t *testing.T) {
	d := &cardigann.Definition{BaseURL: "https://idx.test", Config: map[string]string{}, Raw: map[string]any{"search": map[string]any{
		"rows": map[string]any{"selector": "//table/tbody/tr"},
		"fields": map[string]any{
			"title":    map[string]any{"selector": "./td[@class='name']/a"},
			"download": map[string]any{"selector": "./td/a[@class='dl']", "attribute": "href"},
		},
	}}}
	got, err := parseHTML("idx", d, `<table><tbody><tr><td class="name"><a>Dune</a></td><td><a class="dl" href="/d.torrent">dl</a></td></tr></tbody></table>`, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || *got[0].Title != "Dune" || *got[0].DownloadURL != "https://idx.test/d.torrent" {
		t.Fatalf("unexpected: %#v", got)
	}
}

func TestParseHTMLXPathLibraryFeatures(t *testing.T) {
	d := &cardigann.Definition{BaseURL: "https://idx.test", Config: map[string]string{}, Raw: map[string]any{"search": map[string]any{
		"rows": map[string]any{"selector": "//table//tr[@data-id='1'][1]"},
		"fields": map[string]any{
			"title":    map[string]any{"selector": ".//td[contains(@class,'na')]/a"},
			"download": map[string]any{"selector": ".//a[contains(@class,'dl')]", "attribute": "href"},
		},
	}}}
	html := `<div><table><tbody><tr data-id="1"><td class="name"><a>Dune</a></td><td><span><a class="dl" href="/d.torrent">dl</a></span></td></tr></tbody></table></div>`
	got, err := parseHTML("idx", d, html, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || *got[0].Title != "Dune" || *got[0].DownloadURL != "https://idx.test/d.torrent" {
		t.Fatalf("unexpected: %#v", got)
	}
}
