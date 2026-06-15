// SPDX-License-Identifier: Apache-2.0

// Stage 3.3.4 A5: gen-typed entity walk for module describe.
// Replaces the *domainmodel.Entity-typed sortEntitiesByGeneralization
// + describe loop in cmd_modules.go::describeModule.

package executor

import (
	"strings"

	genDm "github.com/mendixlabs/mxcli/modelsdk/gen/domainmodels"
)

// sortEntitiesByGeneralizationGen mirrors sortEntitiesByGeneralization
// for gen-typed entities (Stage 3.3.4 A5). Topological sort: base
// entities come before derived entities.
func sortEntitiesByGeneralizationGen(entities []*genDm.Entity, moduleName string) []*genDm.Entity {
	if len(entities) <= 1 {
		return entities
	}

	entityByName := make(map[string]*genDm.Entity)
	for _, ent := range entities {
		entityByName[ent.Name()] = ent
		entityByName[moduleName+"."+ent.Name()] = ent
	}

	parentOf := make(map[string]string)
	for _, ent := range entities {
		if ref := entityGeneralizationQNGen(ent); ref != "" {
			parentOf[ent.Name()] = ref
		}
	}

	inDegree := make(map[string]int)
	for _, ent := range entities {
		inDegree[ent.Name()] = 0
	}
	for child, parent := range parentOf {
		if _, inModule := entityByName[parent]; inModule {
			inDegree[child]++
		}
	}

	var queue []string
	for _, ent := range entities {
		if inDegree[ent.Name()] == 0 {
			queue = append(queue, ent.Name())
		}
	}

	childrenOf := make(map[string][]string)
	for child, parent := range parentOf {
		if _, inModule := entityByName[parent]; inModule {
			parentName := parent
			if strings.Contains(parent, ".") {
				parts := strings.Split(parent, ".")
				parentName = parts[len(parts)-1]
			}
			childrenOf[parentName] = append(childrenOf[parentName], child)
		}
	}

	var sorted []*genDm.Entity
	visited := make(map[string]bool)
	for len(queue) > 0 {
		name := queue[0]
		queue = queue[1:]
		if visited[name] {
			continue
		}
		visited[name] = true
		if ent, ok := entityByName[name]; ok {
			sorted = append(sorted, ent)
		}
		for _, child := range childrenOf[name] {
			inDegree[child]--
			if inDegree[child] == 0 {
				queue = append(queue, child)
			}
		}
	}

	// Append any entities missed by the toposort (cycle / external parent).
	for _, ent := range entities {
		if !visited[ent.Name()] {
			sorted = append(sorted, ent)
		}
	}
	return sorted
}

// listEntitiesForModuleGen returns the gen-typed *Entity slice for a
// given module name. Mirrors the inner walk of legacy
// `dm := ctx.Backend.GetDomainModel(targetModule.ID); dm.Entities`
// but routed through the gen cache helper.
func listEntitiesForModuleGen(ctx *ExecContext, moduleName string) ([]*genDm.Entity, error) {
	pairs, err := listDomainModelsWithContainerGen(ctx)
	if err != nil {
		return nil, err
	}
	mods, err := ctx.ModuleLister.ListModules()
	if err != nil {
		return nil, err
	}
	var wantContainer string
	for _, m := range mods {
		if m.Name == moduleName {
			wantContainer = string(m.ID)
			break
		}
	}
	if wantContainer == "" {
		return nil, nil
	}
	for _, p := range pairs {
		if p.DM == nil {
			continue
		}
		if string(p.ContainerID) != wantContainer {
			continue
		}
		out := make([]*genDm.Entity, 0, len(p.DM.EntitiesItems()))
		for _, e := range p.DM.EntitiesItems() {
			ent, ok := e.(*genDm.Entity)
			if !ok {
				continue
			}
			out = append(out, ent)
		}
		return out, nil
	}
	return nil, nil
}

// listAssociationsForModuleGen returns the gen-typed *Association slice
// for a given module name. Mirrors `dm.Associations` walk.
func listAssociationsForModuleGen(ctx *ExecContext, moduleName string) ([]*genDm.Association, error) {
	pairs, err := listDomainModelsWithContainerGen(ctx)
	if err != nil {
		return nil, err
	}
	mods, err := ctx.ModuleLister.ListModules()
	if err != nil {
		return nil, err
	}
	var wantContainer string
	for _, m := range mods {
		if m.Name == moduleName {
			wantContainer = string(m.ID)
			break
		}
	}
	if wantContainer == "" {
		return nil, nil
	}
	for _, p := range pairs {
		if p.DM == nil {
			continue
		}
		if string(p.ContainerID) != wantContainer {
			continue
		}
		out := make([]*genDm.Association, 0, len(p.DM.AssociationsItems()))
		for _, a := range p.DM.AssociationsItems() {
			assoc, ok := a.(*genDm.Association)
			if !ok {
				continue
			}
			out = append(out, assoc)
		}
		return out, nil
	}
	return nil, nil
}
