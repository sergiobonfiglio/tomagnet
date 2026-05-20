package config

import (
	"errors"
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

const Path = "tomagnet.yaml"

var DefaultIndexers = []string{"yts", "1337x", "thepiratebay"}

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
		c.DefaultTimeoutSeconds = 15
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
	want := map[string]bool{}
	for _, id := range ids {
		want[id] = true
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
		if len(want) == 0 || want[idx.ID] {
			out = append(out, idx)
			delete(want, idx.ID)
		}
	}
	for id := range want {
		if disabled[id] {
			return nil, fmt.Errorf("indexer %q disabled", id)
		}
		return nil, fmt.Errorf("indexer %q not configured", id)
	}
	return out, nil
}
