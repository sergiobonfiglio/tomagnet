package tomagnet

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/sergiobonfiglio/tomagnet/internal/cardigann"
	internalconfig "github.com/sergiobonfiglio/tomagnet/internal/config"
	internalsearch "github.com/sergiobonfiglio/tomagnet/internal/search"
)

// Indexer identifies an indexer to query.
//
// ID must match a synced or locally available definition. BaseURL may be empty
// or set to "auto" to resolve from the definition. If Definition is nil,
// Search loads it by ID.
type Indexer struct {
	// ID is the definition id, such as "yts" or "therarbg".
	ID string
	// BaseURL overrides the definition base URL. Use "auto" to let tomagnet
	// choose from the definition's known links.
	BaseURL string
	// TimeoutSeconds overrides the request timeout for this indexer.
	TimeoutSeconds int
	// Definition provides the already loaded definition to use. If nil, Search
	// loads the definition by ID via LoadDefinitionByID.
	Definition *Definition
}

// Query describes a search intent.
//
// Set exactly one of Text, Movie, or TV. Search uses the intent and the
// definition capabilities to choose a mode and parameters.
type Query struct {
	// Text is a plain free-text search.
	Text string
	// Movie is a movie-oriented search intent.
	Movie *MovieQuery
	// TV is a TV-oriented search intent.
	TV *TVQuery
}

// MovieQuery describes a movie search.
//
// If Text is empty, Search derives a text query from Title and Year, then falls
// back to IMDBID or TMDBID.
type MovieQuery struct {
	// Text is the explicit free-text query to send when you do not want tomagnet
	// to derive one from the other fields.
	Text string
	// Title is the movie title.
	Title string
	// Year is the release year.
	Year int
	// IMDBID is the external IMDb identifier, for example "tt1160419".
	IMDBID string
	// TMDBID is the external TMDb identifier.
	TMDBID string
}

// TVQuery describes a TV search.
//
// If Text is empty, Search derives a text query from Title, Year, Season, and
// Episode, then falls back to the first non-empty external ID supported by the
// definition.
type TVQuery struct {
	// Text is the explicit free-text query to send when you do not want tomagnet
	// to derive one from the other fields.
	Text string
	// Title is the series title.
	Title string
	// Year is the release year.
	Year int
	// Season is the season number.
	Season int
	// Episode is the episode number.
	Episode int
	// IMDBID is the external IMDb identifier.
	IMDBID string
	// TMDBID is the external TMDb identifier.
	TMDBID string
	// TVDBID is the external TVDb identifier.
	TVDBID string
	// DoubanID is the external Douban identifier when supported by a definition.
	DoubanID string
	// TVMazeID is the external TVMaze identifier when supported by a definition.
	TVMazeID string
}

// SearchOptions configures Search.
//
// Categories are passed to indexers that support category filtering. Limit is
// applied per indexer. If Concurrency is zero or negative, Search uses the
// internal default.
type SearchOptions struct {
	// Query is the high-level user intent to execute.
	Query Query
	// Categories is an optional normalized category filter.
	Categories []string
	// Indexers is the set of backends to query.
	Indexers []Indexer
	// Limit is the maximum number of results to keep per indexer. Zero means use
	// the internal default behavior of the search engine.
	Limit int
	// Concurrency limits how many indexers may run at once. Zero or negative
	// values use the internal default.
	Concurrency int
	// Debug receives low-level debug log lines when non-nil.
	Debug func(string, ...any)
}

// Result is one normalized search hit returned by an indexer.
type Result struct {
	// Title is the normalized release title.
	Title string
	// Indexer is the indexer id that produced the result.
	Indexer string
	// SizeBytes is the result size in bytes when available.
	SizeBytes int64
	// Seeders is the reported seeder count when available.
	Seeders int
	// Leechers is the reported leecher count when available.
	Leechers int
	// PublishedAt is the normalized publication time when available.
	PublishedAt *time.Time
	// Category is the normalized or source category label when available.
	Category string
	// MagnetURL is the magnet link when available.
	MagnetURL string
	// DownloadURL is the direct download URL when available.
	DownloadURL string
	// InfoHash is the torrent info hash when available.
	InfoHash string
	// DetailsURL is the details page URL when available.
	DetailsURL string
}

// Error describes a per-indexer failure.
type Error struct {
	// Indexer is the id of the failing indexer.
	Indexer string
	// Stage is the failing stage, such as "definition" or "search".
	Stage string
	// Message is the human-readable failure message.
	Message string
}

// Meta summarizes a completed search.
type Meta struct {
	// Query is the effective high-level text query derived from the input intent.
	Query string
	// StartedAt is the UTC time when Search started.
	StartedAt time.Time
	// DurationMS is the total elapsed time in milliseconds.
	DurationMS int64
	// IndexersRequested is the number of indexers requested.
	IndexersRequested int
	// IndexersSucceeded is the number of indexers that completed without error.
	IndexersSucceeded int
	// IndexersFailed is the number of indexers that returned an error.
	IndexersFailed int
	// TotalResults is the total number of normalized results returned.
	TotalResults int
}

// Response is the result of Search.
type Response struct {
	// Results contains the merged normalized hits from all successful indexers.
	Results []Result
	// Errors contains per-indexer failures. One indexer may fail while others
	// still return results.
	Errors []Error
	// Meta summarizes the completed search.
	Meta Meta
}

// Search runs an intent-based query against one or more indexers.
//
// Search keeps callers indexer-agnostic. It inspects each definition, chooses
// the most specific supported mode, omits unset optional parameters, and
// returns normalized results and per-indexer errors.
func Search(ctx context.Context, opt SearchOptions) Response {
	startedAt := time.Now().UTC()
	queryText := strings.TrimSpace(opt.Query.text())
	out := Response{
		Results: []Result{},
		Errors:  []Error{},
		Meta: Meta{
			Query:             queryText,
			StartedAt:         startedAt,
			IndexersRequested: len(opt.Indexers),
		},
	}

	for _, idx := range opt.Indexers {
		definition := idx.Definition
		if definition == nil {
			var err error
			definition, err = LoadDefinitionByID(idx.ID)
			if err != nil {
				out.Errors = append(out.Errors, Error{Indexer: idx.ID, Stage: "definition", Message: err.Error()})
				continue
			}
		}

		planned := planSearch(definition, opt.Query, opt.Categories)
		planned.Indexers = []internalconfig.Indexer{{ID: idx.ID, BaseURL: idx.BaseURL, TimeoutSeconds: idx.TimeoutSeconds}}
		planned.Definitions = map[string]*cardigann.Definition{idx.ID: definition.cardigann()}
		planned.Limit = opt.Limit
		planned.Concurrency = 1
		planned.Debug = opt.Debug

		r := internalsearch.Run(ctx, planned)
		appendResults(&out, r)
	}

	out.Meta.IndexersFailed = len(out.Errors)
	out.Meta.IndexersSucceeded = out.Meta.IndexersRequested - out.Meta.IndexersFailed
	out.Meta.TotalResults = len(out.Results)
	out.Meta.DurationMS = time.Since(startedAt).Milliseconds()
	return out
}

func appendResults(out *Response, in internalsearch.Response) {
	for _, res := range in.Results {
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
	for _, err := range in.Errors {
		out.Errors = append(out.Errors, Error{Indexer: err.Indexer, Stage: err.Stage, Message: err.Message})
	}
}

func planSearch(definition *Definition, query Query, categories []string) internalsearch.Options {
	switch {
	case query.Movie != nil:
		return planMovieSearch(definition, *query.Movie, categories)
	case query.TV != nil:
		return planTVSearch(definition, *query.TV, categories)
	default:
		return internalsearch.Options{Query: strings.TrimSpace(query.Text), Categories: categories}
	}
}

func planMovieSearch(definition *Definition, query MovieQuery, categories []string) internalsearch.Options {
	planned := internalsearch.Options{
		Query:      movieText(query),
		Mode:       "search",
		Title:      strings.TrimSpace(query.Title),
		Year:       intString(query.Year),
		IMDBID:     strings.TrimSpace(query.IMDBID),
		TMDBID:     strings.TrimSpace(query.TMDBID),
		Categories: categories,
	}
	if supportsExplicitMode(definition, "movie-search") && hasAnyModeParam(definition, "movie-search", map[string]string{
		"q":      planned.Query,
		"title":  planned.Title,
		"year":   planned.Year,
		"imdbid": planned.IMDBID,
		"tmdbid": planned.TMDBID,
	}) {
		planned.Mode = "movie-search"
	}
	return planned
}

func planTVSearch(definition *Definition, query TVQuery, categories []string) internalsearch.Options {
	planned := internalsearch.Options{
		Query:      tvText(query),
		Mode:       "search",
		Title:      strings.TrimSpace(query.Title),
		Year:       intString(query.Year),
		Season:     intString(query.Season),
		Episode:    intString(query.Episode),
		IMDBID:     strings.TrimSpace(query.IMDBID),
		TMDBID:     strings.TrimSpace(query.TMDBID),
		TVDBID:     strings.TrimSpace(query.TVDBID),
		DoubanID:   strings.TrimSpace(query.DoubanID),
		TVMazeID:   strings.TrimSpace(query.TVMazeID),
		Categories: categories,
	}
	if supportsExplicitMode(definition, "tv-search") && hasAnyModeParam(definition, "tv-search", map[string]string{
		"q":        planned.Query,
		"title":    planned.Title,
		"year":     planned.Year,
		"season":   planned.Season,
		"ep":       planned.Episode,
		"episode":  planned.Episode,
		"imdbid":   planned.IMDBID,
		"tmdbid":   planned.TMDBID,
		"tvdbid":   planned.TVDBID,
		"doubanid": planned.DoubanID,
		"tvmazeid": planned.TVMazeID,
	}) {
		planned.Mode = "tv-search"
	}
	return planned
}

func supportsExplicitMode(definition *Definition, mode string) bool {
	if definition == nil || definition.Caps.Modes == nil {
		return false
	}
	_, ok := definition.Caps.Modes[mode]
	return ok
}

func hasAnyModeParam(definition *Definition, mode string, values map[string]string) bool {
	searchMode, ok := definition.Caps.Modes[mode]
	if !ok {
		return false
	}
	for _, param := range searchMode.Params {
		if strings.TrimSpace(values[strings.ToLower(param.Name)]) != "" {
			return true
		}
	}
	return false
}

func (q Query) text() string {
	switch {
	case q.Movie != nil:
		return movieText(*q.Movie)
	case q.TV != nil:
		return tvText(*q.TV)
	default:
		return strings.TrimSpace(q.Text)
	}
}

func movieText(query MovieQuery) string {
	if text := strings.TrimSpace(query.Text); text != "" {
		return text
	}
	parts := []string{strings.TrimSpace(query.Title)}
	if query.Year > 0 {
		parts = append(parts, strconv.Itoa(query.Year))
	}
	text := strings.TrimSpace(strings.Join(parts, " "))
	if text != "" {
		return text
	}
	if imdbID := strings.TrimSpace(query.IMDBID); imdbID != "" {
		return imdbID
	}
	return strings.TrimSpace(query.TMDBID)
}

func tvText(query TVQuery) string {
	if text := strings.TrimSpace(query.Text); text != "" {
		return text
	}
	parts := []string{strings.TrimSpace(query.Title)}
	if query.Year > 0 {
		parts = append(parts, strconv.Itoa(query.Year))
	}
	if query.Season > 0 {
		parts = append(parts, "season "+strconv.Itoa(query.Season))
	}
	if query.Episode > 0 {
		parts = append(parts, "episode "+strconv.Itoa(query.Episode))
	}
	text := strings.TrimSpace(strings.Join(parts, " "))
	if text != "" {
		return text
	}
	for _, value := range []string{query.IMDBID, query.TMDBID, query.TVDBID, query.DoubanID, query.TVMazeID} {
		if value = strings.TrimSpace(value); value != "" {
			return value
		}
	}
	return ""
}

func intString(v int) string {
	if v <= 0 {
		return ""
	}
	return strconv.Itoa(v)
}

func value[T any](v *T) T {
	if v == nil {
		var zero T
		return zero
	}
	return *v
}
