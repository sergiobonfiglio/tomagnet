package tomagnet

import internalconfig "github.com/sergiobonfiglio/tomagnet/internal/config"

// DefaultIndexers returns tomagnet's default public indexer configuration.
func DefaultIndexers() []Indexer {
	cfg := internalconfig.Default()
	out := make([]Indexer, 0, len(cfg.Indexers))
	for _, idx := range cfg.Indexers {
		out = append(out, Indexer{
			ID:             idx.ID,
			BaseURL:        idx.BaseURL,
			TimeoutSeconds: idx.TimeoutSeconds,
		})
	}
	return out
}
