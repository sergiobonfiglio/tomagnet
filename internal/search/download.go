package search

import (
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/sergiobonfiglio/tomagnet/internal/cardigann"
)

func buildDownloadBeforeRequest(d *cardigann.Definition, rawURL, page string) cardigann.RequestSpec {
	spec := cardigann.DownloadBeforeRequest(d, rawURL)
	pathSel := cardigann.DownloadBeforePathSelector(d)
	if pathSel.Selector == "" || strings.TrimSpace(page) == "" {
		return spec
	}
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(page))
	if err != nil {
		return spec
	}
	q := selectNodes(doc.Selection, pathSel.Selector).First()
	if q.Length() == 0 {
		return spec
	}
	v := q.Text()
	if pathSel.Attribute != "" {
		v, _ = q.Attr(pathSel.Attribute)
	}
	v = cardigann.ApplyFilterList(d, pathSel.Filters, v, nil)
	if v != "" {
		spec.Path = v
	}
	return spec
}
