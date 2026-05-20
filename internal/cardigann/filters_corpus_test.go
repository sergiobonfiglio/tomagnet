package cardigann

import (
	"regexp"
	"testing"
	"time"
)

func TestApplyFiltersDateparse(t *testing.T) {
	d := defWithFilters("x", []any{map[string]any{"name": "dateparse", "args": []any{"yyyy-MM-dd HH:mm:ss"}}})
	got := ApplyFilters(d, "x", "2024-01-02 03:04:05", nil)
	if got != "2024-01-02T03:04:05Z" {
		t.Fatalf("got %q", got)
	}
}

func TestApplyFiltersTimeagoAndFuzzytime(t *testing.T) {
	d := defWithFilters("x", []any{map[string]any{"name": "timeago"}})
	if got := ApplyFilters(d, "x", "2 hours ago", nil); !regexp.MustCompile(`^\d{4}-`).MatchString(got) {
		t.Fatalf("timeago got %q", got)
	}

	d = defWithFilters("x", []any{map[string]any{"name": "fuzzytime"}})
	if got := ApplyFilters(d, "x", "Yesterday 03:04", nil); !regexp.MustCompile(`^\d{4}-`).MatchString(got) {
		t.Fatalf("fuzzytime got %q", got)
	}
}

func TestApplyFiltersDiacriticsValidfilenameAndAndmatch(t *testing.T) {
	d := defWithFilters("x", []any{map[string]any{"name": "diacritics"}})
	if got := ApplyFilters(d, "x", "Amélie", nil); got != "Amelie" {
		t.Fatalf("diacritics got %q", got)
	}

	d = defWithFilters("x", []any{map[string]any{"name": "validfilename"}})
	if got := ApplyFilters(d, "x", `bad:/\\name?*`, nil); got == `bad:/\\name?*` {
		t.Fatalf("validfilename got %q", got)
	}

	d = defWithFilters("x", []any{map[string]any{"name": "andmatch", "args": []any{"1080p", "x265"}}})
	if got := ApplyFilters(d, "x", "Movie 1080p x265", nil); got == "" {
		t.Fatalf("andmatch got %q", got)
	}
	if got := ApplyFilters(d, "x", "Movie 1080p", nil); got != "" {
		t.Fatalf("andmatch mismatch got %q", got)
	}
}

func TestTimeagoStableEnough(t *testing.T) {
	d := defWithFilters("x", []any{map[string]any{"name": "timeago"}})
	got := ApplyFilters(d, "x", "1 day ago", nil)
	ts, err := time.Parse(time.RFC3339, got)
	if err != nil {
		t.Fatalf("parse %q: %v", got, err)
	}
	if time.Since(ts) < 23*time.Hour || time.Since(ts) > 25*time.Hour {
		t.Fatalf("unexpected delta: %v", time.Since(ts))
	}
}
