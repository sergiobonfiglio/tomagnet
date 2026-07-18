package config

import (
	"errors"
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

const Path = "tomagnet.yaml"

var DefaultIndexers = []string{"btdig", "yts", "limetorrents", "thepiratebay"}

type Config struct {
	DefaultTimeoutSeconds int       `yaml:"default_timeout_seconds"`
	Concurrency           int       `yaml:"concurrency"`
	DisabledIndexers      []string  `yaml:"disabled_indexers"`
	Indexers              []Indexer `yaml:"indexers"`
}

type Indexer struct {
	ID             string `yaml:"id"`
	TimeoutSeconds int    `yaml:"timeout_seconds"`
	BaseURL        string `yaml:"base_url"`
}

func Load(path string) (*Config, error) {
	if path == "" {
		path = Path
	}
	b, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) && path == Path {
			c := Default()
			return &c, nil
		}
		return nil, err
	}
	var c Config
	if err := yaml.Unmarshal(b, &c); err != nil {
		return nil, err
	}
	if err := normalize(&c); err != nil {
		return nil, err
	}
	return &c, nil
}

func Default() Config {
	c := Config{}
	for _, id := range DefaultIndexers {
		c.Indexers = append(c.Indexers, Indexer{ID: id, BaseURL: "auto"})
	}
	_ = normalize(&c)
	return c
}

func normalize(c *Config) error {
	if c.DefaultTimeoutSeconds <= 0 {
		c.DefaultTimeoutSeconds = 30
	}
	if c.Concurrency <= 0 {
		c.Concurrency = 4
	}
	seen := map[string]bool{}
	for i := range c.Indexers {
		idx := &c.Indexers[i]
		if idx.ID == "" {
			return fmt.Errorf("indexer id required")
		}
		if seen[idx.ID] {
			return fmt.Errorf("duplicate indexer %q", idx.ID)
		}
		seen[idx.ID] = true
		if idx.TimeoutSeconds <= 0 {
			idx.TimeoutSeconds = c.DefaultTimeoutSeconds
		}
		if idx.BaseURL == "" {
			idx.BaseURL = "auto"
		}
	}
	return nil
}

func (c *Config) Enabled(ids []string) ([]Indexer, error) {
	if len(ids) > 0 {
		byID := map[string]Indexer{}
		for _, idx := range c.Indexers {
			byID[idx.ID] = idx
		}
		out := make([]Indexer, 0, len(ids))
		seen := map[string]bool{}
		for _, id := range ids {
			if seen[id] {
				continue
			}
			seen[id] = true
			if idx, ok := byID[id]; ok {
				out = append(out, idx)
				continue
			}
			out = append(out, Indexer{ID: id, TimeoutSeconds: c.DefaultTimeoutSeconds, BaseURL: "auto"})
		}
		return out, nil
	}

	disabled := map[string]bool{}
	for _, id := range c.DisabledIndexers {
		disabled[id] = true
	}
	var out []Indexer
	for _, idx := range c.Indexers {
		if disabled[idx.ID] {
			continue
		}
		out = append(out, idx)
	}
	return out, nil
}
