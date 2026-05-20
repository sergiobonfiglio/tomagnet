package search

import (
	"testing"

	"github.com/sergiobonfiglio/tomagnet/internal/cardigann"
)

func TestBuildLoginRequestFromFormPage(t *testing.T) {
	d := &cardigann.Definition{BaseURL: "https://idx.test", Config: map[string]string{"username": "u", "password": "p"}, Raw: map[string]any{"login": map[string]any{
		"path":   "/login",
		"method": "form",
		"form":   `form[action="/takelogin"]`,
		"inputs": map[string]any{"username": "{{ .Config.username }}", "password": "{{ .Config.password }}"},
		"selectorinputs": map[string]any{
			"csrf": map[string]any{"selector": `input[name="csrf"]`, "attribute": "value"},
		},
	}}}
	html := `<html><body><form action="/takelogin"><input name="csrf" value="abc"></form></body></html>`
	r := buildLoginRequest(d, html)
	if r.Method != "post" || r.Path != "/takelogin" || r.Inputs["username"] != "u" || r.Inputs["password"] != "p" || r.Inputs["csrf"] != "abc" {
		t.Fatalf("got %#v", r)
	}
}

func TestBuildLoginRequestSubmitPathOverridesFormAction(t *testing.T) {
	d := &cardigann.Definition{BaseURL: "https://idx.test", Config: map[string]string{}, Raw: map[string]any{"login": map[string]any{
		"path":       "/login",
		"method":     "form",
		"form":       `form[action="/takelogin"]`,
		"submitpath": "/ajax/login",
	}}}
	html := `<html><body><form action="/takelogin"></form></body></html>`
	r := buildLoginRequest(d, html)
	if r.Path != "/ajax/login" {
		t.Fatalf("got %#v", r)
	}
}

func TestBuildLoginRequestSelectorInputsApplyFilters(t *testing.T) {
	d := &cardigann.Definition{BaseURL: "https://idx.test", Config: map[string]string{}, Raw: map[string]any{"login": map[string]any{
		"path":   "/login",
		"method": "form",
		"form":   `form[action="/takelogin"]`,
		"selectorinputs": map[string]any{
			"securitytoken": map[string]any{
				"selector": `script`,
				"filters":  []any{map[string]any{"name": "regexp", "args": `stKey: "(.+?)",`}},
			},
		},
	}}}
	html := `<html><body><form action="/takelogin"></form><script>var x = { stKey: "abc123", test: 1 };</script></body></html>`
	r := buildLoginRequest(d, html)
	if r.Inputs["securitytoken"] != "abc123" {
		t.Fatalf("got %#v", r)
	}
}
