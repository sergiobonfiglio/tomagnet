package search

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sergiobonfiglio/tomagnet/internal/cardigann"
	"github.com/sergiobonfiglio/tomagnet/internal/config"
)

func TestBaseURLsAutoUsesCachedURLFirst(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	home := t.TempDir()
	t.Setenv("HOME", home)
	d := &cardigann.Definition{BaseURL: "https://primary.test", Raw: map[string]any{"links": []any{"https://primary.test", "https://mirror.test", "https://cached.test"}}}
	path := baseURLCachePath("idx")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(baseURLCacheKey(d)+"\nhttps://cached.test\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	got, auto := baseURLs(config.Indexer{ID: "idx", BaseURL: "auto"}, d)
	want := []string{"https://cached.test", "https://primary.test", "https://mirror.test"}
	if !auto {
		t.Fatal("expected auto")
	}
	if len(got) != len(want) {
		t.Fatalf("got %#v want %#v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %#v want %#v", got, want)
		}
	}
}

func TestCacheBaseURLWritesUserCacheFile(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	home := t.TempDir()
	t.Setenv("HOME", home)

	d := &cardigann.Definition{BaseURL: "https://ok.test"}
	cacheBaseURL("idx", "https://ok.test", d, func(string, ...any) {})
	b, err := os.ReadFile(baseURLCachePath("idx"))
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != baseURLCacheKey(d)+"\nhttps://ok.test\n" {
		t.Fatalf("got %q", string(b))
	}
}
