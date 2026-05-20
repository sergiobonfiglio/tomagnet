package cardigann

import "testing"

func TestSearchRequestRendersCookies(t *testing.T) {
	d := &Definition{Config: map[string]string{"uid": "7", "pass": "abc"}, Raw: map[string]any{"search": map[string]any{
		"path":    "/search",
		"cookies": map[string]any{"uid": "{{ .Config.uid }}", "pass": "{{ .Config.pass }}"},
	}}}
	got := SearchRequest(d, "dune")
	if got.Headers["Cookie"] != "pass=abc; uid=7" {
		t.Fatalf("Cookie = %q", got.Headers["Cookie"])
	}
}
