package search

import (
	"testing"

	"github.com/sergiobonfiglio/tomagnet/internal/cardigann"
)

func TestParseRSSFeedItems(t *testing.T) {
	d := &cardigann.Definition{BaseURL: "https://idx.test", Config: map[string]string{}, Raw: map[string]any{"search": map[string]any{
		"paths": []any{map[string]any{"response": map[string]any{"type": "xml"}}},
		"rows":  map[string]any{"selector": "item"},
		"fields": map[string]any{
			"title":    map[string]any{"selector": "title"},
			"download": map[string]any{"selector": "enclosure", "attribute": "url"},
			"size":     map[string]any{"selector": "enclosure", "attribute": "length"},
			"date":     map[string]any{"selector": "pubDate"},
		},
	}}}
	body := []byte(`<rss><channel><item><title>Dune</title><pubDate>Mon, 02 Jan 2006 15:04:05 GMT</pubDate><enclosure url="/dune.torrent" length="123" /></item></channel></rss>`)
	got, err := Parse("idx", d, body, "application/rss+xml", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || *got[0].Title != "Dune" || *got[0].DownloadURL != "https://idx.test/dune.torrent" || *got[0].Size != 123 || got[0].PublishDate == nil {
		t.Fatalf("unexpected: %#v", got)
	}
}
