package search

import (
	"context"
	"testing"

	"github.com/sergiobonfiglio/tomagnet/internal/cardigann"
	"github.com/sergiobonfiglio/tomagnet/internal/fetch"
)

func TestEnrichDetailsFillsMissingMagnetAndDownload(t *testing.T) {
	d := &cardigann.Definition{BaseURL: "https://idx.test", Config: map[string]string{}, Raw: map[string]any{"details": map[string]any{
		"fields": map[string]any{
			"magnet":   map[string]any{"selector": "a.mag", "attribute": "href"},
			"download": map[string]any{"selector": "a.dl", "attribute": "href"},
		},
	}}}
	details := "https://idx.test/t/1"
	r := Result{Indexer: "idx", DetailsURL: &details}
	fetcher := func(ctx context.Context, req fetch.Request) ([]byte, string, error) {
		if req.Path != details {
			t.Fatalf("req = %#v", req)
		}
		return []byte(`<a class="mag" href="magnet:?xt=urn:btih:ABC"></a><a class="dl" href="/d.torrent"></a>`), "text/html", nil
	}
	got := EnrichDetails(context.Background(), d, []Result{r}, fetcher)
	if got[0].MagnetURL == nil || *got[0].MagnetURL != "magnet:?xt=urn:btih:ABC" || got[0].DownloadURL == nil || *got[0].DownloadURL != "https://idx.test/d.torrent" {
		t.Fatalf("unexpected: %#v", got[0])
	}
}

func TestEnrichDetailsUsesTopLevelDownloadSelectors(t *testing.T) {
	d := &cardigann.Definition{BaseURL: "https://idx.test", Config: map[string]string{}, Raw: map[string]any{"download": map[string]any{
		"selectors": []any{
			map[string]any{"selector": "a[href^='magnet:']", "attribute": "href"},
			map[string]any{"selector": "a[href$='.torrent']", "attribute": "href"},
		},
	}}}
	details := "https://idx.test/t/1"
	r := Result{Indexer: "idx", DetailsURL: &details}
	fetcher := func(ctx context.Context, req fetch.Request) ([]byte, string, error) {
		return []byte(`<a href="magnet:?xt=urn:btih:ABC"></a><a href="/d.torrent"></a>`), "text/html", nil
	}
	got := EnrichDetails(context.Background(), d, []Result{r}, fetcher)
	if got[0].MagnetURL == nil || *got[0].MagnetURL != "magnet:?xt=urn:btih:ABC" || got[0].DownloadURL == nil || *got[0].DownloadURL != "https://idx.test/d.torrent" {
		t.Fatalf("unexpected: %#v", got[0])
	}
}

func TestEnrichDetailsUsesDownloadInfohashSelectors(t *testing.T) {
	d := &cardigann.Definition{BaseURL: "https://idx.test", Config: map[string]string{}, Raw: map[string]any{"download": map[string]any{
		"infohash": map[string]any{
			"hash": map[string]any{
				"selector":  `a[href^="magnet:?xt"]`,
				"attribute": "href",
				"filters":   []any{map[string]any{"name": "regexp", "args": `([A-F|a-f|0-9]{40})`}},
			},
			"title": map[string]any{
				"selector":  `a[href^="magnet:?xt"]`,
				"attribute": "href",
				"filters":   []any{map[string]any{"name": "regexp", "args": `&dn=(.+?)&`}},
			},
		},
	}}}
	details := "https://idx.test/t/1"
	r := Result{Indexer: "idx", DetailsURL: &details}
	fetcher := func(ctx context.Context, req fetch.Request) ([]byte, string, error) {
		return []byte(`<a href="magnet:?xt=urn:btih:0123456789ABCDEF0123456789ABCDEF01234567&dn=Dune.Part.Two&tr=x"></a>`), "text/html", nil
	}
	got := EnrichDetails(context.Background(), d, []Result{r}, fetcher)
	if got[0].InfoHash == nil || *got[0].InfoHash != "0123456789ABCDEF0123456789ABCDEF01234567" {
		t.Fatalf("unexpected infohash: %#v", got[0])
	}
	if got[0].MagnetURL == nil || *got[0].MagnetURL != "magnet:?xt=urn:btih:0123456789ABCDEF0123456789ABCDEF01234567&dn=Dune.Part.Two" {
		t.Fatalf("unexpected magnet: %#v", got[0])
	}
}

func TestEnrichDetailsUsesBeforeResponseDownloadSelector(t *testing.T) {
	d := &cardigann.Definition{BaseURL: "https://idx.test", Config: map[string]string{}, Raw: map[string]any{"download": map[string]any{
		"before": map[string]any{
			"pathselector": map[string]any{"selector": "a.pre", "attribute": "href"},
		},
		"selectors": []any{
			map[string]any{"selector": "script", "filters": []any{map[string]any{"name": "regexp", "args": `torrentLink = '(.+?)';`}}, "usebeforeresponse": true},
		},
	}}}
	details := "https://idx.test/t/1"
	r := Result{Indexer: "idx", DetailsURL: &details}
	fetcher := func(ctx context.Context, req fetch.Request) ([]byte, string, error) {
		switch req.Path {
		case details:
			return []byte(`<a class="pre" href="/pre"></a>`), "text/html", nil
		case "/pre":
			return []byte(`<script>torrentLink = '/final.torrent';</script>`), "text/html", nil
		default:
			t.Fatalf("unexpected req: %#v", req)
			return nil, "", nil
		}
	}
	got := EnrichDetails(context.Background(), d, []Result{r}, fetcher)
	if got[0].DownloadURL == nil || *got[0].DownloadURL != "https://idx.test/final.torrent" {
		t.Fatalf("unexpected: %#v", got[0])
	}
}

func TestEnrichDetailsUsesBeforeResponseInfohash(t *testing.T) {
	d := &cardigann.Definition{BaseURL: "https://idx.test", Config: map[string]string{}, Raw: map[string]any{"download": map[string]any{
		"before": map[string]any{"path": "/api/info", "inputs": map[string]any{"id": "{{ .DownloadUri.Query.id }}"}},
		"infohash": map[string]any{
			"usebeforeresponse": true,
			"hash":              map[string]any{"selector": ":root", "filters": []any{map[string]any{"name": "regexp", "args": `([A-F|a-f|0-9]{40})`}}},
			"title":             map[string]any{"selector": ":root", "filters": []any{map[string]any{"name": "regexp", "args": `name=([^&]+)`}}},
		},
	}}}
	dl := "https://idx.test/download.php?id=42"
	details := "https://idx.test/t/1"
	r := Result{Indexer: "idx", DetailsURL: &details, DownloadURL: &dl}
	fetcher := func(ctx context.Context, req fetch.Request) ([]byte, string, error) {
		switch req.Path {
		case details:
			return []byte(`<html></html>`), "text/html", nil
		case "/api/info":
			if req.Inputs["id"] != "42" {
				t.Fatalf("bad req: %#v", req)
			}
			return []byte(`hash=0123456789ABCDEF0123456789ABCDEF01234567&name=Dune.Part.Two`), "text/plain", nil
		default:
			t.Fatalf("unexpected req: %#v", req)
			return nil, "", nil
		}
	}
	got := EnrichDetails(context.Background(), d, []Result{r}, fetcher)
	if got[0].InfoHash == nil || *got[0].InfoHash != "0123456789ABCDEF0123456789ABCDEF01234567" {
		t.Fatalf("unexpected: %#v", got[0])
	}
	if got[0].MagnetURL == nil || *got[0].MagnetURL != "magnet:?xt=urn:btih:0123456789ABCDEF0123456789ABCDEF01234567&dn=Dune.Part.Two" {
		t.Fatalf("unexpected: %#v", got[0])
	}
}

func TestEnrichDetailsSkipsUselessFetchWhenMagnetAlreadyPresent(t *testing.T) {
	d := &cardigann.Definition{BaseURL: "https://idx.test", Config: map[string]string{}, Raw: map[string]any{}}
	details := "https://idx.test/t/1"
	magnet := "magnet:?xt=urn:btih:ABC"
	r := Result{Indexer: "idx", DetailsURL: &details, MagnetURL: &magnet}
	called := false
	fetcher := func(ctx context.Context, req fetch.Request) ([]byte, string, error) {
		called = true
		return nil, "", nil
	}
	got := EnrichDetails(context.Background(), d, []Result{r}, fetcher)
	if called {
		t.Fatal("unexpected detail fetch")
	}
	if got[0].MagnetURL == nil || *got[0].MagnetURL != magnet {
		t.Fatalf("unexpected: %#v", got[0])
	}
}

func TestEnrichDetailsSkipsUselessFetchWhenDownloadAlreadyPresent(t *testing.T) {
	d := &cardigann.Definition{BaseURL: "https://idx.test", Config: map[string]string{}, Raw: map[string]any{}}
	details := "https://idx.test/t/1"
	download := "https://idx.test/d.torrent"
	r := Result{Indexer: "idx", DetailsURL: &details, DownloadURL: &download}
	called := false
	fetcher := func(ctx context.Context, req fetch.Request) ([]byte, string, error) {
		called = true
		return nil, "", nil
	}
	got := EnrichDetails(context.Background(), d, []Result{r}, fetcher)
	if called {
		t.Fatal("unexpected detail fetch")
	}
	if got[0].DownloadURL == nil || *got[0].DownloadURL != download {
		t.Fatalf("unexpected: %#v", got[0])
	}
}
