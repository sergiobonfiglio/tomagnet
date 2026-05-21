package tomagnet

import "testing"

func TestSearchOptionsExposeModeAwareFields(t *testing.T) {
	opt := SearchOptions{
		Mode:       "movie-search",
		Season:     "1",
		Episode:    "2",
		IMDBID:     "tt1",
		TMDBID:     "tm1",
		TVDBID:     "tv1",
		DoubanID:   "db1",
		TVMazeID:   "mz1",
		Artist:     "art",
		Album:      "alb",
		Author:     "auth",
		Title:      "ttl",
		Genre:      "gen",
		Year:       "2024",
		Categories: []string{"5000"},
	}

	if opt.Mode != "movie-search" || opt.Season != "1" || opt.Episode != "2" || opt.IMDBID != "tt1" || opt.TMDBID != "tm1" || opt.TVDBID != "tv1" || opt.DoubanID != "db1" || opt.TVMazeID != "mz1" || opt.Artist != "art" || opt.Album != "alb" || opt.Author != "auth" || opt.Title != "ttl" || opt.Genre != "gen" || opt.Year != "2024" || len(opt.Categories) != 1 {
		t.Fatalf("opt=%#v", opt)
	}
}
