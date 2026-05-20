package search

import (
	"testing"

	"github.com/sergiobonfiglio/tomagnet/internal/cardigann"
)

func TestBuildLoginRequestResolvesSelectorKeysToInputNames(t *testing.T) {
	d := &cardigann.Definition{BaseURL: "https://idx.test", Config: map[string]string{"username": "u", "password": "p"}, Raw: map[string]any{"login": map[string]any{
		"path":      "/login",
		"method":    "form",
		"form":      `form[action="/takelogin"]`,
		"selectors": true,
		"inputs": map[string]any{
			`input[placeholder="Username"]`: "{{ .Config.username }}",
			`input[placeholder="Password"]`: "{{ .Config.password }}",
		},
	}}}
	html := `<html><body><form action="/takelogin"><input name="user_name" placeholder="Username"><input name="user_pass" placeholder="Password"></form></body></html>`
	r := buildLoginRequest(d, html)
	if r.Inputs["user_name"] != "u" || r.Inputs["user_pass"] != "p" {
		t.Fatalf("got %#v", r)
	}
	if _, ok := r.Inputs[`input[placeholder="Username"]`]; ok {
		t.Fatalf("unexpected selector key in inputs: %#v", r)
	}
}
