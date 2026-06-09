package executor

import (
	"go/parser"
	"go/token"
	"go/ast"
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
	// Batch 3 (cmd_diff_local.go, flowbuilder_raw_setter_v2.go): investigation
	// pending — see docs/superpowers/specs/2026-05-27-executor-bson-cleanup-design.md
	allowlist := map[string]bool{
		"pages_builder_v3.go":          true, // Task 10 — see comment above
		"diff_local.go":                true, // Batch 3, investigation pending
		"flowbuilder_raw_setter_v2.go": true, // Batch 3, investigation pending
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

// TestNoRawBSONTypeStringsInExecutor scans executor files for string literals that
// contain Mendix model type names (e.g. "Forms$NanoflowSource") written inline.
//
// Raw type strings indicate that a BSON document is being hand-built instead of
// using gen types. This is fragile: if a type name changes or a new field becomes
// required, the string must be updated manually everywhere it appears.
//
// Correct pattern: use gen types (genPg.NewNanoflowSource()) + genElementToBSONDoc()
// so the TypeName is always derived from the generated code.
//
// Files in the allowlist are excluded because they are tracked under Task 10
// (backend abstraction) and their migration is in progress.
func TestNoRawBSONTypeStringsInExecutor(t *testing.T) {
	// Prefixes that identify Mendix model type name string literals.
	// These should only appear in gen-package init() calls, not in executor hand-built BSON.
	forbiddenPrefixes := []string{
		`"Forms$`,
		`"CustomWidgets$`,
		`"DomainModels$`,
		`"Microflows$`,
		`"Workflows$`,
	}

	// allowlist contains files that use type strings legitimately:
	//   a) BSON construction still being migrated to gen types (Task 10 / Batch 3)
	//   b) Type strings used for READING/IDENTIFYING existing BSON (switch/map key/TypeName check)
	//   c) SetTypeName("...") calls to override gen-type defaults — acceptable; type is still gen-managed
	//
	// New files must NOT be added here without explanation.
	// The purpose of this test is to prevent NEW violations, not to fix all existing ones at once.
	allowlist := map[string]bool{
		"pages_builder_v3.go":          true, // Task 10: buildDataGridDataSourceBSON still uses raw bson.D
		"diff_local.go":                true, // Batch 3, investigation pending
		"flowbuilder_raw_setter_v2.go": true, // Batch 3, investigation pending
		"theme_reader.go":              true, // (b) reads/identifies existing widget types; no construction
		"cmd_workflows_write_v2.go":    true, // (b) type switch/comparisons for reading, not construction
		"flowbuilder_calls_page_v2.go": true, // (c) SetTypeName override — gen element, not raw bson.D
		"workflows_v2.go":              true, // (b) type switch/comparisons when reading workflow activities
		"structure.go":                 true, // (b) type switch/comparisons for structure display
		// (b) type strings used for reading/identifying existing BSON in describe/show commands:
		"cmd_drop_entity_v2.go":          true,
		"entities_v2.go":                 true,
		"microflows_show_v2.go":          true,
		"microflows_show_list_v2.go":     true,
		"nanoflow_elk_v2.go":             true,
		"nanoflows_show_v2.go":           true,
		"page_wireframe.go":              true,
		"pages_describe.go":              true,
		"pages_describe_output.go":       true,
		"pages_describe_parse.go":        true,
		"pages_describe_pluggable.go":    true,
		"cmd_security_write_entity_v2.go": true,
		// (b) describe formatter-dispatch registration keys: type strings
		// identify which BSON $Type each formatter reads; no BSON construction.
		"widget_fmt_basic.go":     true,
		"widget_fmt_container.go": true,
		"widget_fmt_data.go":      true,
		"widget_fmt_datagrid.go":  true,
		"widget_fmt_datagrid2.go": true,
		"widget_fmt_layout.go":    true,
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
		f, err := parser.ParseFile(fset, name, nil, 0)
		if err != nil {
			continue // skip files with parse errors
		}
		ast.Inspect(f, func(n ast.Node) bool {
			lit, ok := n.(*ast.BasicLit)
			if !ok || lit.Kind.String() != "STRING" {
				return true
			}
			val := lit.Value // includes surrounding quotes
			for _, prefix := range forbiddenPrefixes {
				if strings.HasPrefix(val, prefix) {
					pos := fset.Position(lit.Pos())
					t.Errorf("%s:%d: raw BSON type string %s — use gen types instead of hand-building BSON documents",
						name, pos.Line, val)
				}
			}
			return true
		})
	}
}

