package search

import (
	"os"
	"testing"

	"github.com/sergiobonfiglio/tomagnet/internal/cardigann"
	"github.com/sergiobonfiglio/tomagnet/internal/config"
)

func TestBaseURLCacheInvalidatedOnLinkChanges(t *testing.T) {
	for _, tt := range []struct {
		name          string
		links, legacy []any
	}{
		{"removed", []any{"https://primary.test"}, nil},
		{"moved to legacy", []any{"https://primary.test"}, []any{"https://mirror.test"}},
		{"added", []any{"https://primary.test", "https://mirror.test", "https://new.test"}, nil},
		{"reordered", []any{"https://mirror.test", "https://primary.test"}, nil},
		{"legacy changed", []any{"https://primary.test", "https://mirror.test"}, []any{"https://old.test"}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("HOME", t.TempDir())
			t.Setenv("XDG_CACHE_HOME", t.TempDir())
			d := &cardigann.Definition{BaseURL: "https://primary.test", Raw: map[string]any{"links": []any{"https://primary.test", "https://mirror.test"}}}
			cacheBaseURL("idx", "https://mirror.test", d, func(string, ...any) {})
			d.Raw["links"] = tt.links
			d.Raw["legacylinks"] = tt.legacy
			if got := cachedBaseURL("idx", d); got != "" {
				t.Fatalf("stale cache: %q", got)
			}
			if _, err := os.Stat(baseURLCachePath("idx")); !os.IsNotExist(err) {
				t.Fatalf("cache not removed: %v", err)
			}
			bases, _ := baseURLs(config.Indexer{ID: "idx", BaseURL: "auto"}, d)
			if bases[0] != d.BaseURL {
				t.Fatalf("expected primary first, got %v", bases)
			}
		})
	}
}

func TestBaseURLCacheInvalidatesOldFormat(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	d := &cardigann.Definition{BaseURL: "https://primary.test"}
	cacheBaseURL("idx", "https://mirror.test", d, func(string, ...any) {})
	if err := os.WriteFile(baseURLCachePath("idx"), []byte("https://mirror.test\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if got := cachedBaseURL("idx", d); got != "" {
		t.Fatalf("stale cache: %q", got)
	}
}
