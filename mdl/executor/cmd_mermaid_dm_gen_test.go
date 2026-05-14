// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"bytes"
	"strings"
	"testing"
)

func TestDomainModelToMermaidGen_RendersFixtureModule(t *testing.T) {
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
	var buf bytes.Buffer
	ctx.Output = &buf
	if err := domainModelToMermaidGen(ctx, modName); err != nil {
		t.Fatalf("domainModelToMermaidGen(%q): %v", modName, err)
	}
	out := buf.String()
	if !strings.HasPrefix(out, "erDiagram\n") {
		t.Errorf("output should start with erDiagram, got:\n%s", out[:min(300, len(out))])
	}
	if !strings.Contains(out, "%% @type erDiagram") {
		t.Errorf("output missing @type metadata; got:\n%s", out)
	}
	if !strings.Contains(out, "%% @colors {") {
		t.Errorf("output missing @colors metadata; got:\n%s", out)
	}
}

func TestDomainModelToMermaidGen_ConsistentWithLegacy(t *testing.T) {
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
	if err := domainModelToMermaidGen(ctx, modName); err != nil {
		t.Fatalf("gen: %v", err)
	}
	ctx.Output = &bufLegacy
	if err := domainModelToMermaid(ctx, modName); err != nil {
		t.Fatalf("legacy: %v", err)
	}
	// Compare entity count by counting "{" lines (one per entity).
	genEntities := strings.Count(bufGen.String(), "    {")
	legacyEntities := strings.Count(bufLegacy.String(), "    {")
	// Each entity opens with "    {" pattern via "    %s {\n"; both must agree.
	if genEntities != legacyEntities {
		t.Errorf("entity-block count mismatch: gen=%d legacy=%d", genEntities, legacyEntities)
	}
}
