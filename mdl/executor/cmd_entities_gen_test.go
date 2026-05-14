// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"bytes"
	"strings"
	"testing"
)

// runListEntitiesGen builds a fixture-backed gen ctx, captures the
// table output, and returns the rendered string.
func runListEntitiesGen(t *testing.T, moduleName string) string {
	t.Helper()
	ctx := newDomainModelsTestContext(t)
	var buf bytes.Buffer
	ctx.Output = &buf
	ctx.Format = FormatTable
	if err := listEntitiesGen(ctx, moduleName); err != nil {
		t.Fatalf("listEntitiesGen(%q): %v", moduleName, err)
	}
	return buf.String()
}

func TestListEntitiesGen_RendersFixtureEntities(t *testing.T) {
	out := runListEntitiesGen(t, "")
	if out == "" {
		t.Fatal("listEntitiesGen produced no output")
	}
	if !strings.Contains(out, "entities)") {
		t.Errorf("output missing entity count summary; got:\n%s", out)
	}
	for _, want := range []string{"Entity", "Type", "Attrs", "Assocs"} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing column %q; got:\n%s", want, out)
		}
	}
}

func TestListEntitiesGen_FilterByModule(t *testing.T) {
	ctx := newDomainModelsTestContext(t)
	mods, err := ctx.Backend.ListModules()
	if err != nil || len(mods) == 0 {
		t.Skip("fixture has no modules")
	}
	// Pick the first non-System module that has entities.
	var modName string
	for _, m := range mods {
		if m.Name == "System" {
			continue
		}
		dms, err := ctx.DomainModels.List(m.ID)
		if err != nil || len(dms) == 0 {
			continue
		}
		if dms[0] != nil && len(dms[0].EntitiesItems()) > 0 {
			modName = m.Name
			break
		}
	}
	if modName == "" {
		t.Skip("fixture has no module with entities")
	}
	out := runListEntitiesGen(t, modName)
	if !strings.Contains(out, modName+".") {
		t.Errorf("filtered output should contain qualified name with module prefix %q; got:\n%s", modName, out)
	}
}

func TestEntityKindForGen_NilEntity(t *testing.T) {
	if got := entityKindForGen(nil); got != "" {
		t.Errorf("entityKindForGen(nil) = %q, want empty", got)
	}
}

func TestEntityGeneralizationQNGen_NilEntity(t *testing.T) {
	if got := entityGeneralizationQNGen(nil); got != "" {
		t.Errorf("entityGeneralizationQNGen(nil) = %q, want empty", got)
	}
}

// TestListEntitiesGen_DeterministicOutput verifies two consecutive runs
// produce byte-identical output (per CLAUDE.md "Map iteration is
// deterministic" rule).
func TestListEntitiesGen_DeterministicOutput(t *testing.T) {
	a := runListEntitiesGen(t, "")
	b := runListEntitiesGen(t, "")
	if a != b {
		t.Fatalf("listEntitiesGen output is not deterministic:\na=%s\nb=%s", a, b)
	}
}
