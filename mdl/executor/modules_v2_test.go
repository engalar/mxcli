// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"testing"

	genDm "github.com/mendixlabs/mxcli/modelsdk/gen/domainmodels"
)

func TestListEntitiesForModuleGen_FindsEntities(t *testing.T) {
	ctx := newDomainModelsTestContext(t)
	mods, err := ctx.ModuleLister.ListModules()
	if err != nil || len(mods) == 0 {
		t.Skip("fixture has no modules")
	}
	var found bool
	for _, m := range mods {
		ents, err := listEntitiesForModuleGen(ctx, m.Name)
		if err != nil {
			t.Errorf("listEntitiesForModuleGen(%q): %v", m.Name, err)
			continue
		}
		if len(ents) > 0 {
			found = true
			for _, e := range ents {
				if e == nil || e.Name() == "" {
					t.Errorf("module %s: entity has empty name", m.Name)
				}
			}
		}
	}
	if !found {
		t.Skip("fixture has no module with entities")
	}
}

func TestListEntitiesForModuleGen_UnknownModule(t *testing.T) {
	ctx := newDomainModelsTestContext(t)
	ents, err := listEntitiesForModuleGen(ctx, "DoesNotExistModule")
	if err != nil {
		t.Errorf("expected nil error for unknown module, got %v", err)
	}
	if ents != nil {
		t.Errorf("expected nil slice for unknown module, got %v", ents)
	}
}

func TestSortEntitiesByGeneralizationGen_EmptyAndSingleton(t *testing.T) {
	if got := sortEntitiesByGeneralizationGen(nil, "Mod"); got != nil {
		t.Errorf("nil input: got %v", got)
	}
	one := []*genDm.Entity{{}}
	if got := sortEntitiesByGeneralizationGen(one, "Mod"); len(got) != 1 || got[0] != one[0] {
		t.Errorf("singleton input: got %v", got)
	}
}

func TestListAssociationsForModuleGen_RunsAcrossFixture(t *testing.T) {
	ctx := newDomainModelsTestContext(t)
	mods, _ := ctx.ModuleLister.ListModules()
	for _, m := range mods {
		if _, err := listAssociationsForModuleGen(ctx, m.Name); err != nil {
			t.Errorf("listAssociationsForModuleGen(%q): %v", m.Name, err)
		}
	}
}
