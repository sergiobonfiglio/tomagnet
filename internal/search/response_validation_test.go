package search

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/sergiobonfiglio/tomagnet/internal/cardigann"
	"github.com/sergiobonfiglio/tomagnet/internal/config"
)

func TestSearchResponseValidation(t *testing.T) {
	for _, tt := range []struct {
		name            string
		status          int
		body, wantError string
	}{
		{"empty", 200, "", "empty body"},
		{"whitespace", 200, " \n\t", "empty body"},
		{"redirect", 301, "", "unexpected HTTP status 301"},
		{"redirect with body", 302, "<html>Moved</html>", "unexpected HTTP status 302"},
		{"no content", 204, "", "unexpected HTTP status 204"},
		{"valid no results", 200, "<html><body>No results</body></html>", ""},
	} {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Location", "/elsewhere")
				w.WriteHeader(tt.status)
				_, _ = w.Write([]byte(tt.body))
			}))
			defer server.Close()
			d := &cardigann.Definition{BaseURL: server.URL, Raw: map[string]any{}}
			_, err := runOneBase(context.Background(), d, config.Indexer{ID: "test", TimeoutSeconds: 5}, Options{})
			if tt.wantError == "" {
				if err != nil {
					t.Fatal(err)
				}
			} else if err == nil || !strings.Contains(err.Error(), tt.wantError) {
				t.Fatalf("got %v, want %q", err, tt.wantError)
			}
		})
	}
}

func TestInvalidSearchResponseFallsBack(t *testing.T) {
	for _, status := range []int{http.StatusOK, http.StatusMovedPermanently} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			bad := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(status) }))
			defer bad.Close()
			calls := 0
			good := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls++
				_, _ = w.Write([]byte("<html><body>No results</body></html>"))
			}))
			defer good.Close()
			_, err := tryBases([]string{bad.URL, good.URL}, func(base string) ([]Result, error) {
				return runOneBase(context.Background(), &cardigann.Definition{BaseURL: base, Raw: map[string]any{}}, config.Indexer{ID: "test", TimeoutSeconds: 5}, Options{})
			})
			if err != nil {
				t.Fatal(err)
			}
			if calls != 1 {
				t.Fatalf("good mirror called %d times", calls)
			}
		})
	}
}
