// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"context"
	"fmt"
	"strings"

	"github.com/mendixlabs/mxcli/mdl/ast"
	mdlerrors "github.com/mendixlabs/mxcli/mdl/errors"
	"github.com/mendixlabs/mxcli/model"
	"github.com/mendixlabs/mxcli/sdk/domainmodel"
)

// describeMermaid generates a Mermaid diagram for the given object type and name.
// Supported types: entity (renders full domain model), microflow, nanoflow, page.
//
// Stage 3.2.6.3a: microflow / nanoflow paths route to the gen-typed
// builders (`microflowToMermaidGen` / `nanoflowToMermaidGen` in
// cmd_mermaid_gen.go); the legacy sdk/microflows-typed
// `microflowToMermaid`, `nanoflowToMermaid`, and `renderFlowMermaid`
// (along with the `mermaidActivityLabel` / `mermaidActionLabel` /
// `mermaidActivityDetails` / `mermaidActionDetails` /
// `mermaidMemberName` / `mermaidTextPreview` / `mermaidCaseLabel` /
// `mermaidResolveEntityName` helpers) were deleted. Equivalents using
// gen types live in cmd_microflows_viz_helpers_gen.go (the `*Gen`
// suffix variants).
func describeMermaid(ctx *ExecContext, objectType, name string) error {
	if !ctx.Connected() {
		return mdlerrors.NewNotConnected()
	}

	parts := strings.SplitN(name, ".", 2)
	var qn ast.QualifiedName
	if len(parts) == 2 {
		qn = ast.QualifiedName{Module: parts[0], Name: parts[1]}
	} else {
		qn = ast.QualifiedName{Module: name}
	}

	switch strings.ToLower(objectType) {
	case "entity", "domainmodel":
		// Stage 3.3.4 B3: prefer the gen-typed renderer when the
		// modelsdk-native DomainModels repo is wired (production
		// MprBackend path). Mock backends don't carry the repo and
		// fall back to the legacy renderer until they're updated.
		if ctx.DomainModels != nil {
			return domainModelToMermaidGen(ctx, qn.Module)
		}
		return domainModelToMermaid(ctx, qn.Module)
	case "microflow":
		return microflowToMermaidGen(ctx, qn)
	case "page":
		return pageToMermaid(ctx, qn)
	case "nanoflow":
		return nanoflowToMermaidGen(ctx, qn)
	default:
		return mdlerrors.NewUnsupported(fmt.Sprintf("mermaid format not supported for type: %s", objectType))
	}
}

// DescribeMermaid is a method wrapper for external callers.
func (e *Executor) DescribeMermaid(objectType, name string) error {
	return describeMermaid(e.newExecContext(context.Background()), objectType, name)
}

// domainModelToMermaid generates a Mermaid erDiagram for a module's domain model.
func domainModelToMermaid(ctx *ExecContext, moduleName string) error {
	module, err := findModule(ctx, moduleName)
	if err != nil {
		return err
	}

	dm, err := ctx.Backend.GetDomainModel(module.ID)
	if err != nil {
		return mdlerrors.NewBackend("get domain model", err)
	}

	// Build entity ID-to-name map for this module
	entityNames := make(map[model.ID]string)
	for _, entity := range dm.Entities {
		entityNames[entity.ID] = entity.Name
	}

	// Also load entities from all modules for cross-module associations
	allEntityNames := make(map[model.ID]string)
	h, err := getHierarchy(ctx)
	if err == nil {
		domainModels, _ := ctx.Backend.ListDomainModels()
		for _, otherDM := range domainModels {
			modName := h.GetModuleName(otherDM.ContainerID)
			for _, entity := range otherDM.Entities {
				allEntityNames[entity.ID] = modName + "." + entity.Name
			}
		}
	}

	// Classify entities by type for coloring
	type entityInfo struct {
		label    string
		category string // "persistent", "nonpersistent", "external", "view"
	}
	entityInfos := make([]entityInfo, 0, len(dm.Entities))
	for _, entity := range dm.Entities {
		label := sanitizeMermaidID(entity.Name)
		cat := "persistent"
		if strings.Contains(entity.Source, "OqlView") {
			cat = "view"
		} else if strings.Contains(entity.Source, "OData") || entity.RemoteSource != "" || entity.RemoteSourceDocument != "" {
			cat = "external"
		} else if !entity.Persistable {
			cat = "nonpersistent"
		}
		entityInfos = append(entityInfos, entityInfo{label: label, category: cat})
	}

	var sb strings.Builder
	sb.WriteString("erDiagram\n")

	// Emit entities with their attributes
	for i, entity := range dm.Entities {
		entityLabel := entityInfos[i].label
		sb.WriteString(fmt.Sprintf("    %s {\n", entityLabel))
		for _, attr := range entity.Attributes {
			typeName := attr.Type.GetTypeName()
			attrName := sanitizeMermaidID(attr.Name)
			sb.WriteString(fmt.Sprintf("        %s %s\n", typeName, attrName))
		}
		sb.WriteString("    }\n")
	}

	// Emit associations as relationships
	for _, assoc := range dm.Associations {
		parentName := entityNames[assoc.ParentID]
		childName := entityNames[assoc.ChildID]

		// For cross-module associations, use full qualified name
		if parentName == "" {
			if qn, ok := allEntityNames[assoc.ParentID]; ok {
				parentName = sanitizeMermaidID(qn)
			} else {
				parentName = "Unknown"
			}
		} else {
			parentName = sanitizeMermaidID(parentName)
		}
		if childName == "" {
			if qn, ok := allEntityNames[assoc.ChildID]; ok {
				childName = sanitizeMermaidID(qn)
			} else {
				childName = "Unknown"
			}
		} else {
			childName = sanitizeMermaidID(childName)
		}

		// Determine relationship cardinality
		// Parent (FROM) is the owner/many side, Child (TO) is the referenced/one side
		rel := "}o--||" // Reference: many-to-one (parent=*, child=1)
		if assoc.Type == domainmodel.AssociationTypeReferenceSet {
			rel = "}o--o{" // ReferenceSet: many-to-many
		}

		label := sanitizeMermaidLabel(assoc.Name)
		sb.WriteString(fmt.Sprintf("    %s %s %s : \"%s\"\n", parentName, rel, childName, label))
	}

	// Emit generalizations
	for _, entity := range dm.Entities {
		if entity.GeneralizationRef != "" {
			childName := sanitizeMermaidID(entity.Name)
			// GeneralizationRef is a qualified name like "System.User"
			parentName := sanitizeMermaidID(entity.GeneralizationRef)
			sb.WriteString(fmt.Sprintf("    %s ||--|{ %s : \"generalizes\"\n", parentName, childName))
		}
	}

	// Emit style classes for entity coloring
	sb.WriteString("\n")

	// Emit metadata for the webview
	sb.WriteString("%% @type erDiagram\n")

	// Emit a JSON metadata comment that the webview can parse for coloring
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

// buildEntityNames builds a map from entity ID to qualified name (Module.Entity)
// using the hierarchy for module name resolution. Used by both the entity
// erDiagram path above and the gen-typed flow visualisation builders
// (microflowToMermaidGen, nanoflowELKGen, etc.).
func buildEntityNames(ctx *ExecContext, h *ContainerHierarchy) (map[model.ID]string, error) {
	entityNames := make(map[model.ID]string)
	domainModels, err := ctx.Backend.ListDomainModels()
	if err != nil {
		return nil, mdlerrors.NewBackend("list domain models", err)
	}
	for _, dm := range domainModels {
		modName := h.GetModuleName(dm.ContainerID)
		for _, entity := range dm.Entities {
			entityNames[entity.ID] = modName + "." + entity.Name
		}
	}
	return entityNames, nil
}

// pageToMermaid generates a Mermaid block diagram for a page's widget structure.
func pageToMermaid(ctx *ExecContext, name ast.QualifiedName) error {
	h, err := getHierarchy(ctx)
	if err != nil {
		return mdlerrors.NewBackend("build hierarchy", err)
	}

	// Stage 3.3.5.B1: walk gen-typed Page listings via the
	// listPagesWithContainerGen cache helper.
	pairs, err := listPagesWithContainerGen(ctx)
	if err != nil {
		return mdlerrors.NewBackend("list pages", err)
	}

	var foundID model.ID
	for _, p := range pairs {
		if p.Elem == nil {
			continue
		}
		modID := h.FindModuleID(model.ID(p.ContainerID))
		modName := h.GetModuleName(modID)
		if modName == name.Module && p.Elem.Name() == name.Name {
			foundID = model.ID(p.Elem.ID())
			break
		}
	}

	if foundID == "" {
		return mdlerrors.NewNotFound("page", name.String())
	}

	// Use raw widget data (same approach as describePage)
	rawWidgets := getPageWidgetsFromRaw(ctx, foundID)

	var sb strings.Builder
	sb.WriteString("block-beta\n")
	sb.WriteString("    columns 1\n")

	// Title block
	title := name.Module + "." + name.Name
	sb.WriteString(fmt.Sprintf("    page_title[\"%s\"]\n", sanitizeMermaidLabel(title)))

	// Render widget tree
	renderRawWidgetMermaid(ctx, &sb, rawWidgets, 1, 0)

	// Emit metadata for the webview
	sb.WriteString("\n%% @type block\n")

	fmt.Fprint(ctx.Output, sb.String())
	return nil
}

// renderRawWidgetMermaid recursively renders raw widgets in Mermaid block-beta syntax.
func renderRawWidgetMermaid(ctx *ExecContext, sb *strings.Builder, widgets []rawWidget, depth int, counter int) int {
	for _, w := range widgets {
		counter++
		id := fmt.Sprintf("w%d", counter)
		label := w.Type
		if w.Name != "" {
			label = fmt.Sprintf("%s: %s", w.Type, w.Name)
		}
		indent := strings.Repeat("    ", depth)

		if len(w.Children) > 0 {
			fmt.Fprintf(sb, "%s%s[\"%s\"]:\n", indent, id, sanitizeMermaidLabel(label))
			counter = renderRawWidgetMermaid(ctx, sb, w.Children, depth+1, counter)
		} else {
			fmt.Fprintf(sb, "%s%s[\"%s\"]\n", indent, id, sanitizeMermaidLabel(label))
		}
	}
	return counter
}

// mermaidShortID generates a short, safe Mermaid node ID from a model.ID.
// Used by the gen-typed flow builders (cmd_mermaid_gen.go,
// cmd_nanoflow_elk_gen.go) — kept here because the node-ID convention
// is shared across all mermaid output, not per-flow.
func mermaidShortID(id model.ID) string {
	s := string(id)
	// Use last 8 chars of UUID to keep it short but unique
	if len(s) > 8 {
		s = s[len(s)-8:]
	}
	// Replace hyphens with underscores for Mermaid compatibility
	s = strings.ReplaceAll(s, "-", "_")
	return "n_" + s
}

// sanitizeMermaidID replaces characters that are not safe in Mermaid identifiers.
func sanitizeMermaidID(s string) string {
	s = strings.ReplaceAll(s, ".", "_")
	s = strings.ReplaceAll(s, " ", "_")
	s = strings.ReplaceAll(s, "-", "_")
	s = strings.ReplaceAll(s, "/", "_")
	return s
}

// sanitizeMermaidLabel escapes characters in a Mermaid label string.
func sanitizeMermaidLabel(s string) string {
	s = strings.ReplaceAll(s, "\"", "#quot;")
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "\r", "")
	return s
}

// mermaidTruncate truncates a string to max length with "..." suffix.
func mermaidTruncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	if max <= 3 {
		return s[:max]
	}
	return s[:max-3] + "..."
}
