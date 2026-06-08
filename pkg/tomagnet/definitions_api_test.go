package tomagnet

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadDefinitionByIDFromUsesProvidedCacheDir(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() error = %v", err)
	}
	tmp := t.TempDir()
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("Chdir() error = %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(cwd) })

	cacheDir := filepath.Join(".custom", "definitions")
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(cacheDir, "demo.yml"), []byte(`id: demo
name: Demo
`), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	definition, err := LoadDefinitionByIDFrom("demo", cacheDir)
	if err != nil {
		t.Fatalf("LoadDefinitionByIDFrom() error = %v", err)
	}
	if definition.ID != "demo" {
		t.Fatalf("definition ID = %q, want demo", definition.ID)
	}
}
