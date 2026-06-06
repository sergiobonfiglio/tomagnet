package cardigann

import "testing"

func TestRenderQueryVariablesAndBooleans(t *testing.T) {
	got := Render(`{{ if eq .Config.disabled .False }}{{ .Query.Keywords }}|{{ .Query.Q }}{{ end }}`, map[string]string{"disabled": "false"}, map[string]string{"Keywords": "dune two", "Query": "dune+two"}, nil)
	if got != "dune two|dune+two" {
		t.Fatalf("got %q", got)
	}
}

func TestRenderSupportsOrAndCategoryRange(t *testing.T) {
	got := Render(`get-posts/order:-a{{ range .Categories }}:category:{{ . }}{{ end }}{{ if or .Query.IMDBID .Keywords }}:keywords:{{ or .Query.IMDBID .Keywords }}{{ else }}:time:10D{{ end }}:paginate_by:100:format:json/`, nil, map[string]string{"Keywords": "spicci", "Categories": "Movies,TV"}, nil)
	want := `get-posts/order:-a:category:Movies:category:TV:keywords:spicci:paginate_by:100:format:json/`
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}
