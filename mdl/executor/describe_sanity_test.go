// SPDX-License-Identifier: Apache-2.0

// describe_sanity_test.go — L6b: Describe Sanity layer.
//
// Opens each testdata MPR, iterates all documents of a given kind,
// calls the corresponding DESCRIBE function, and feeds the resulting
// MDL back through visitor.Build() to check it parses.
//
// L6b is a coverage smoke test, not a regression gate. The tests
// always run to completion against every microflow / entity in every
// testdata MPR and report aggregate pass/fail counts at the end.
// Per-document failures are recorded with t.Logf and surfaced as a
// single test-level failure only when nothing passes (which would
// indicate a structural regression rather than an MDL-rendering bug).
//
// This is intentional: the renderer has known gaps for niche activity
// types, and we don't want a single new pattern in a third-party module
// to break CI. The aggregated counts make regressions visible in
// reports without blocking unrelated work.
//
// This test does NOT require Docker or network access — it runs purely
// against the committed testdata MPR files.
package executor

import (
	"bytes"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/mendixlabs/mxcli/mdl/ast"
	mprbackend "github.com/mendixlabs/mxcli/mdl/backend/mpr"
	"github.com/mendixlabs/mxcli/mdl/canonical"
	entitymodel "github.com/mendixlabs/mxcli/mdl/canonical/entity"
	"github.com/mendixlabs/mxcli/mdl/visitor"
	genDm "github.com/mendixlabs/mxcli/modelsdk/gen/domainmodels"
	mmpr "github.com/mendixlabs/mxcli/modelsdk/mpr"
)

// testdataMPRs lists all testdata MPR files to exercise in L6b.
var testdataMPRs = []string{
	"../../testdata/expr-checker/minimal.mpr",
	"../../testdata/corpus-b/app.mpr",
}

// TestDescribeSanity_Microflows verifies DescribeMicroflowGenToString
// produces parseable MDL for every microflow in testdata. See file
// header for L6b reporter-mode rationale.
func TestDescribeSanity_Microflows(t *testing.T) {
	t.Parallel()
	var total, passed int
	for _, mprPath := range testdataMPRs {
		mprPath := mprPath
		t.Run(mprPath, func(t *testing.T) {
			w := openMprWriterForFixedPath(t, mprPath)
			ctx := newSanityContext(t, w)

			repo := ctx.Microflows
			if repo == nil {
				t.Skip("no microflow repo available")
			}
			mfs, err := repo.ListAll()
			if err != nil {
				t.Fatalf("ListAll microflows: %v", err)
			}
			t.Logf("microflows found: %d", len(mfs))

			localTotal, localPass := 0, 0
			for _, mf := range mfs {
				_, qn := genMicroflowQualifiedName(ctx, mf)
				out, err := DescribeMicroflowGenToString(ctx, mf)
				if err != nil {
					t.Logf("DescribeMicroflowGenToString(%q) error: %v", qn, err)
					localTotal++
					continue
				}
				mdl := extractMDLFromDescribeOutput(out)
				localTotal++
				if mdl == "" {
					localPass++
					continue
				}
				if _, errs := visitor.Build(mdl); len(errs) > 0 {
					t.Logf("DESCRIBE %q produced invalid MDL (%d parse errors): first=%v", qn, len(errs), errs[0])
					continue
				}
				localPass++
			}
			t.Logf("microflows passed: %d / %d", localPass, localTotal)
			total += localTotal
			passed += localPass
		})
	}
	t.Logf("TOTAL microflows: %d / %d parseable", passed, total)
	if total == 0 {
		t.Fatal("no microflows exercised — check testdata paths or ListAll wiring")
	}
	if passed == 0 {
		t.Fatal("every microflow rendered invalid MDL — describer is structurally broken")
	}
}

// TestDescribeSanity_Entities verifies describeEntityGen produces
// parseable MDL for every entity in testdata. See file header for L6b
// reporter-mode rationale.
func TestDescribeSanity_Entities(t *testing.T) {
	t.Parallel()
	var total, passed int
	for _, mprPath := range testdataMPRs {
		mprPath := mprPath
		t.Run(mprPath, func(t *testing.T) {
			w := openMprWriterForFixedPath(t, mprPath)
			ctx := newSanityContext(t, w)

			be := ctx.Backend
			if be == nil {
				t.Skip("no backend available")
			}
			mods, err := be.ListModules()
			if err != nil {
				t.Fatalf("ListModules: %v", err)
			}

			localTotal, localPass := 0, 0
			for _, mod := range mods {
				if mod == nil {
					continue
				}
				// Skip system / platform modules whose entities aren't
				// part of user-authored MDL surface.
				if mod.Name == "System" || mod.Name == "Administration" {
					continue
				}
				dm, err := be.GetDomainModelGen(mod.ID)
				if err != nil || dm == nil {
					continue
				}
				for _, e := range dm.EntitiesItems() {
					ent, ok := e.(*genDm.Entity)
					if !ok || ent == nil {
						continue
					}
					qn := fmt.Sprintf("%s.%s", mod.Name, ent.Name())
					var sb bytes.Buffer
					ctx.Output = &sb
					name := ast.QualifiedName{Module: mod.Name, Name: ent.Name()}
					localTotal++
					if err := describeEntityGen(ctx, name); err != nil {
						t.Logf("describeEntityGen(%q) error: %v", qn, err)
						continue
					}
					mdl := extractMDLFromDescribeOutput(sb.String())
					if mdl == "" {
						localPass++
						continue
					}
					if _, errs := visitor.Build(mdl); len(errs) > 0 {
						t.Logf("DESCRIBE %q produced invalid MDL (%d parse errors): first=%v", qn, len(errs), errs[0])
						continue
					}
					localPass++
				}
			}
			t.Logf("entities passed: %d / %d", localPass, localTotal)
			total += localTotal
			passed += localPass
		})
	}
	t.Logf("TOTAL entities: %d / %d parseable", passed, total)
	if total == 0 {
		t.Fatal("no entities exercised — check testdata paths or ListModules wiring")
	}
	if passed == 0 {
		t.Fatal("every entity rendered invalid MDL — describer is structurally broken")
	}
}

// openMprWriterForFixedPath opens an mmpr.Writer for the given fixture
// path (copies to a tempdir first to avoid mutating testdata).
func openMprWriterForFixedPath(t *testing.T, path string) *mmpr.Writer {
	t.Helper()
	dst := copyMPRFixture(t, path, t.TempDir())
	w, err := mmpr.NewWriter(dst)
	if err != nil {
		t.Fatalf("mmpr.NewWriter(%s): %v", path, err)
	}
	t.Cleanup(func() { _ = w.Close() })
	return w
}

// newSanityContext builds an ExecContext with both the Microflows AND
// DomainModels repos wired up. newGenDescribeContext alone is enough
// for the microflow describer but not for describeEntityGen, which
// needs DomainModels to resolve entities. This helper exists so that
// L6b can exercise the full describe surface without disturbing the
// microflow-focused helper used elsewhere.
func newSanityContext(t *testing.T, w *mmpr.Writer) *ExecContext {
	t.Helper()
	repoCtx := mprbackend.NewExecutorContext(w)
	path := w.ConcreteReader().Path()
	be, err := mprbackend.NewFromPath(path)
	if err != nil {
		t.Fatalf("mprbackend.NewFromPath(%s): %v", path, err)
	}
	t.Cleanup(func() { _ = be.Disconnect() })
	mc := canonical.NewDefaultRegistry()
	entitymodel.RegisterCodec(mc)
	ctx := &ExecContext{
		Backend:      be,
		Microflows:   repoCtx.Microflows,
		DomainModels: repoCtx.DomainModels,
		Output:       io.Discard,
		ModelCodecs:  mc,
	}
	// Cache the listDomainModelsWithContainerGen result so the entity
	// loop runs in O(N) instead of O(N * modules).
	ctx.ensureCache()
	return ctx
}

// extractMDLFromDescribeOutput strips the JSON wrapper that
// writeDescribeJSON adds in FormatJSON mode and returns the inner MDL
// content. In FormatText mode (the default in tests) the input is
// already raw MDL, so the function returns it trimmed.
func extractMDLFromDescribeOutput(s string) string {
	const marker = `"mdl":`
	idx := strings.Index(s, marker)
	if idx < 0 {
		return strings.TrimSpace(s)
	}
	rest := strings.TrimSpace(s[idx+len(marker):])
	if !strings.HasPrefix(rest, `"`) {
		return strings.TrimSpace(s)
	}
	rest = rest[1:]
	end := strings.LastIndex(rest, `"`)
	if end < 0 {
		return strings.TrimSpace(rest)
	}
	raw := rest[:end]
	raw = strings.ReplaceAll(raw, `\n`, "\n")
	raw = strings.ReplaceAll(raw, `\"`, `"`)
	raw = strings.ReplaceAll(raw, `\\`, `\`)
	return strings.TrimSpace(raw)
}
