package cardigann

import "testing"

func TestApplyFiltersValidate(t *testing.T) {
	d := defWithFilters("x", []any{map[string]any{"name": "validate", "args": []any{"^magnet:"}}})
	if got := ApplyFilters(d, "x", "http://bad", nil); got != "" {
		t.Fatalf("got %q", got)
	}
	if got := ApplyFilters(d, "x", "magnet:?xt=1", nil); got != "magnet:?xt=1" {
		t.Fatalf("got %q", got)
	}
}
