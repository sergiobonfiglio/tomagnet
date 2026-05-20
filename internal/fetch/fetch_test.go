package fetch

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestDoGETAddsInputsAndHeaders(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Query().Get("q") != "dune" || r.URL.Query().Get("page") != "1" || r.Header.Get("X-Test") != "abc" {
			t.Fatalf("bad request method=%s url=%s header=%s", r.Method, r.URL.String(), r.Header.Get("X-Test"))
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[]`))
	}))
	defer ts.Close()

	r, err := Do(context.Background(), Request{Method: "get", Base: ts.URL, Path: "/search", Inputs: map[string]string{"q": "dune", "page": "1"}, Headers: map[string]string{"X-Test": "abc"}}, time.Second, nil)
	if err != nil {
		t.Fatal(err)
	}
	if r.ContentType != "application/json" {
		t.Fatalf("content-type %q", r.ContentType)
	}
}

func TestDoPOSTSendsFormInputs(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		if r.Method != http.MethodPost || string(body) != "q=dune" || r.Header.Get("Content-Type") != "application/x-www-form-urlencoded" {
			t.Fatalf("bad post method=%s body=%q type=%s", r.Method, string(body), r.Header.Get("Content-Type"))
		}
		_, _ = w.Write([]byte(`ok`))
	}))
	defer ts.Close()

	if _, err := Do(context.Background(), Request{Method: "post", Base: ts.URL, Path: "/search", Inputs: map[string]string{"q": "dune"}}, time.Second, nil); err != nil {
		t.Fatal(err)
	}
}

func TestDoCanDisableRedirectFollowing(t *testing.T) {
	redirected := false
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/start":
			http.Redirect(w, r, "/final", http.StatusFound)
		case "/final":
			redirected = true
			_, _ = w.Write([]byte("final"))
		default:
			http.NotFound(w, r)
		}
	}))
	defer ts.Close()

	fr := false
	r, err := Do(context.Background(), Request{Method: "get", Base: ts.URL, Path: "/start", FollowRedirect: &fr}, time.Second, nil)
	if err != nil {
		t.Fatal(err)
	}
	if redirected {
		t.Fatal("redirect should not be followed")
	}
	if string(r.Body) == "final" {
		t.Fatalf("unexpected body %q", string(r.Body))
	}
}
