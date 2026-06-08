package definitions

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func TestSyncBundledWritesBTDigDefinition(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() error = %v", err)
	}
	tmp := t.TempDir()
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("Chdir() error = %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(cwd) })
	if err := os.MkdirAll(CacheDir, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}

	metadata := Metadata{Files: []string{"yts.yml"}}
	if err := syncBundled(&metadata); err != nil {
		t.Fatalf("syncBundled() error = %v", err)
	}
	if !slices.Contains(metadata.Files, "btdig.yml") {
		t.Fatalf("metadata files = %v, want btdig.yml", metadata.Files)
	}

	content, err := os.ReadFile(filepath.Join(CacheDir, "btdig.yml"))
	if err != nil {
		t.Fatalf("ReadFile(btdig.yml) error = %v", err)
	}
	if !strings.Contains(string(content), "id: btdig") {
		t.Fatalf("unexpected btdig definition: %q", string(content))
	}
}

func TestSyncBundledDoesNotDuplicateMetadataFiles(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() error = %v", err)
	}
	tmp := t.TempDir()
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("Chdir() error = %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(cwd) })
	if err := os.MkdirAll(CacheDir, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}

	metadata := Metadata{Files: []string{"btdig.yml"}}
	if err := syncBundled(&metadata); err != nil {
		t.Fatalf("syncBundled() error = %v", err)
	}
	count := 0
	for _, file := range metadata.Files {
		if file == "btdig.yml" {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("btdig.yml count = %d in %v, want 1", count, metadata.Files)
	}
}
