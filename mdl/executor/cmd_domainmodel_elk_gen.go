// SPDX-License-Identifier: Apache-2.0

// Stage 3.3.4 B1: gen-typed ELK rendering for the domainmodel domain.
// Mirrors cmd_domainmodel_elk.go's domainModelELK + helpers but reads
// from gen-typed *Entity / *Association instead of sdk types. The JSON
// schema (domainModelELKData / Entity / Assoc / Generalization) and the
// shared constants (elkCharWidth, elkHeaderHeight, etc.) live in
// cmd_domainmodel_elk.go and are used here unchanged so the wire shape
// stays byte-identical.
//
// entityFocusELK (focused single-entity view) is intentionally not
// migrated in B1 — the dispatcher (B3) keeps focused renders on the
// legacy path until a follow-up commit.

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

	// Focused single-entity view stays on the legacy path until its
	// gen-typed sibling lands.
	if strings.Contains(name, ".") {
		return entityFocusELK(ctx, name)
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
