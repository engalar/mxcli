package executor

import (
	"go/parser"
	"go/token"
	"os"
	"strings"
	"testing"
)

// TestNoDirectBSONImportInExecutor scans all non-test .go files in mdl/executor/
// and fails if any file directly imports bson/codec packages.
// All BSON access must go through ctx.Backend.* or gen-package supplement functions.
func TestNoDirectBSONImportInExecutor(t *testing.T) {
	forbidden := []string{
		"go.mongodb.org/mongo-driver/bson",
		"go.mongodb.org/mongo-driver/bson/primitive",
		"go.mongodb.org/mongo-driver/bson/bsoncore",
		"github.com/mendixlabs/mxcli/modelsdk/codec",
	}

	// allowlist contains files still being migrated.
	// Remove each entry here as the corresponding Task is completed.
	// Batch 3 (cmd_diff_local.go, flowbuilder_raw_setter_gen.go) are intentionally
	// deferred — see docs/superpowers/specs/2026-05-27-executor-bson-cleanup-design.md
	allowlist := map[string]bool{
		// Batch 2 – Task 10 (deferred to a separate PR — would touch
		// mdl/backend/mpr/datagrid_builder.go and re-trigger the
		// helpdesk-golden rebuild):
		"cmd_pages_builder_v3.go": true, // Task 10
		// Batch 3 – deferred (investigation pending):
		"cmd_diff_local.go":             true,
		"flowbuilder_raw_setter_gen.go": true,
	}

	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("read dir: %v", err)
	}

	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || strings.HasSuffix(name, "_test.go") || !strings.HasSuffix(name, ".go") {
			continue
		}
		if allowlist[name] {
			continue
		}
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, name, nil, parser.ImportsOnly)
		if err != nil {
			t.Fatalf("parse %s: %v", name, err)
		}
		for _, imp := range f.Imports {
			path := strings.Trim(imp.Path.Value, `"`)
			for _, bad := range forbidden {
				if path == bad || strings.HasPrefix(path, bad+"/") {
					t.Errorf("%s: forbidden import %q (use ctx.Backend.* or gen supplement functions instead)", name, path)
				}
			}
		}
	}
}
