// SPDX-License-Identifier: Apache-2.0

package canonical_test

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestNoCanonicalLifecycleImportInExecutor verifies that the executor package
// does not import the canonical lifecycle sub-packages that were removed.
// This is a ratchet — if this test fails after a refactor, the canonical
// lifecycle layer is being re-introduced. Do not add to the allowlist.
func TestNoCanonicalLifecycleImportInExecutor(t *testing.T) {
	forbidden := []string{
		"github.com/mendixlabs/mxcli/mdl/canonical/entity",
		"github.com/mendixlabs/mxcli/mdl/canonical/association",
	}

	executorDir := filepath.Join("..", "executor")
	entries, err := os.ReadDir(executorDir)
	if err != nil {
		t.Fatalf("read executor dir: %v", err)
	}

	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || strings.HasSuffix(name, "_test.go") || !strings.HasSuffix(name, ".go") {
			continue
		}
		fullPath := filepath.Join(executorDir, name)
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, fullPath, nil, parser.ImportsOnly)
		if err != nil {
			t.Fatalf("parse %s: %v", name, err)
		}
		for _, imp := range f.Imports {
			path := strings.Trim(imp.Path.Value, `"`)
			for _, bad := range forbidden {
				if path == bad || strings.HasPrefix(path, bad+"/") {
					t.Errorf("%s: forbidden import %q — canonical lifecycle packages were deleted", name, path)
				}
			}
		}
	}
}
