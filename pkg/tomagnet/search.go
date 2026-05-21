package tomagnet

import (
	"context"
	"time"

	internalconfig "github.com/sergiobonfiglio/tomagnet/internal/config"
	internalsearch "github.com/sergiobonfiglio/tomagnet/internal/search"
)

type Indexer struct {
	ID             string
	BaseURL        string
	TimeoutSeconds int
}

type SearchOptions struct {
	Query       string
	Mode        string
	Season      string
	Episode     string
	IMDBID      string
	TMDBID      string
	TVDBID      string
	DoubanID    string
	TVMazeID    string
	Artist      string
	Album       string
	Author      string
	Title       string
	Genre       string
	Year        string
	Categories  []string
	Indexers    []Indexer
	Limit       int
	Concurrency int
	Debug       func(string, ...any)
}

type Result struct {
	Title       string
	Indexer     string
	SizeBytes   int64
	Seeders     int
	Leechers    int
	PublishedAt *time.Time
	Category    string
	MagnetURL   string
	DownloadURL string
	InfoHash    string
	DetailsURL  string
}

type Error struct {
	Indexer string
	Stage   string
	Message string
}

type Meta struct {
	Query             string
	StartedAt         time.Time
	DurationMS        int64
	IndexersRequested int
	IndexersSucceeded int
	IndexersFailed    int
	TotalResults      int
}

type Response struct {
	Results []Result
	Errors  []Error
	Meta    Meta
}

func Search(ctx context.Context, opt SearchOptions) Response {
	indexers := make([]internalconfig.Indexer, 0, len(opt.Indexers))
	for _, idx := range opt.Indexers {
		indexers = append(indexers, internalconfig.Indexer{ID: idx.ID, BaseURL: idx.BaseURL, TimeoutSeconds: idx.TimeoutSeconds})
	}
	r := internalsearch.Run(ctx, internalsearch.Options{Query: opt.Query, Mode: opt.Mode, Season: opt.Season, Episode: opt.Episode, IMDBID: opt.IMDBID, TMDBID: opt.TMDBID, TVDBID: opt.TVDBID, DoubanID: opt.DoubanID, TVMazeID: opt.TVMazeID, Artist: opt.Artist, Album: opt.Album, Author: opt.Author, Title: opt.Title, Genre: opt.Genre, Year: opt.Year, Categories: opt.Categories, Indexers: indexers, Limit: opt.Limit, Concurrency: opt.Concurrency, Debug: opt.Debug})
	out := Response{Meta: Meta{Query: r.Meta.Query, StartedAt: r.Meta.StartedAt, DurationMS: r.Meta.DurationMS, IndexersRequested: r.Meta.IndexersRequested, IndexersSucceeded: r.Meta.IndexersSucceeded, IndexersFailed: r.Meta.IndexersFailed, TotalResults: r.Meta.TotalResults}}
	out.Results = make([]Result, 0, len(r.Results))
	for _, res := range r.Results {
		out.Results = append(out.Results, Result{
			Title:       value(res.Title),
			Indexer:     res.Indexer,
			SizeBytes:   value(res.Size),
			Seeders:     value(res.Seeders),
			Leechers:    value(res.Leechers),
			PublishedAt: res.PublishDate,
			Category:    value(res.Category),
			MagnetURL:   value(res.MagnetURL),
			DownloadURL: value(res.DownloadURL),
			InfoHash:    value(res.InfoHash),
			DetailsURL:  value(res.DetailsURL),
		})
	}
	out.Errors = make([]Error, 0, len(r.Errors))
	for _, err := range r.Errors {
		out.Errors = append(out.Errors, Error{Indexer: err.Indexer, Stage: err.Stage, Message: err.Message})
	}
	return out
}

func value[T any](v *T) T {
	if v == nil {
		var zero T
		return zero
	}
	return *v
}
