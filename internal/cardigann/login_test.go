package cardigann

import "testing"

func TestLoginRequestRendersPathInputsAndHeaders(t *testing.T) {
	d := &Definition{BaseURL: "https://idx.test", Config: map[string]string{"username": "u", "password": "p"}, Raw: map[string]any{"login": map[string]any{
		"path":    "/login",
		"method":  "post",
		"inputs":  map[string]any{"user": "{{ .Config.username }}", "pass": "{{ .Config.password }}"},
		"headers": map[string]any{"X-Login": []any{"1"}},
	}}}
	got := LoginRequest(d)
	if got.Method != "post" || got.Path != "/login" || got.Inputs["user"] != "u" || got.Inputs["pass"] != "p" || got.Headers["X-Login"] != "1" {
		t.Fatalf("LoginRequest() = %#v", got)
	}
}

func TestLoginRequestRendersTemplatedMethod(t *testing.T) {
	d := &Definition{Config: map[string]string{"use_post": "1"}, Raw: map[string]any{"login": map[string]any{
		"path":   "/login",
		"method": "{{ if .Config.use_post }}post{{ else }}get{{ end }}",
	}}}
	if got := LoginRequest(d).Method; got != "post" {
		t.Fatalf("method = %q, want post", got)
	}
}
