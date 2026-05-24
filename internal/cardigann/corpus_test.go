package cardigann

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestAllDownloadedDefinitionsUseSupportedFeatures(t *testing.T) {
	files, err := filepath.Glob(filepath.Join("..", "..", ".tomagnet", "definitions", "*.yml"))
	if err != nil {
		t.Fatal(err)
	}
	if len(files) == 0 {
		t.Skip("no synced definition files; run `tomagnet definitions sync` to enable corpus test")
	}
	for _, path := range files {
		path := path
		t.Run(filepath.Base(path), func(t *testing.T) {
			d, err := Load(path)
			if err != nil {
				t.Fatalf("load: %v", err)
			}
			if err := CheckSupport(d); err != nil {
				t.Fatalf("support: %v", err)
			}
			s := SearchRequest(d, "dune")
			if strings.TrimSpace(s.Path) == "" {
				t.Fatalf("empty search path")
			}
			_ = LoginRequest(d)
		})
	}
}
