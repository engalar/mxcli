// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"bytes"
	"encoding/json"
	"testing"
)

// TestDomainModelELKGen_RendersFixtureModule verifies the gen-typed ELK
// renderer produces a valid JSON document with entities/associations/
// generalizations sections for a fixture module (Stage 3.3.4 B1).
func TestDomainModelELKGen_RendersFixtureModule(t *testing.T) {
	ctx := newDomainModelsTestContext(t)
	mods, err := ctx.Backend.ListModules()
	if err != nil || len(mods) == 0 {
		t.Skip("fixture has no modules")
	}
	// Pick first non-System module that has entities.
	var modName string
	for _, m := range mods {
		if m.Name == "System" {
			continue
		}
		ents, err := listEntitiesForModuleGen(ctx, m.Name)
		if err == nil && len(ents) > 0 {
			modName = m.Name
			break
		}
	}
	if modName == "" {
		t.Skip("fixture has no module with entities")
	}
	var buf bytes.Buffer
	ctx.Output = &buf
	if err := domainModelELKGen(ctx, modName); err != nil {
		t.Fatalf("domainModelELKGen(%q): %v", modName, err)
	}
	out := buf.String()
	if out == "" {
		t.Fatal("renderer produced no output")
	}
	// Must be valid JSON with the expected schema.
	var data domainModelELKData
	if err := json.Unmarshal([]byte(out), &data); err != nil {
		t.Fatalf("output is not valid JSON: %v\nout:\n%s", err, out[:min(500, len(out))])
	}
	if data.Format != "elk" {
		t.Errorf("Format = %q, want elk", data.Format)
	}
	if data.Type != "domainmodel" {
		t.Errorf("Type = %q, want domainmodel", data.Type)
	}
	if data.ModuleName != modName {
		t.Errorf("ModuleName = %q, want %q", data.ModuleName, modName)
	}
	if len(data.Entities) == 0 {
		t.Error("Entities section is empty")
	}
}

// TestDomainModelELKGen_ConsistentWithLegacy compares the gen-typed
// output structure against the legacy domainModelELK on the same
// fixture module — the entity count and association count should match
// (per Q5 default: intermediate-representation snapshot, not byte-
// identical SVG).
func TestDomainModelELKGen_ConsistentWithLegacy(t *testing.T) {
	ctx := newDomainModelsTestContext(t)
	mods, _ := ctx.Backend.ListModules()
	var modName string
	for _, m := range mods {
		if m.Name == "System" {
			continue
		}
		ents, _ := listEntitiesForModuleGen(ctx, m.Name)
		if len(ents) > 0 {
			modName = m.Name
			break
		}
	}
	if modName == "" {
		t.Skip("fixture has no module with entities")
	}
	var bufGen, bufLegacy bytes.Buffer
	ctx.Output = &bufGen
	if err := domainModelELKGen(ctx, modName); err != nil {
		t.Fatalf("gen: %v", err)
	}
	ctx.Output = &bufLegacy
	if err := domainModelELK(ctx, modName); err != nil {
		t.Fatalf("legacy: %v", err)
	}
	var dataGen, dataLegacy domainModelELKData
	if err := json.Unmarshal(bufGen.Bytes(), &dataGen); err != nil {
		t.Fatalf("gen JSON: %v", err)
	}
	if err := json.Unmarshal(bufLegacy.Bytes(), &dataLegacy); err != nil {
		t.Fatalf("legacy JSON: %v", err)
	}
	if len(dataGen.Entities) != len(dataLegacy.Entities) {
		t.Errorf("Entities count mismatch: gen=%d legacy=%d", len(dataGen.Entities), len(dataLegacy.Entities))
	}
	if len(dataGen.Associations) != len(dataLegacy.Associations) {
		t.Errorf("Associations count mismatch: gen=%d legacy=%d", len(dataGen.Associations), len(dataLegacy.Associations))
	}
	if len(dataGen.Generalizations) != len(dataLegacy.Generalizations) {
		t.Errorf("Generalizations count mismatch: gen=%d legacy=%d", len(dataGen.Generalizations), len(dataLegacy.Generalizations))
	}
}

func TestClassifyEntityGen_Defaults(t *testing.T) {
	if got := classifyEntityGen(nil); got != "persistent" {
		t.Errorf("nil entity → %q, want persistent", got)
	}
}

func TestAssocTypeStrFromGen(t *testing.T) {
	if got := assocTypeStrFromGen("ReferenceSet"); got != "referenceSet" {
		t.Errorf("ReferenceSet → %q", got)
	}
	if got := assocTypeStrFromGen("Reference"); got != "reference" {
		t.Errorf("Reference → %q", got)
	}
	if got := assocTypeStrFromGen(""); got != "reference" {
		t.Errorf("empty → %q (want default reference)", got)
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
