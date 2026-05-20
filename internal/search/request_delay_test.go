package search

import (
	"context"
	"testing"
	"time"

	"github.com/sergiobonfiglio/tomagnet/internal/fetch"
)

func TestRequesterHonorsRequestDelayBetweenCalls(t *testing.T) {
	now := time.Unix(1700000000, 0)
	var slept []time.Duration
	calls := 0
	r := requester{
		delay: 2500 * time.Millisecond,
		now:   func() time.Time { return now },
		sleep: func(ctx context.Context, d time.Duration) error {
			slept = append(slept, d)
			now = now.Add(d)
			return nil
		},
		do: func(ctx context.Context, req fetch.Request, timeout time.Duration, debug func(string, ...any)) (fetch.Response, error) {
			calls++
			return fetch.Response{}, nil
		},
	}

	if _, err := r.Do(context.Background(), fetch.Request{}); err != nil {
		t.Fatal(err)
	}
	if calls != 1 || len(slept) != 0 {
		t.Fatalf("first call should not sleep: calls=%d slept=%v", calls, slept)
	}

	now = now.Add(1 * time.Second)
	if _, err := r.Do(context.Background(), fetch.Request{}); err != nil {
		t.Fatal(err)
	}
	if calls != 2 {
		t.Fatalf("calls=%d", calls)
	}
	if len(slept) != 1 || slept[0] != 1500*time.Millisecond {
		t.Fatalf("slept=%v", slept)
	}
}

func TestSearchOptionsBuildCardigannOptions(t *testing.T) {
	opt := Options{Query: "dune", Mode: "tv-search", Season: "1", Episode: "2", IMDBID: "tt1", TMDBID: "tm1", TVDBID: "tv1", DoubanID: "db1", TVMazeID: "mz1", Artist: "art", Album: "alb", Author: "auth", Title: "ttl", Genre: "gen", Year: "2024", Categories: []string{"5000"}}
	got := opt.cardigann()
	if got.Keywords != "dune" || got.Mode != "tv-search" || got.Season != "1" || got.Episode != "2" || got.IMDBID != "tt1" || got.TMDBID != "tm1" || got.TVDBID != "tv1" || got.DoubanID != "db1" || got.TVMazeID != "mz1" || got.Artist != "art" || got.Album != "alb" || got.Author != "auth" || got.Title != "ttl" || got.Genre != "gen" || got.Year != "2024" || len(got.Categories) != 1 || got.Categories[0] != "5000" {
		t.Fatalf("got=%#v", got)
	}
}
