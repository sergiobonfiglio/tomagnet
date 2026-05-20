package cardigann

import "testing"

func TestRenderQueryVariablesAndBooleans(t *testing.T) {
	got := Render(`{{ if eq .Config.disabled .False }}{{ .Query.Keywords }}|{{ .Query.Q }}{{ end }}`, map[string]string{"disabled": "false"}, map[string]string{"Keywords": "dune two", "Query": "dune+two"}, nil)
	if got != "dune two|dune+two" {
		t.Fatalf("got %q", got)
	}
}
