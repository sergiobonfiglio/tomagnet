package search

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"sync"

	"github.com/PuerkitoBio/goquery"
	"github.com/sergiobonfiglio/tomagnet/internal/cardigann"
	"github.com/sergiobonfiglio/tomagnet/internal/fetch"
	"github.com/sergiobonfiglio/tomagnet/internal/normalize"
)

const defaultDetailConcurrency = 6

type DetailFetcher func(context.Context, fetch.Request) ([]byte, string, error)

func EnrichDetails(ctx context.Context, d *cardigann.Definition, results []Result, fetcher DetailFetcher) []Result {
	return enrichDetails(ctx, d, results, fetcher, defaultDetailConcurrency)
}

func enrichDetails(ctx context.Context, d *cardigann.Definition, results []Result, fetcher DetailFetcher, concurrency int) []Result {
	if concurrency <= 0 {
		concurrency = defaultDetailConcurrency
	}
	if concurrency > len(results) {
		concurrency = len(results)
	}
	out := append([]Result(nil), results...)
	jobs := make(chan int)
	var workers sync.WaitGroup
	for range concurrency {
		workers.Go(func() {
			for i := range jobs {
				out[i] = enrichDetail(ctx, d, out[i], fetcher)
			}
		})
	}
	for i := range out {
		jobs <- i
	}
	close(jobs)
	workers.Wait()
	return out
}

func enrichDetail(ctx context.Context, d *cardigann.Definition, result Result, fetcher DetailFetcher) Result {
	if !shouldEnrichDetails(d, result) {
		return result
	}
	fr := cardigann.FollowRedirect(d)
	body, contentType, err := fetcher(ctx, fetch.Request{Method: "get", Base: d.BaseURL, Path: *result.DetailsURL, FollowRedirect: &fr})
	if err != nil {
		return withEnrichmentError(result, "fetch details: %v", err)
	}
	if !strings.Contains(contentType, "html") {
		return withEnrichmentError(result, "fetch details: expected HTML response, got %q", contentType)
	}
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(body)))
	if err != nil {
		return withEnrichmentError(result, "parse details: %v", err)
	}

	rawURL := *result.DetailsURL
	if result.DownloadURL != nil {
		rawURL = *result.DownloadURL
	}
	resolved := false
	if result.MagnetURL == nil {
		if value := detailValue(doc, d, "magnet"); value != "" {
			result.MagnetURL = &value
			resolved = true
		}
	}
	if result.DownloadURL == nil {
		if value := detailValue(doc, d, "download"); value != "" {
			value = normalize.Abs(d.BaseURL, value)
			result.DownloadURL = &value
			resolved = true
		}
	}
	if applyDownloadSelectors(&result, d, string(body), false) {
		resolved = true
	}
	if result.InfoHash == nil && !cardigann.DownloadInfoHashUsesBeforeResponse(d) {
		if infoHash := downloadInfohashValue(string(body), d, "hash", false); infoHash != "" {
			result.InfoHash = &infoHash
			resolved = true
		}
	}
	if setMagnetFromInfoHash(&result, d, string(body), false) {
		resolved = true
	}

	// Download selectors are fallbacks. If the original page already exposed a
	// direct result, the before request and its side effects are unnecessary.
	if cardigann.HasDownloadBefore(d) && !resolved {
		beforeRequest := buildDownloadBeforeRequest(d, rawURL, string(body))
		if beforeRequest.Path == "" {
			return withEnrichmentError(result, "download before path not found")
		}
		beforeBody, _, err := fetcher(ctx, fetch.Request{Method: beforeRequest.Method, Base: d.BaseURL, Path: beforeRequest.Path, Inputs: beforeRequest.Inputs, Headers: beforeRequest.Headers, FollowRedirect: &beforeRequest.FollowRedirect})
		if err != nil {
			return withEnrichmentError(result, "download before request: %v", err)
		}
		if applyDownloadSelectors(&result, d, string(beforeBody), true) {
			resolved = true
		}
		if result.InfoHash == nil && cardigann.DownloadInfoHashUsesBeforeResponse(d) {
			if infoHash := downloadInfohashValue(string(beforeBody), d, "hash", true); infoHash != "" {
				result.InfoHash = &infoHash
				resolved = true
			}
		}
		if setMagnetFromInfoHash(&result, d, string(beforeBody), true) {
			resolved = true
		}
		if !resolved && needsDownloadPageAfterBefore(d) {
			refreshedBody, contentType, err := fetcher(ctx, fetch.Request{Method: "get", Base: d.BaseURL, Path: rawURL, FollowRedirect: &fr})
			if err != nil {
				return withEnrichmentError(result, "refetch details: %v", err)
			}
			if !strings.Contains(contentType, "html") {
				return withEnrichmentError(result, "refetch details: expected HTML response, got %q", contentType)
			}
			body = refreshedBody
			if applyDownloadSelectors(&result, d, string(body), false) {
				resolved = true
			}
			if result.InfoHash == nil && !cardigann.DownloadInfoHashUsesBeforeResponse(d) {
				if infoHash := downloadInfohashValue(string(body), d, "hash", false); infoHash != "" {
					result.InfoHash = &infoHash
					resolved = true
				}
			}
			if setMagnetFromInfoHash(&result, d, string(body), false) {
				resolved = true
			}
		}
	}
	if !resolved && expectsDetailResolution(d) {
		return withEnrichmentError(result, "download selectors did not resolve a direct URL")
	}
	return result
}

func withEnrichmentError(result Result, format string, args ...any) Result {
	result.EnrichmentError = &Error{Indexer: result.Indexer, Stage: "enrichment", Message: fmt.Sprintf(format, args...)}
	return result
}

func setMagnetFromInfoHash(result *Result, d *cardigann.Definition, body string, before bool) bool {
	if result.MagnetURL != nil || result.InfoHash == nil {
		return false
	}
	magnet := "magnet:?xt=urn:btih:" + *result.InfoHash
	if title := downloadInfohashValue(body, d, "title", before); title != "" {
		magnet += "&dn=" + url.QueryEscape(title)
	}
	result.MagnetURL = &magnet
	return true
}

func shouldEnrichDetails(d *cardigann.Definition, result Result) bool {
	if result.DetailsURL == nil || result.MagnetURL != nil {
		return false
	}
	if cardigann.HasDownloadBefore(d) {
		return true
	}
	if cardigann.DetailFieldSelector(d, "magnet") != "" {
		return true
	}
	if result.DownloadURL == nil && cardigann.DetailFieldSelector(d, "download") != "" {
		return true
	}
	if selectors := cardigann.DownloadSelectors(d); len(selectors) > 0 && (result.MagnetURL == nil || result.DownloadURL == nil) {
		return true
	}
	return result.InfoHash == nil && cardigann.DownloadInfoHashSelector(d, "hash").Selector != ""
}

func expectsDetailResolution(d *cardigann.Definition) bool {
	return cardigann.HasDownloadBefore(d) ||
		cardigann.DetailFieldSelector(d, "magnet") != "" ||
		cardigann.DetailFieldSelector(d, "download") != "" ||
		len(cardigann.DownloadSelectors(d)) > 0 ||
		cardigann.DownloadInfoHashSelector(d, "hash").Selector != ""
}

func needsDownloadPageAfterBefore(d *cardigann.Definition) bool {
	if cardigann.DownloadInfoHashSelector(d, "hash").Selector != "" && !cardigann.DownloadInfoHashUsesBeforeResponse(d) {
		return true
	}
	for _, selector := range cardigann.DownloadSelectors(d) {
		if !selector.UseBeforeResponse {
			return true
		}
	}
	return false
}

func applyDownloadSelectors(result *Result, d *cardigann.Definition, body string, before bool) bool {
	matched := false
	downloadMatched := false
	for _, selector := range cardigann.DownloadSelectors(d) {
		if selector.UseBeforeResponse != before {
			continue
		}
		value := selectorValue(d, body, selector)
		if value == "" {
			continue
		}
		matched = true
		if strings.HasPrefix(value, "magnet:") {
			if result.MagnetURL == nil {
				result.MagnetURL = &value
			}
			continue
		}
		if !downloadMatched {
			value = normalize.Abs(d.BaseURL, value)
			result.DownloadURL = &value
			downloadMatched = true
		}
	}
	return matched
}

func detailValue(doc *goquery.Document, d *cardigann.Definition, field string) string {
	selector := cardigann.DetailFieldSelector(d, field)
	if selector == "" {
		return ""
	}
	selection := selectNodes(doc.Selection, selector).First()
	if attribute := cardigann.DetailFieldAttr(d, field); attribute != "" {
		value, _ := selection.Attr(attribute)
		return value
	}
	return selection.Text()
}

func downloadInfohashValue(body string, d *cardigann.Definition, field string, before bool) string {
	selector := cardigann.DownloadInfoHashSelector(d, field)
	if selector.Selector == "" || cardigann.DownloadInfoHashUsesBeforeResponse(d) != before {
		return ""
	}
	return selectorValue(d, body, selector)
}

func selectorValue(d *cardigann.Definition, body string, selector cardigann.SelectorSpec) string {
	if selector.Selector == ":root" {
		return cardigann.ApplyFilterList(d, selector.Filters, body, nil)
	}
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(body))
	if err != nil {
		return ""
	}
	selection := selectNodes(doc.Selection, selector.Selector).First()
	if selection.Length() == 0 {
		return ""
	}
	value := selection.Text()
	if selector.Attribute != "" {
		value, _ = selection.Attr(selector.Attribute)
	}
	return cardigann.ApplyFilterList(d, selector.Filters, value, nil)
}
