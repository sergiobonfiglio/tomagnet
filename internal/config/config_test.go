package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadMissingDefaultConfigUsesTopPublicIndexers(t *testing.T) {
	dir := t.TempDir()
	old, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(old)
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}

	c, err := Load("")
	if err != nil {
		t.Fatal(err)
	}
	idx, err := c.Enabled(nil)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"yts", "limetorrents", "thepiratebay"}
	if len(idx) != len(want) {
		t.Fatalf("got %d indexers want %d", len(idx), len(want))
	}
	for i := range want {
		if idx[i].ID != want[i] {
			t.Fatalf("indexer %d got %q want %q", i, idx[i].ID, want[i])
		}
		if idx[i].BaseURL != "auto" {
			t.Fatalf("indexer %q base url got %q want auto", idx[i].ID, idx[i].BaseURL)
		}
		if idx[i].TimeoutSeconds != 15 {
			t.Fatalf("indexer %q timeout got %d want 15", idx[i].ID, idx[i].TimeoutSeconds)
		}
	}
}

func TestLoadConfigIndexersEnabledByPresenceAndDisabledIndexers(t *testing.T) {
	path := filepath.Join(t.TempDir(), "tomagnet.yaml")
	if err := os.WriteFile(path, []byte(`default_timeout_seconds: 9
disabled_indexers:
  - yts
indexers:
  - id: yts
  - id: 1337x
    base_url: https://1337x.to
`), 0o644); err != nil {
		t.Fatal(err)
	}

	c, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	idx, err := c.Enabled(nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(idx) != 1 || idx[0].ID != "1337x" {
		t.Fatalf("got %#v want only 1337x", idx)
	}
	idx, err = c.Enabled([]string{"yts"})
	if err != nil {
		t.Fatal(err)
	}
	if len(idx) != 1 || idx[0].ID != "yts" {
		t.Fatalf("got %#v want only yts", idx)
	}
}

func TestEnabledFiltersOnlyRequestedIndexers(t *testing.T) {
	c := &Config{Indexers: []Indexer{{ID: "therarbg"}, {ID: "bitmagnet"}, {ID: "btdig"}}}
	if err := normalize(c); err != nil {
		t.Fatal(err)
	}
	idx, err := c.Enabled([]string{"therarbg"})
	if err != nil {
		t.Fatal(err)
	}
	if len(idx) != 1 || idx[0].ID != "therarbg" {
		t.Fatalf("got %#v want only therarbg", idx)
	}
}

func TestEnabledSynthesizesExplicitIndexerNotInConfig(t *testing.T) {
	c := &Config{DefaultTimeoutSeconds: 9, Indexers: []Indexer{{ID: "bitmagnet", BaseURL: "http://example"}}}
	if err := normalize(c); err != nil {
		t.Fatal(err)
	}
	idx, err := c.Enabled([]string{"therarbg"})
	if err != nil {
		t.Fatal(err)
	}
	if len(idx) != 1 {
		t.Fatalf("got %#v want 1 indexer", idx)
	}
	if idx[0].ID != "therarbg" || idx[0].BaseURL != "auto" || idx[0].TimeoutSeconds != 9 {
		t.Fatalf("got %#v want synthesized therarbg", idx[0])
	}
}
