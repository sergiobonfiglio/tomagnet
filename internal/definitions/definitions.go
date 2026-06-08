package definitions

import (
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"
)

const CacheDir = ".tomagnet/definitions"

// UpstreamRef pins Jackett definitions sync to a known-compatible upstream commit.
const UpstreamRef = "bbd0821c34afbc4c6a1d1b40760247f3bef5f20e"

const apiURL = "https://api.github.com/repos/Jackett/Jackett/contents/src/Jackett.Common/Definitions?ref=" + UpstreamRef

//go:embed bundled/*.yml
var bundled embed.FS

type Metadata struct {
	SyncedAt  time.Time `json:"synced_at"`
	SourceURL string    `json:"source_url"`
	SourceRef string    `json:"source_ref"`
	Files     []string  `json:"files"`
}

func Resolve(id string) (string, error) {
	return ResolveIn(id, CacheDir)
}

func ResolveIn(id, cacheDir string) (string, error) {
	for _, p := range []string{filepath.Join("definitions", id+".yml"), filepath.Join("definitions", id+".yaml"), filepath.Join(cacheDir, id+".yml"), filepath.Join(cacheDir, id+".yaml")} {
		if st, err := os.Stat(p); err == nil && !st.IsDir() {
			return p, nil
		}
	}
	return "", fmt.Errorf("definition not found for %q; run a definitions sync first", id)
}

func Sync() (Metadata, error) {
	return SyncTo(CacheDir)
}

func SyncTo(cacheDir string) (Metadata, error) {
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
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		return Metadata{}, err
	}
	m := Metadata{SyncedAt: time.Now().UTC(), SourceURL: apiURL, SourceRef: UpstreamRef}
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
		if err := os.WriteFile(filepath.Join(cacheDir, it.Name), b, 0o644); err != nil {
			return m, err
		}
		m.Files = append(m.Files, it.Name)
	}
	if err := syncBundled(cacheDir, &m); err != nil {
		return m, err
	}
	b, _ := json.MarshalIndent(m, "", "  ")
	return m, os.WriteFile(filepath.Join(cacheDir, "index.json"), b, 0o644)
}

func syncBundled(cacheDir string, m *Metadata) error {
	entries, err := bundled.ReadDir("bundled")
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		content, err := bundled.ReadFile(filepath.Join("bundled", entry.Name()))
		if err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(cacheDir, entry.Name()), content, 0o644); err != nil {
			return err
		}
		m.Files = appendUnique(m.Files, entry.Name())
	}
	return nil
}

func appendUnique(values []string, value string) []string {
	if slices.Contains(values, value) {
		return values
	}
	return append(values, value)
}
