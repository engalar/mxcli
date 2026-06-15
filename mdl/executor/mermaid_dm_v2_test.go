// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"bytes"
	"strings"
	"testing"
)

func TestDomainModelToMermaidGen_RendersFixtureModule(t *testing.T) {
	ctx := newDomainModelsTestContext(t)
	mods, _ := ctx.ModuleLister.ListModules()
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

func TestDomainModelToMermaidGen_UsedByDescribeMermaid(t *testing.T) {
	ctx := newDomainModelsTestContext(t)
	mods, _ := ctx.ModuleLister.ListModules()
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
	var bufGen, bufDescribe bytes.Buffer
	ctx.Output = &bufGen
	if err := domainModelToMermaidGen(ctx, modName); err != nil {
		t.Fatalf("gen: %v", err)
	}
	ctx.Output = &bufDescribe
	if err := describeMermaid(ctx, "domainmodel", modName); err != nil {
		t.Fatalf("describeMermaid: %v", err)
	}
	genEntities := strings.Count(bufGen.String(), "    {")
	describeEntities := strings.Count(bufDescribe.String(), "    {")
	if genEntities != describeEntities {
		t.Errorf("entity-block count mismatch: gen=%d describe=%d", genEntities, describeEntities)
	}
}
