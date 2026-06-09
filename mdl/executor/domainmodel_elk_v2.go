// SPDX-License-Identifier: Apache-2.0

// Stage 3.3.4 B1: gen-typed ELK rendering for the domainmodel domain.
// Mirrors cmd_domainmodel_elk.go's domainModelELK + helpers but reads
// from gen-typed *Entity / *Association instead of sdk types. The JSON
// schema (domainModelELKData / Entity / Assoc / Generalization) and the
// shared constants (elkCharWidth, elkHeaderHeight, etc.) live in
// cmd_domainmodel_elk.go and are used here unchanged so the wire shape
// stays byte-identical.
package executor

import (
	"fmt"
	"strings"

	"github.com/mendixlabs/mxcli/mdl/ast"
	mdlerrors "github.com/mendixlabs/mxcli/mdl/errors"
	"github.com/mendixlabs/mxcli/model"
	genDm "github.com/mendixlabs/mxcli/modelsdk/gen/domainmodels"
)

// domainModelELKGen mirrors domainModelELK on the gen-typed read path
// for module-level (non-focused) renders.
func domainModelELKGen(ctx *ExecContext, name string) error {
	if !ctx.Connected() {
		return mdlerrors.NewNotConnected()
	}

	if strings.Contains(name, ".") {
		return entityFocusELKGen(ctx, name)
	}

	moduleName := name
	module, err := findModule(ctx, moduleName)
	if err != nil {
		return err
	}

	allEntityNames, _ := buildAllEntityNamesGen(ctx)

	entities, err := listEntitiesForModuleGen(ctx, module.Name)
	if err != nil {
		return mdlerrors.NewBackend("list module entities", err)
	}
	moduleEntityIDs := make(map[model.ID]bool)
	for _, ent := range entities {
		moduleEntityIDs[model.ID(ent.ID())] = true
	}

	ghosts := make(map[string]*domainModelELKEntity)

	var elkEntities []domainModelELKEntity
	for _, ent := range entities {
		elkEntities = append(elkEntities, buildELKEntityGen(ent))
	}

	// Associations
	assocs, err := listAssociationsForModuleGen(ctx, module.Name)
	if err != nil {
		return mdlerrors.NewBackend("list module associations", err)
	}
	var elkAssocs []domainModelELKAssoc
	for i, assoc := range assocs {
		parentID := model.ID(assoc.ParentRefID())
		childID := model.ID(assoc.ChildRefID())
		addGhostIfNeeded(parentID, moduleEntityIDs, allEntityNames, ghosts)
		addGhostIfNeeded(childID, moduleEntityIDs, allEntityNames, ghosts)
		elkAssocs = append(elkAssocs, domainModelELKAssoc{
			ID:       fmt.Sprintf("assoc-%d", i),
			SourceID: "entity-" + string(childID),
			TargetID: "entity-" + string(parentID),
			Name:     assoc.Name(),
			Type:     assocTypeStrFromGen(assoc.Type()),
		})
	}

	// Generalizations
	var generalizations []domainModelELKGeneralization
	for _, ent := range entities {
		ref := entityGeneralizationQNGen(ent)
		if ref == "" {
			continue
		}
		gen, _ := buildGeneralizationGen(ent, ref, moduleEntityIDs, allEntityNames, ghosts)
		generalizations = append(generalizations, gen)
	}

	for _, ghost := range ghosts {
		if ghost.Width < elkMinWidth {
			ghost.Width = elkMinWidth
		}
		elkEntities = append(elkEntities, *ghost)
	}

	mdlSource, sourceMap := buildDomainModelMdlSourceGen(ctx, entities, moduleName)

	return emitDomainModelELK(ctx, domainModelELKData{
		Format:          "elk",
		Type:            "domainmodel",
		ModuleName:      moduleName,
		Entities:        elkEntities,
		Associations:    elkAssocs,
		Generalizations: generalizations,
		MdlSource:       mdlSource,
		SourceMap:       sourceMap,
	})
}

// entityFocusELKGen generates a focused ELK diagram showing only the selected
// entity and directly connected entities via associations or generalization.
func entityFocusELKGen(ctx *ExecContext, qualifiedName string) error {
	if !ctx.Connected() {
		return mdlerrors.NewNotConnected()
	}

	parts := strings.SplitN(qualifiedName, ".", 2)
	if len(parts) != 2 {
		return mdlerrors.NewValidationf("expected qualified name Module.Entity, got: %s", qualifiedName)
	}
	moduleName, entityName := parts[0], parts[1]

	focusEntity, _, err := findEntityGen(ctx, ast.QualifiedName{Module: moduleName, Name: entityName})
	if err != nil {
		return mdlerrors.NewBackend("get entity", err)
	}
	if focusEntity == nil {
		return mdlerrors.NewNotFound("entity", qualifiedName)
	}

	if classifyEntityGen(focusEntity) == "view" {
		if src, ok := focusEntity.Source().(*genDm.OqlViewEntitySource); ok && src.Oql() != "" {
			return OqlQueryPlanELK(ctx, qualifiedName, src.Oql())
		}
	}

	allEntityNames, _ := buildAllEntityNamesGen(ctx)
	entities, err := listEntitiesForModuleGen(ctx, moduleName)
	if err != nil {
		return mdlerrors.NewBackend("list module entities", err)
	}

	moduleEntitiesByID := make(map[model.ID]*genDm.Entity, len(entities))
	for _, entity := range entities {
		moduleEntitiesByID[model.ID(entity.ID())] = entity
	}

	includedIDs := make(map[model.ID]bool)
	focusID := model.ID(focusEntity.ID())
	includedIDs[focusID] = true

	assocs, err := listAssociationsForModuleGen(ctx, moduleName)
	if err != nil {
		return mdlerrors.NewBackend("list module associations", err)
	}

	var relevantAssocs []*genDm.Association
	for _, assoc := range assocs {
		parentID := model.ID(assoc.ParentRefID())
		childID := model.ID(assoc.ChildRefID())
		if parentID == focusID || childID == focusID {
			relevantAssocs = append(relevantAssocs, assoc)
			includedIDs[parentID] = true
			includedIDs[childID] = true
		}
	}

	allPairs, err := listDomainModelsWithContainerGen(ctx)
	if err == nil {
		moduleNames := map[model.ID]string{}
		if mods, modsErr := ctx.Backend.ListModules(); modsErr == nil {
			for _, m := range mods {
				moduleNames[m.ID] = m.Name
			}
		}
		for _, pair := range allPairs {
			if pair.DM == nil {
				continue
			}
			if moduleNames[pair.ContainerID] == moduleName {
				continue
			}
			for _, a := range pair.DM.AssociationsItems() {
				assoc, ok := a.(*genDm.Association)
				if !ok {
					continue
				}
				parentID := model.ID(assoc.ParentRefID())
				childID := model.ID(assoc.ChildRefID())
				if parentID == focusID || childID == focusID {
					relevantAssocs = append(relevantAssocs, assoc)
					includedIDs[parentID] = true
					includedIDs[childID] = true
				}
			}
		}
	}

	if ref := entityGeneralizationQNGen(focusEntity); ref != "" {
		for id, qn := range allEntityNames {
			if qn == ref {
				includedIDs[id] = true
				break
			}
		}
	}

	for _, entity := range entities {
		if entityGeneralizationQNGen(entity) == qualifiedName && model.ID(entity.ID()) != focusID {
			includedIDs[model.ID(entity.ID())] = true
		}
	}

	ghostEntities := make(map[string]*domainModelELKEntity)
	var elkEntities []domainModelELKEntity
	for id := range includedIDs {
		if ent, ok := moduleEntitiesByID[id]; ok {
			elkEnt := buildELKEntityGen(ent)
			if id == focusID {
				elkEnt.IsFocus = true
			}
			elkEntities = append(elkEntities, elkEnt)
			continue
		}
		ghostID := "entity-" + string(id)
		if _, exists := ghostEntities[ghostID]; exists {
			continue
		}
		name := "Unknown"
		if qn, ok := allEntityNames[id]; ok {
			name = qn
		}
		ghost := makeGhostEntity(ghostID, name)
		ghostEntities[ghostID] = &ghost
	}

	var elkAssocs []domainModelELKAssoc
	for i, assoc := range relevantAssocs {
		elkAssocs = append(elkAssocs, domainModelELKAssoc{
			ID:       fmt.Sprintf("assoc-%d", i),
			SourceID: "entity-" + string(assoc.ChildRefID()),
			TargetID: "entity-" + string(assoc.ParentRefID()),
			Name:     assoc.Name(),
			Type:     assocTypeStrFromGen(assoc.Type()),
		})
	}

	var generalizations []domainModelELKGeneralization
	if ref := entityGeneralizationQNGen(focusEntity); ref != "" {
		gen, _ := buildGeneralizationGen(focusEntity, ref, includedIDs, allEntityNames, ghostEntities)
		generalizations = append(generalizations, gen)
	}
	for _, entity := range entities {
		if entityGeneralizationQNGen(entity) == qualifiedName && model.ID(entity.ID()) != focusID {
			gen, _ := buildGeneralizationGen(entity, qualifiedName, includedIDs, allEntityNames, ghostEntities)
			generalizations = append(generalizations, gen)
		}
	}

	for _, ghost := range ghostEntities {
		if ghost.Width < elkMinWidth {
			ghost.Width = elkMinWidth
		}
		elkEntities = append(elkEntities, *ghost)
	}

	mdlSource, sourceMap := buildDomainModelMdlSourceGen(ctx, []*genDm.Entity{focusEntity}, moduleName)

	return emitDomainModelELK(ctx, domainModelELKData{
		Format:          "elk",
		Type:            "domainmodel",
		ModuleName:      moduleName,
		FocusEntity:     entityName,
		Entities:        elkEntities,
		Associations:    elkAssocs,
		Generalizations: generalizations,
		MdlSource:       mdlSource,
		SourceMap:       sourceMap,
	})
}

// classifyEntityGen mirrors classifyEntity for gen-typed entities.
func classifyEntityGen(entity *genDm.Entity) string {
	switch entityKindForGen(entity) {
	case "View":
		return "view"
	case "External":
		return "external"
	case "Non-Persistent":
		return "nonpersistent"
	}
	return "persistent"
}

// buildELKEntityGen mirrors buildELKEntity for gen-typed entities.
func buildELKEntityGen(entity *genDm.Entity) domainModelELKEntity {
	cat := classifyEntityGen(entity)
	var attrs []domainModelELKAttribute
	maxTextLen := float64(len(entity.Name()))

	for _, a := range entity.AttributesItems() {
		attr, ok := a.(*genDm.Attribute)
		if !ok {
			continue
		}
		typeName := formatAttributeTypeGen(attr.Type())
		attrs = append(attrs, domainModelELKAttribute{
			Name: attr.Name(),
			Type: typeName,
		})
		lineLen := float64(len(typeName) + 1 + len(attr.Name()))
		if lineLen > maxTextLen {
			maxTextLen = lineLen
		}
	}

	width := maxTextLen*elkCharWidth + elkHPadding
	if width < elkMinWidth {
		width = elkMinWidth
	}
	height := elkHeaderHeight + float64(len(attrs))*elkAttrLineHeight
	if len(attrs) == 0 {
		height = elkHeaderHeight + elkAttrLineHeight
	}
	return domainModelELKEntity{
		ID:         "entity-" + string(entity.ID()),
		Name:       entity.Name(),
		Category:   cat,
		Attributes: attrs,
		Width:      width,
		Height:     height,
	}
}

// buildAllEntityNamesGen mirrors buildAllEntityNames using the gen
// cache helper instead of ctx.Backend.ListDomainModels.
func buildAllEntityNamesGen(ctx *ExecContext) (map[model.ID]string, map[model.ID]string) {
	allEntityNames := make(map[model.ID]string)
	allEntityModules := make(map[model.ID]string)
	pairs, err := listDomainModelsWithContainerGen(ctx)
	if err != nil {
		return allEntityNames, allEntityModules
	}
	mods, err := ctx.Backend.ListModules()
	if err != nil {
		return allEntityNames, allEntityModules
	}
	moduleNames := make(map[model.ID]string, len(mods))
	for _, m := range mods {
		moduleNames[m.ID] = m.Name
	}
	for _, p := range pairs {
		if p.DM == nil {
			continue
		}
		modName := moduleNames[p.ContainerID]
		for _, e := range p.DM.EntitiesItems() {
			ent, ok := e.(*genDm.Entity)
			if !ok {
				continue
			}
			id := model.ID(ent.ID())
			allEntityNames[id] = modName + "." + ent.Name()
			allEntityModules[id] = modName
		}
	}
	return allEntityNames, allEntityModules
}

// buildGeneralizationGen mirrors buildGeneralization for gen-typed
// entities. Gen has no GeneralizationID — we look up the parent ID by
// resolving the qualified name.
func buildGeneralizationGen(entity *genDm.Entity, ref string, includedIDs map[model.ID]bool, allEntityNames map[model.ID]string, ghosts map[string]*domainModelELKEntity) (domainModelELKGeneralization, string) {
	// Resolve parent entity ID by reverse-looking-up the QN.
	var parentID string
	resolved := false
	for id, qn := range allEntityNames {
		if qn == ref {
			parentID = "entity-" + string(id)
			if !includedIDs[id] {
				if _, exists := ghosts[parentID]; !exists {
					ghost := makeGhostEntity(parentID, ref)
					ghosts[parentID] = &ghost
				}
			}
			resolved = true
			break
		}
	}
	if !resolved {
		// Synthetic ghost when the parent isn't in the project (e.g. System.User).
		syntheticID := "entity-gen-" + strings.ReplaceAll(ref, ".", "-")
		parentID = syntheticID
		if _, exists := ghosts[syntheticID]; !exists {
			ghost := makeGhostEntity(syntheticID, ref)
			ghosts[syntheticID] = &ghost
		}
	}
	return domainModelELKGeneralization{
		ChildID:    "entity-" + string(entity.ID()),
		ParentID:   parentID,
		ParentName: ref,
	}, parentID
}

// assocTypeStrFromGen mirrors assocTypeStr for the gen-typed Association.
// The gen API exposes Type() as a plain string ("Reference" or "ReferenceSet")
// rather than the sdk enum.
func assocTypeStrFromGen(t string) string {
	if t == "ReferenceSet" {
		return "referenceSet"
	}
	return "reference"
}

// buildDomainModelMdlSourceGen mirrors buildDomainModelMdlSource using
// describeEntityToString — which itself routes to describeEntity (legacy).
// When describeEntityGen lands in A6 dispatch, this naturally picks it up.
func buildDomainModelMdlSourceGen(ctx *ExecContext, entities []*genDm.Entity, moduleName string) (string, map[string]elkSourceRange) {
	sourceMap := make(map[string]elkSourceRange)
	var allSource strings.Builder
	lineCount := 0
	for i, ent := range entities {
		qn := ast.QualifiedName{Module: moduleName, Name: ent.Name()}
		entityMdl, err := describeEntityToString(ctx, qn)
		if err != nil {
			continue
		}
		startLine := lineCount
		if i > 0 {
			allSource.WriteString("\n")
			lineCount++
		}
		allSource.WriteString(entityMdl)
		entityLines := strings.Count(entityMdl, "\n")
		endLine := lineCount + entityLines
		sourceMap["entity-"+string(ent.ID())] = elkSourceRange{StartLine: startLine, EndLine: endLine}
		lineCount = endLine
	}
	return allSource.String(), sourceMap
}
