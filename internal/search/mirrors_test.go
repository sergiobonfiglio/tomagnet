package search

import (
	"errors"
	"testing"
)

func TestTryBasesFallsBackToLaterMirror(t *testing.T) {
	calls := []string{}
	got, err := tryBases([]string{"https://a.test", "https://b.test"}, func(base string) (string, error) {
		calls = append(calls, base)
		if base == "https://a.test" {
			return "", errors.New("down")
		}
		return "ok", nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if got != "ok" {
		t.Fatalf("got %q", got)
	}
	if len(calls) != 2 || calls[0] != "https://a.test" || calls[1] != "https://b.test" {
		t.Fatalf("calls=%v", calls)
	}
}
