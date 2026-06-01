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
		"go.mongodb.org/mongo-driver/v2/bson",
		"go.mongodb.org/mongo-driver/v2/bson/bsoncore",
		"github.com/mendixlabs/mxcli/modelsdk/codec",
	}

	// allowlist contains files still being migrated.
	// Remove each entry here as the corresponding Task is completed.
	//
	// Task 10 (cmd_pages_builder_v3.go): Move buildDataGridDataSourceBSON to
	// mdl/backend/mpr/datagrid_builder.go as a backend interface method so the
	// executor calls ctx.Backend.BuildDataGridDatasource() instead of constructing
	// raw BSON directly. The nanoflow case already uses gen types via genElementToBSONDoc;
	// remaining work: database/association/parameter/selection/microflow cases.
	// Once complete, remove cmd_pages_builder_v3.go from this allowlist and
	// remove the codec import from that file.
	//
	// Batch 3 (cmd_diff_local.go, flowbuilder_raw_setter_gen.go): investigation
	// pending — see docs/superpowers/specs/2026-05-27-executor-bson-cleanup-design.md
	allowlist := map[string]bool{
		"cmd_pages_builder_v3.go":      true, // Task 10 — see comment above
		"cmd_diff_local.go":             true, // Batch 3, investigation pending
		"flowbuilder_raw_setter_gen.go": true, // Batch 3, investigation pending
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
