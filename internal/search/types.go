package search

import "time"

type Result struct {
	Title       *string    `json:"title"`
	Indexer     string     `json:"indexer"`
	GUID        *string    `json:"guid"`
	Size        *int64     `json:"size"`
	Seeders     *int       `json:"seeders"`
	Leechers    *int       `json:"leechers"`
	PublishDate *time.Time `json:"publish_date"`
	Category    *string    `json:"category"`
	MagnetURL   *string    `json:"magnet_url"`
	DownloadURL *string    `json:"download_url"`
	InfoHash    *string    `json:"infohash"`
	DetailsURL  *string    `json:"details_url"`
}

type Error struct {
	Indexer string `json:"indexer,omitempty"`
	Stage   string `json:"stage,omitempty"`
	Message string `json:"message,omitempty"`
}

type Meta struct {
	Query             string    `json:"query"`
	StartedAt         time.Time `json:"started_at"`
	DurationMS        int64     `json:"duration_ms"`
	IndexersRequested int       `json:"indexers_requested"`
	IndexersSucceeded int       `json:"indexers_succeeded"`
	IndexersFailed    int       `json:"indexers_failed"`
	TotalResults      int       `json:"total_results"`
}

type Response struct {
	Results []Result `json:"results"`
	Errors  []Error  `json:"errors"`
	Meta    Meta     `json:"meta"`
}
