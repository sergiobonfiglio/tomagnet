package definitions

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const CacheDir = ".tomagnet/definitions"
const apiURL = "https://api.github.com/repos/Jackett/Jackett/contents/src/Jackett.Common/Definitions?ref=master"

type Metadata struct {
	SyncedAt  time.Time `json:"synced_at"`
	SourceURL string    `json:"source_url"`
	Files     []string  `json:"files"`
}

func Resolve(id string) (string, error) {
	for _, p := range []string{filepath.Join("definitions", id+".yml"), filepath.Join("definitions", id+".yaml"), filepath.Join(CacheDir, id+".yml"), filepath.Join(CacheDir, id+".yaml")} {
		if st, err := os.Stat(p); err == nil && !st.IsDir() {
			return p, nil
		}
	}
	return "", fmt.Errorf("definition not found for %q", id)
}

type ghItem struct {
	Name, DownloadURL string `json:"name","download_url"`
}

func Sync() (Metadata, error) {
	resp, err := http.Get(apiURL)
	if err != nil {
		return Metadata{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return Metadata{}, fmt.Errorf("github discovery: %s", resp.Status)
	}
	var items []struct {
		Name        string `json:"name"`
		DownloadURL string `json:"download_url"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&items); err != nil {
		return Metadata{}, err
	}
	if err := os.MkdirAll(CacheDir, 0o755); err != nil {
		return Metadata{}, err
	}
	m := Metadata{SyncedAt: time.Now().UTC(), SourceURL: apiURL}
	for _, it := range items {
		if !(strings.HasSuffix(it.Name, ".yml") || strings.HasSuffix(it.Name, ".yaml")) {
			continue
		}
		if it.DownloadURL == "" {
			continue
		}
		r, err := http.Get(it.DownloadURL)
		if err != nil {
			return m, err
		}
		if r.StatusCode != 200 {
			r.Body.Close()
			return m, fmt.Errorf("download %s: %s", it.Name, r.Status)
		}
		b, err := io.ReadAll(r.Body)
		r.Body.Close()
		if err != nil {
			return m, err
		}
		if err := os.WriteFile(filepath.Join(CacheDir, it.Name), b, 0o644); err != nil {
			return m, err
		}
		m.Files = append(m.Files, it.Name)
	}
	b, _ := json.MarshalIndent(m, "", "  ")
	return m, os.WriteFile(filepath.Join(CacheDir, "index.json"), b, 0o644)
}
