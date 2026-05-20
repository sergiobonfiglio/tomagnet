package fetch

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestDoReturnsSetCookies(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.SetCookie(w, &http.Cookie{Name: "sid", Value: "1"})
		_, _ = w.Write([]byte("ok"))
	}))
	defer ts.Close()
	r, err := Do(context.Background(), Request{Base: ts.URL, Path: "/"}, time.Second, nil)
	if err != nil {
		t.Fatal(err)
	}
	if r.Cookies["sid"] != "1" {
		t.Fatalf("cookies = %#v", r.Cookies)
	}
}

func TestSessionPersistsCookiesAcrossRequests(t *testing.T) {
	step := 0
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		step++
		switch step {
		case 1:
			http.SetCookie(w, &http.Cookie{Name: "sid", Value: "1"})
			_, _ = w.Write([]byte("login"))
		case 2:
			if c, err := r.Cookie("sid"); err != nil || c.Value != "1" {
				t.Fatalf("cookie not persisted: %v %v", c, err)
			}
			_, _ = w.Write([]byte("search"))
		default:
			t.Fatalf("unexpected step %d", step)
		}
	}))
	defer ts.Close()

	s := NewSession(time.Second, nil)
	if _, err := s.Do(context.Background(), Request{Base: ts.URL, Path: "/login"}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Do(context.Background(), Request{Base: ts.URL, Path: "/search"}); err != nil {
		t.Fatal(err)
	}
}

func TestSessionAllowsPinnedSelfSignedCertificate(t *testing.T) {
	ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("ok"))
	}))
	defer ts.Close()

	cert := ts.TLS.Certificates[0].Certificate[0]
	sum := sha1.Sum(cert)
	fp := hex.EncodeToString(sum[:])

	s := NewSession(time.Second, nil, fp)
	r, err := s.Do(context.Background(), Request{Base: ts.URL, Path: "/"})
	if err != nil {
		t.Fatal(err)
	}
	if string(r.Body) != "ok" {
		t.Fatalf("body=%q", string(r.Body))
	}
}
