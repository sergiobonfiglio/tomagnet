package cardigann

import "testing"

func TestRenderJackettTemplateExpressions(t *testing.T) {
	cfg := map[string]string{"apiurl": "api.example", "enabled": "true"}
	query := map[string]string{"Keywords": "dune", "Categories": "2000,5000"}
	result := map[string]string{"_id": "42"}

	got := Render(`https://{{ .Config.apiurl }}/{{ if and .Keywords .Config.enabled }}q={{ .Keywords }}{{ else }}top{{ end }}/{{ .Result._id }}`, cfg, query, result)
	want := "https://api.example/q=dune/42"
	if got != want {
		t.Fatalf("Render() = %q, want %q", got, want)
	}
}

func TestRenderComparisons(t *testing.T) {
	got := Render(`{{ if eq .Config.mode "json" }}yes{{ else }}no{{ end }} {{ if ne .Keywords "" }}kw{{ end }}`, map[string]string{"mode": "json"}, map[string]string{"Keywords": "x"}, nil)
	if got != "yes kw" {
		t.Fatalf("Render() = %q", got)
	}
}

func TestSearchRequestUsesPathMethodInputsAndHeaders(t *testing.T) {
	d := &Definition{BaseURL: "https://example.test", Config: map[string]string{"token": "abc"}, Raw: map[string]any{
		"search": map[string]any{
			"method":  "post",
			"headers": map[string]any{"X-Test": []any{"{{ .Config.token }}"}},
			"inputs":  map[string]any{"q": "{{ .Keywords }}", "t": "search"},
			"paths": []any{map[string]any{
				"path":   "/search",
				"inputs": map[string]any{"page": "1"},
			}},
		},
	}}

	s := SearchRequest(d, "dune")
	if s.Method != "post" || s.Path != "/search" || s.Inputs["q"] != "dune" || s.Inputs["t"] != "search" || s.Inputs["page"] != "1" || s.Headers["X-Test"] != "abc" {
		t.Fatalf("SearchRequest() = %#v", s)
	}
}

func TestSearchRequestUsesPathFollowRedirectOverride(t *testing.T) {
	d := &Definition{Raw: map[string]any{
		"followredirect": false,
		"search": map[string]any{
			"paths": []any{map[string]any{
				"path":           "/search",
				"followredirect": true,
			}},
		},
	}}

	s := SearchRequest(d, "dune")
	if !s.FollowRedirect {
		t.Fatalf("expected path-level followredirect override: %#v", s)
	}
}

func TestAllowEmptyInputs(t *testing.T) {
	d := &Definition{Raw: map[string]any{"search": map[string]any{"allowEmptyInputs": true}}}
	if !AllowEmptyInputs(d) {
		t.Fatal("expected allowEmptyInputs true")
	}
}

func TestBaseURLsUsesLinksAndLegacyLinks(t *testing.T) {
	d := &Definition{BaseURL: "https://primary.test", Raw: map[string]any{
		"links":       []any{"https://primary.test", "https://mirror.test"},
		"legacylinks": []any{"http://legacy.test", "https://primary.test"},
	}}
	got := BaseURLs(d, "")
	want := []string{"https://primary.test", "https://mirror.test", "http://legacy.test"}
	if len(got) != len(want) {
		t.Fatalf("got %#v want %#v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %#v want %#v", got, want)
		}
	}
}
