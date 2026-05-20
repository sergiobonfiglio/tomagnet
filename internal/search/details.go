package search

import (
	"context"
	"net/url"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/sergiobonfiglio/tomagnet/internal/cardigann"
	"github.com/sergiobonfiglio/tomagnet/internal/fetch"
	"github.com/sergiobonfiglio/tomagnet/internal/normalize"
)

type DetailFetcher func(context.Context, fetch.Request) ([]byte, string, error)

func EnrichDetails(ctx context.Context, d *cardigann.Definition, rs []Result, fetcher DetailFetcher) []Result {
	for i := range rs {
		if !shouldEnrichDetails(d, rs[i]) {
			continue
		}
		fr := cardigann.FollowRedirect(d)
		body, ct, err := fetcher(ctx, fetch.Request{Method: "get", Base: d.BaseURL, Path: *rs[i].DetailsURL, FollowRedirect: &fr})
		if err != nil || !strings.Contains(ct, "html") {
			continue
		}
		doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(body)))
		if err != nil {
			continue
		}
		if rs[i].MagnetURL == nil {
			if v := detailValue(doc, d, "magnet"); v != "" {
				rs[i].MagnetURL = &v
			}
		}
		if rs[i].DownloadURL == nil {
			if v := detailValue(doc, d, "download"); v != "" {
				v = normalize.Abs(d.BaseURL, v)
				rs[i].DownloadURL = &v
			}
		}
		applyDownloadSelectors(&rs[i], d, string(body), false)
		if rs[i].InfoHash == nil && !cardigann.DownloadInfoHashUsesBeforeResponse(d) {
			if ih := downloadInfohashValue(string(body), d, "hash", false); ih != "" {
				rs[i].InfoHash = &ih
			}
		}
		if needsBeforeResponse(d) {
			rawURL := *rs[i].DetailsURL
			if rs[i].DownloadURL != nil {
				rawURL = *rs[i].DownloadURL
			}
			preq := buildDownloadBeforeRequest(d, rawURL, string(body))
			if preq.Path != "" {
				b2, _, err := fetcher(ctx, fetch.Request{Method: preq.Method, Base: d.BaseURL, Path: preq.Path, Inputs: preq.Inputs, Headers: preq.Headers, FollowRedirect: &preq.FollowRedirect})
				if err == nil {
					applyDownloadSelectors(&rs[i], d, string(b2), true)
					if rs[i].InfoHash == nil && cardigann.DownloadInfoHashUsesBeforeResponse(d) {
						if ih := downloadInfohashValue(string(b2), d, "hash", true); ih != "" {
							rs[i].InfoHash = &ih
						}
					}
					if rs[i].MagnetURL == nil && rs[i].InfoHash != nil {
						if title := downloadInfohashValue(string(b2), d, "title", true); title != "" {
							mag := "magnet:?xt=urn:btih:" + *rs[i].InfoHash + "&dn=" + url.QueryEscape(title)
							rs[i].MagnetURL = &mag
						}
					}
				}
			}
		}
		if rs[i].MagnetURL == nil && rs[i].InfoHash != nil {
			mag := "magnet:?xt=urn:btih:" + *rs[i].InfoHash
			if title := downloadInfohashValue(string(body), d, "title", false); title != "" {
				mag += "&dn=" + url.QueryEscape(title)
			}
			rs[i].MagnetURL = &mag
		}
	}
	return rs
}

func shouldEnrichDetails(d *cardigann.Definition, r Result) bool {
	if r.DetailsURL == nil {
		return false
	}
	if needsBeforeResponse(d) {
		return true
	}
	if r.MagnetURL == nil && cardigann.DetailFieldSelector(d, "magnet") != "" {
		return true
	}
	if r.DownloadURL == nil && cardigann.DetailFieldSelector(d, "download") != "" {
		return true
	}
	if sels := cardigann.DownloadSelectors(d); len(sels) > 0 && (r.MagnetURL == nil || r.DownloadURL == nil) {
		return true
	}
	if r.InfoHash == nil && cardigann.DownloadInfoHashSelector(d, "hash").Selector != "" {
		return true
	}
	return false
}

func needsBeforeResponse(d *cardigann.Definition) bool {
	if cardigann.DownloadInfoHashUsesBeforeResponse(d) {
		return true
	}
	for _, sel := range cardigann.DownloadSelectors(d) {
		if sel.UseBeforeResponse {
			return true
		}
	}
	return false
}

func applyDownloadSelectors(r *Result, d *cardigann.Definition, body string, before bool) {
	for _, sel := range cardigann.DownloadSelectors(d) {
		if sel.UseBeforeResponse != before {
			continue
		}
		v := selectorValue(d, body, sel)
		if v == "" {
			continue
		}
		if strings.HasPrefix(v, "magnet:") && r.MagnetURL == nil {
			r.MagnetURL = &v
			continue
		}
		if r.DownloadURL == nil {
			vv := normalize.Abs(d.BaseURL, v)
			r.DownloadURL = &vv
		}
	}
}

func detailValue(doc *goquery.Document, d *cardigann.Definition, field string) string {
	sel := cardigann.DetailFieldSelector(d, field)
	if sel == "" {
		return ""
	}
	q := selectNodes(doc.Selection, sel).First()
	if attr := cardigann.DetailFieldAttr(d, field); attr != "" {
		v, _ := q.Attr(attr)
		return v
	}
	return q.Text()
}

func downloadInfohashValue(body string, d *cardigann.Definition, field string, before bool) string {
	s := cardigann.DownloadInfoHashSelector(d, field)
	if s.Selector == "" || cardigann.DownloadInfoHashUsesBeforeResponse(d) != before {
		return ""
	}
	return selectorValue(d, body, s)
}

func selectorValue(d *cardigann.Definition, body string, s cardigann.SelectorSpec) string {
	if s.Selector == ":root" {
		return cardigann.ApplyFilterList(d, s.Filters, body, nil)
	}
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(body))
	if err != nil {
		return ""
	}
	q := selectNodes(doc.Selection, s.Selector).First()
	if q.Length() == 0 {
		return ""
	}
	v := q.Text()
	if s.Attribute != "" {
		v, _ = q.Attr(s.Attribute)
	}
	return cardigann.ApplyFilterList(d, s.Filters, v, nil)
}
