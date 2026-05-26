package gen_test

import (
	"go/format"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestGeneratedCodeIsFormatted guards against gofmt drift in generated files.
// If this fails, re-run: go run ./cmd/modelsdk-codegen ...
func TestGeneratedCodeIsFormatted(t *testing.T) {
	root := "."
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(path, ".go") {
			return nil
		}
		src, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		formatted, err := format.Source(src)
		if err != nil {
			// not valid Go — skip (parse errors surfaced by go vet)
			return nil
		}
		if string(formatted) != string(src) {
			t.Errorf("generated file not gofmt-clean: %s\n  run: gofmt -w %s", path, path)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk error: %v", err)
	}
}
