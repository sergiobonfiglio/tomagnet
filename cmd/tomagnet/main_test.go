package main

import (
	"testing"
)

func TestSearchCommandAllowsEmptyQuery(t *testing.T) {
	cmd := root()
	found := false
	for _, c := range cmd.Commands() {
		if c.Name() != "search" {
			continue
		}
		found = true
		if err := c.Args(c, []string{}); err != nil {
			t.Fatalf("expected empty query allowed, got %v", err)
		}
	}
	if !found {
		t.Fatal("search command not found")
	}
}

func TestSearchCommandHasModeFlags(t *testing.T) {
	cmd := root()
	for _, c := range cmd.Commands() {
		if c.Name() != "search" {
			continue
		}
		for _, name := range []string{"mode", "season", "episode", "imdbid", "tmdbid", "tvdbid", "doubanid", "tvmazeid", "artist", "album", "author", "title", "genre", "year", "category"} {
			if c.Flags().Lookup(name) == nil {
				t.Fatalf("missing flag %q", name)
			}
		}
		return
	}
	t.Fatal("search command not found")
}

func TestSearchCommandRejectsUnknownMode(t *testing.T) {
	cmd := root()
	for _, c := range cmd.Commands() {
		if c.Name() != "search" {
			continue
		}
		if err := c.Flags().Set("mode", "nope"); err == nil {
			t.Fatal("expected mode validation error")
		}
		return
	}
	t.Fatal("search command not found")
}

func TestRootCommandHasVersion(t *testing.T) {
	cmd := root()
	if cmd.Version == "" {
		t.Fatal("expected root command version")
	}
	if got, want := versionOutput(), "tomagnet 0.1.0"; got != want {
		t.Fatalf("versionOutput() = %q, want %q", got, want)
	}
}
