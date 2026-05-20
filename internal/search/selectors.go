package search

import (
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/antchfx/htmlquery"
	"golang.org/x/net/html"

	"github.com/sergiobonfiglio/tomagnet/internal/cardigann"
)

func isXPathSelector(sel string) bool {
	sel = strings.TrimSpace(sel)
	return strings.HasPrefix(sel, "/") || strings.HasPrefix(sel, "./") || strings.HasPrefix(sel, ".//")
}

func selectNodes(scope *goquery.Selection, sel string) *goquery.Selection {
	sel = strings.TrimSpace(sel)
	if scope == nil || len(scope.Nodes) == 0 || sel == "" {
		return emptySelection(scope)
	}
	if !isXPathSelector(sel) {
		return scope.Find(cardigann.CSSSelector(sel))
	}
	var out []*html.Node
	for _, n := range scope.Nodes {
		nodes, err := htmlquery.QueryAll(n, sel)
		if err == nil {
			out = append(out, nodes...)
		}
	}
	if len(out) == 0 {
		return emptySelection(scope)
	}
	return goquery.NewDocumentFromNode(scope.Nodes[0]).Find("__pi_empty__").AddNodes(out...)
}

func matchesSelector(node *goquery.Selection, sel string) bool {
	sel = strings.TrimSpace(sel)
	if node == nil || len(node.Nodes) == 0 || sel == "" {
		return false
	}
	if !isXPathSelector(sel) {
		return node.Is(cardigann.CSSSelector(sel))
	}
	parent := node.Parent()
	if parent.Length() == 0 {
		parent = goquery.NewDocumentFromNode(node.Nodes[0]).Selection
	}
	for _, n := range selectNodes(parent, sel).Nodes {
		if n == node.Nodes[0] {
			return true
		}
	}
	return false
}

func emptySelection(scope *goquery.Selection) *goquery.Selection {
	if scope == nil || len(scope.Nodes) == 0 {
		return goquery.NewDocumentFromNode(&html.Node{Type: html.DocumentNode}).Find("__pi_empty__")
	}
	return goquery.NewDocumentFromNode(scope.Nodes[0]).Find("__pi_empty__")
}
