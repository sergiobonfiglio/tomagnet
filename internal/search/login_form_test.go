package search

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sergiobonfiglio/tomagnet/internal/cardigann"
	"github.com/sergiobonfiglio/tomagnet/internal/config"
)

func TestBuildLoginRequestFromFormPage(t *testing.T) {
	d := &cardigann.Definition{BaseURL: "https://idx.test", Config: map[string]string{"username": "u", "password": "p"}, Raw: map[string]any{"login": map[string]any{
		"path":   "forum/login",
		"method": "form",
		"form":   `form[action="./takelogin"]`,
		"inputs": map[string]any{"username": "{{ .Config.username }}", "password": "{{ .Config.password }}"},
	}}}
	html := `<html><body><form action="./takelogin"><input name="username" value=""><input name="csrf" type="hidden" value="abc"><input name="remember" type="checkbox" value="yes"><input name="login" type="submit" value="Login"></form></body></html>`
	r := buildLoginRequest(d, html, "https://idx.test/forum/login")
	if r.Method != "post" || r.Path != "https://idx.test/forum/takelogin" || r.Inputs["username"] != "u" || r.Inputs["password"] != "p" || r.Inputs["csrf"] != "abc" || r.Inputs["login"] != "Login" {
		t.Fatalf("got %#v", r)
	}
	if _, ok := r.Inputs["remember"]; ok {
		t.Fatalf("unchecked checkbox submitted: %#v", r.Inputs)
	}
}

func TestRunFormLoginSubmitsRelativeActionAndHiddenInputs(t *testing.T) {
	const (
		username = "fixture-user"
		password = "fixture-password"
	)
	var loginReceived, searchReceived bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/forum/login":
			_, _ = w.Write([]byte(`<form id="login" action="./session?sid=action-token"><input name="username"><input name="password"><input type="hidden" name="form_token" value="form-token"><input type="submit" name="login" value="Login"></form>`))
		case "/forum/session":
			if r.Method != http.MethodPost {
				http.Error(w, "wrong method", http.StatusMethodNotAllowed)
				return
			}
			if err := r.ParseForm(); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			if r.URL.Query().Get("sid") != "action-token" || r.Form.Get("username") != username || r.Form.Get("password") != password || r.Form.Get("form_token") != "form-token" || r.Form.Get("login") != "Login" {
				http.Error(w, "invalid login form", http.StatusUnauthorized)
				return
			}
			loginReceived = true
			http.SetCookie(w, &http.Cookie{Name: "session", Value: "valid", Path: "/"})
		case "/forum/home":
			if cookie, err := r.Cookie("session"); err == nil && cookie.Value == "valid" {
				_, _ = w.Write([]byte(`<div class="authenticated"></div>`))
				return
			}
			_, _ = w.Write([]byte(`<div class="signed-out"></div>`))
		case "/forum/search":
			if cookie, err := r.Cookie("session"); err != nil || cookie.Value != "valid" {
				http.Error(w, "authentication required", http.StatusUnauthorized)
				return
			}
			searchReceived = true
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"results":[{"title":"Result","magnet":"magnet:?xt=urn:btih:abc"}]}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	d := &cardigann.Definition{BaseURL: server.URL, Config: map[string]string{"username": username, "password": password}, Raw: map[string]any{
		"login": map[string]any{
			"path":   "/forum/login",
			"method": "form",
			"form":   "form#login",
			"inputs": map[string]any{"username": "{{ .Config.username }}", "password": "{{ .Config.password }}"},
			"test":   map[string]any{"path": "/forum/home", "selector": ".authenticated"},
		},
		"search": map[string]any{
			"path":   "/forum/search",
			"inputs": map[string]any{"q": "{{ .Keywords }}"},
			"rows":   map[string]any{"selector": "results"},
			"fields": map[string]any{
				"title":  map[string]any{"selector": "title"},
				"magnet": map[string]any{"selector": "magnet"},
			},
		},
	}}

	results, err := runOneBase(context.Background(), d, config.Indexer{ID: "form", TimeoutSeconds: 5}, Options{Query: "Dune", Mode: "search"})
	if err != nil {
		t.Fatal(err)
	}
	if !loginReceived || !searchReceived || len(results) != 1 {
		t.Fatalf("login=%t search=%t results=%#v", loginReceived, searchReceived, results)
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
	r := buildLoginRequest(d, html, "https://idx.test/login")
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
	r := buildLoginRequest(d, html, "https://idx.test/login")
	if r.Inputs["securitytoken"] != "abc123" {
		t.Fatalf("got %#v", r)
	}
}
