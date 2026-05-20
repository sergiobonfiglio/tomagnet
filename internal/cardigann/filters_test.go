package cardigann

import "testing"

func TestApplyFiltersCommonCardigannFilters(t *testing.T) {
	d := defWithFilters("title", []any{
		map[string]any{"name": "trim"},
		map[string]any{"name": "replace", "args": []any{" ", "."}},
		map[string]any{"name": "tolower"},
	})
	got := ApplyFilters(d, "title", " Dune Part Two ", nil)
	if got != "dune.part.two" {
		t.Fatalf("got %q", got)
	}
}

func TestApplyFiltersRegexpSplitMathAndEncoding(t *testing.T) {
	d := defWithFilters("x", []any{
		map[string]any{"name": "regexp", "args": []any{"size=(\\d+)"}},
		map[string]any{"name": "num_mul", "args": []any{"2"}},
	})
	if got := ApplyFilters(d, "x", "size=21", nil); got != "42" {
		t.Fatalf("math got %q", got)
	}

	d = defWithFilters("x", []any{
		map[string]any{"name": "split", "args": []any{"/", "1"}},
		map[string]any{"name": "urlencode"},
		map[string]any{"name": "urldecode"},
	})
	if got := ApplyFilters(d, "x", "a/b c", nil); got != "b c" {
		t.Fatalf("split/encoding got %q", got)
	}
}

func defWithFilters(field string, filters []any) *Definition {
	return &Definition{Config: map[string]string{}, Raw: map[string]any{
		"search": map[string]any{
			"fields": map[string]any{
				field: map[string]any{"filters": filters},
			},
		},
	}}
}
