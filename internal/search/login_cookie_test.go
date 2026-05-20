package search

import (
	"context"
	"errors"
	"testing"

	"github.com/sergiobonfiglio/tomagnet/internal/cardigann"
	"github.com/sergiobonfiglio/tomagnet/internal/fetch"
)

func TestLoginCookieHeaderUsesCookieMethodInput(t *testing.T) {
	d := &cardigann.Definition{BaseURL: "https://idx.test", Config: map[string]string{"cookie": "uid=1; pass=abc"}, Raw: map[string]any{"login": map[string]any{
		"method": "cookie",
		"inputs": map[string]any{"cookie": "{{ .Config.cookie }}"},
	}}}
	login := cardigann.LoginRequest(d)
	got := loginCookieHeader(login)
	if got != "uid=1; pass=abc" {
		t.Fatalf("got %q", got)
	}
}

func TestLoginCookieHeaderPrefersExplicitHeader(t *testing.T) {
	d := &cardigann.Definition{BaseURL: "https://idx.test", Config: map[string]string{}, Raw: map[string]any{"login": map[string]any{
		"method":  "cookie",
		"headers": map[string]any{"Cookie": "a=1"},
		"inputs":  map[string]any{"cookie": "b=2"},
	}}}
	login := cardigann.LoginRequest(d)
	got := loginCookieHeader(login)
	if got != "a=1" {
		t.Fatalf("got %q", got)
	}
}

func TestVerifyLoginPassesWhenSelectorFound(t *testing.T) {
	d := &cardigann.Definition{BaseURL: "https://idx.test", Config: map[string]string{}, Raw: map[string]any{"login": map[string]any{
		"test": map[string]any{"path": "/", "selector": "a.logout"},
	}}}
	var got fetch.Request
	err := verifyLogin(context.Background(), d, map[string]string{"__raw__": "sid=1"}, func(ctx context.Context, req fetch.Request) (fetch.Response, error) {
		got = req
		return fetch.Response{Body: []byte(`<a class="logout"></a>`), ContentType: "text/html"}, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.Path != "/" || got.Headers["Cookie"] != "sid=1" {
		t.Fatalf("bad req: %#v", got)
	}
}

func TestVerifyLoginFailsWhenSelectorMissing(t *testing.T) {
	d := &cardigann.Definition{BaseURL: "https://idx.test", Config: map[string]string{}, Raw: map[string]any{"login": map[string]any{
		"test": map[string]any{"path": "/", "selector": "a.logout"},
	}}}
	err := verifyLogin(context.Background(), d, nil, func(ctx context.Context, req fetch.Request) (fetch.Response, error) {
		return fetch.Response{Body: []byte(`<div>guest</div>`), ContentType: "text/html"}, nil
	})
	if err == nil || !errors.Is(err, errLoginTestFailed) {
		t.Fatalf("err = %v", err)
	}
}

func TestLoginCaptchaUnsupported(t *testing.T) {
	d := &cardigann.Definition{Raw: map[string]any{"login": map[string]any{
		"path":    "/login",
		"captcha": map[string]any{"type": "image", "selector": "img.captcha", "input": "code"},
	}}}
	err := loginUnsupported(d)
	if err == nil {
		t.Fatal("expected error")
	}
}
