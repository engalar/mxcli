// SPDX-License-Identifier: Apache-2.0

// Stage 3.3.4 B2: gen-typed Mermaid renderer for the domainmodel domain.
// Mirrors cmd_mermaid.go::domainModelToMermaid. Reads via the gen cache
// helper; emits the same `erDiagram` block + @type/@colors metadata
// comments expected by the webview consumer.

package executor

import (
	"fmt"
	"strings"

	mdlerrors "github.com/mendixlabs/mxcli/mdl/errors"
	"github.com/mendixlabs/mxcli/model"
	genDm "github.com/mendixlabs/mxcli/modelsdk/gen/domainmodels"
)

// domainModelToMermaidGen mirrors domainModelToMermaid for the gen-typed
// read path. Renders an `erDiagram` block for one module's domain model
// with cross-module association support (parent / child entities resolved
// via buildAllEntityNamesGen).
func domainModelToMermaidGen(ctx *ExecContext, moduleName string) error {
	module, err := findModule(ctx, moduleName)
	if err != nil {
		return err
	}

	entities, err := listEntitiesForModuleGen(ctx, module.Name)
	if err != nil {
		return mdlerrors.NewBackend("list module entities", err)
	}
	assocs, err := listAssociationsForModuleGen(ctx, module.Name)
	if err != nil {
		return mdlerrors.NewBackend("list module associations", err)
	}

	// Local (this module) entity ID -> simple name lookup.
	entityNames := make(map[model.ID]string)
	for _, ent := range entities {
		entityNames[model.ID(ent.ID())] = ent.Name()
	}

	// Cross-module (whole project) entity ID -> qualified name lookup.
	allEntityNames, _ := buildAllEntityNamesGen(ctx)

	type entityInfo struct {
		label    string
		category string
	}
	entityInfos := make([]entityInfo, 0, len(entities))
	for _, ent := range entities {
		label := sanitizeMermaidID(ent.Name())
		entityInfos = append(entityInfos, entityInfo{label: label, category: classifyEntityGen(ent)})
	}

	var sb strings.Builder
	sb.WriteString("erDiagram\n")

	for i, ent := range entities {
		entityLabel := entityInfos[i].label
		sb.WriteString(fmt.Sprintf("    %s {\n", entityLabel))
		for _, a := range ent.AttributesItems() {
			attr, ok := a.(*genDm.Attribute)
			if !ok {
				continue
			}
			typeName := formatAttributeTypeGen(attr.Type())
			attrName := sanitizeMermaidID(attr.Name())
			sb.WriteString(fmt.Sprintf("        %s %s\n", typeName, attrName))
		}
		sb.WriteString("    }\n")
	}

	for _, assoc := range assocs {
		parentID := model.ID(assoc.ParentRefID())
		childID := model.ID(assoc.ChildRefID())
		parentName := entityNames[parentID]
		childName := entityNames[childID]
		if parentName == "" {
			if qn, ok := allEntityNames[parentID]; ok {
				parentName = sanitizeMermaidID(qn)
			} else {
				parentName = "Unknown"
			}
		} else {
			parentName = sanitizeMermaidID(parentName)
		}
		if childName == "" {
			if qn, ok := allEntityNames[childID]; ok {
				childName = sanitizeMermaidID(qn)
			} else {
				childName = "Unknown"
			}
		} else {
			childName = sanitizeMermaidID(childName)
		}
		rel := "}o--||"
		if assoc.Type() == "ReferenceSet" {
			rel = "}o--o{"
		}
		label := sanitizeMermaidLabel(assoc.Name())
		sb.WriteString(fmt.Sprintf("    %s %s %s : \"%s\"\n", parentName, rel, childName, label))
	}

	for _, ent := range entities {
		ref := entityGeneralizationQNGen(ent)
		if ref == "" {
			continue
		}
		childName := sanitizeMermaidID(ent.Name())
		parentName := sanitizeMermaidID(ref)
		sb.WriteString(fmt.Sprintf("    %s ||--|{ %s : \"generalizes\"\n", parentName, childName))
	}

	sb.WriteString("\n")
	sb.WriteString("%% @type erDiagram\n")
	sb.WriteString("%% @colors {")
	first := true
	for _, info := range entityInfos {
		if !first {
			sb.WriteString(",")
		}
		sb.WriteString(fmt.Sprintf(`"%s":"%s"`, info.label, info.category))
		first = false
	}
	sb.WriteString("}\n")

	fmt.Fprint(ctx.Output, sb.String())
	return nil
}
