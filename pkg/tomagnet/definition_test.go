package tomagnet

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestDecodeDefinition(t *testing.T) {
	def, err := DecodeDefinition(strings.NewReader(`id: custom
name: Custom
base_url: https://idx.test
search:
  path: /
  rows:
    selector: results
  fields:
    title:
      selector: title
`))
	if err != nil {
		t.Fatal(err)
	}
	if def.ID != "custom" || def.BaseURL != "https://idx.test" || def.Search.Rows.Selector != "results" || def.Search.Fields["title"].Selector != "title" {
		t.Fatalf("definition = %#v", def)
	}
}

func TestDecodeDefinitionSupportsJackettShorthand(t *testing.T) {
	def, err := DecodeDefinition(strings.NewReader(`id: custom
name: Custom
caps:
  modes:
    search: [q, imdbid]
search:
  path: /
  headers:
    Authorization: ["Bearer token"]
  rows:
    selector: results
  fields:
    title:
      selector: title
      filters:
        - name: append
          args: " world"
login:
  cookies: ["JAVA=OK"]
`))
	if err != nil {
		t.Fatal(err)
	}
	if got := def.Caps.Modes["search"].Params[0].Name; got != "q" {
		t.Fatalf("search param = %q, want q", got)
	}
	if got := def.Caps.Modes["search"].Params[1].Name; got != "imdbid" {
		t.Fatalf("search param = %q, want imdbid", got)
	}
	if got := def.Search.Fields["title"].Filters[0].Args[0]; got != " world" {
		t.Fatalf("filter arg = %q, want %q", got, " world")
	}
	if got := def.Search.Headers["Authorization"]; got != "Bearer token" {
		t.Fatalf("header = %q, want %q", got, "Bearer token")
	}
	if got := def.Login.Cookies["JAVA"]; got != "OK" {
		t.Fatalf("cookie = %q, want %q", got, "OK")
	}
}

func TestDecodeDefinitionSupportsEscapedSlashes(t *testing.T) {
	def, err := DecodeDefinition(strings.NewReader("id: custom\nname: Custom\nsearch:\n  path: /\n  rows:\n    selector: results\n  fields:\n    title:\n      selector: title\n      filters:\n        - name: regexp\n          args: \"torrent-category-(\\\\d+)\\/\"\n"))
	if err != nil {
		t.Fatal(err)
	}
	if got := def.Search.Fields["title"].Filters[0].Args[0]; got != "torrent-category-(\\d+)/" {
		t.Fatalf("filter arg = %q", got)
	}
}

func TestLoadDefinitionJackettCorpusSamples(t *testing.T) {
	tests := []struct {
		name       string
		file       string
		mode       string
		wantParams []string
		wantArg    string
	}{
		{name: "yts", file: "yts.yml", mode: "movie-search", wantParams: []string{"q", "imdbid"}, wantArg: " -YTS"},
		{name: "1337x", file: "1337x.yml", mode: "tv-search", wantParams: []string{"q", "season", "ep"}},
		{name: "thepiratebay", file: "thepiratebay.yml", mode: "search", wantParams: []string{"q"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			def, err := LoadDefinition(filepath.Join("testdata", "definitions", tt.file))
			if err != nil {
				t.Fatal(err)
			}
			mode := def.Caps.Modes[tt.mode]
			if len(mode.Params) != len(tt.wantParams) {
				t.Fatalf("params len = %d, want %d", len(mode.Params), len(tt.wantParams))
			}
			for i, want := range tt.wantParams {
				if got := mode.Params[i].Name; got != want {
					t.Fatalf("param %d = %q, want %q", i, got, want)
				}
			}
			if tt.wantArg != "" {
				if got := def.Search.Fields["title"].Filters[0].Args[0]; got != tt.wantArg {
					t.Fatalf("filter arg = %q, want %q", got, tt.wantArg)
				}
			}
		})
	}
}
