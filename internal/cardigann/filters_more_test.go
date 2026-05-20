package cardigann

import "testing"

func TestApplyFiltersStripTagsAndJsonJoinArray(t *testing.T) {
	d := defWithFilters("x", []any{
		map[string]any{"name": "striptags"},
		map[string]any{"name": "trim"},
	})
	if got := ApplyFilters(d, "x", "<b>Dune</b>", nil); got != "Dune" {
		t.Fatalf("striptags got %q", got)
	}

	d = defWithFilters("x", []any{map[string]any{"name": "jsonjoinarray", "args": []any{", "}}})
	if got := ApplyFilters(d, "x", `["a","b"]`, nil); got != "a, b" {
		t.Fatalf("jsonjoinarray got %q", got)
	}
}
