// SPDX-License-Identifier: Apache-2.0

// Package executor - Entity display and describe commands (SHOW/DESCRIBE ENTITY)
package executor

import (
	"context"
	"fmt"
	"strings"

	"github.com/mendixlabs/mxcli/mdl/ast"
	mdlerrors "github.com/mendixlabs/mxcli/mdl/errors"
	"github.com/mendixlabs/mxcli/model"
	genDm "github.com/mendixlabs/mxcli/modelsdk/gen/domainmodels"
)

// listEntities handles SHOW ENTITIES command.
func listEntities(ctx *ExecContext, moduleName string) error {
	return listEntitiesGen(ctx, moduleName)
}

// listEntity handles SHOW ENTITY command.
func listEntity(ctx *ExecContext, name *ast.QualifiedName) error {
	if name == nil {
		return mdlerrors.NewValidation("entity name required")
	}

	entity, modName, err := findEntityGen(ctx, *name)
	if err != nil {
		return mdlerrors.NewBackend("get entity", err)
	}
	if entity == nil {
		return mdlerrors.NewNotFound("entity", name.String())
	}

	fmt.Fprintf(ctx.Output, "**Entity: %s.%s**\n\n", modName, entity.Name())
	fmt.Fprintf(ctx.Output, "- Type: %s\n", entityKindForGen(entity))
	if extends := entityGeneralizationQNGen(entity); extends != "" {
		fmt.Fprintf(ctx.Output, "- Extends: %s\n", extends)
	}
	if loc := entity.Location(); loc != "" {
		fmt.Fprintf(ctx.Output, "- Location: %s\n", loc)
	}
	fmt.Fprintln(ctx.Output)

	if len(entity.AttributesItems()) > 0 {
		nameWidth, typeWidth := len("Attribute"), len("Type")
		type attrRow struct {
			name, typeName string
		}
		var rows []attrRow
		for _, a := range entity.AttributesItems() {
			attr, ok := a.(*genDm.Attribute)
			if !ok {
				continue
			}
			typeName := formatAttributeTypeGen(attr.Type())
			rows = append(rows, attrRow{attr.Name(), typeName})
			if len(attr.Name()) > nameWidth {
				nameWidth = len(attr.Name())
			}
			if len(typeName) > typeWidth {
				typeWidth = len(typeName)
			}
		}

		fmt.Fprintf(ctx.Output, "| %-*s | %-*s |\n", nameWidth, "Attribute", typeWidth, "Type")
		fmt.Fprintf(ctx.Output, "|-%s-|-%s-|\n", strings.Repeat("-", nameWidth), strings.Repeat("-", typeWidth))
		for _, r := range rows {
			fmt.Fprintf(ctx.Output, "| %-*s | %-*s |\n", nameWidth, r.name, typeWidth, r.typeName)
		}
		fmt.Fprintf(ctx.Output, "\n(%d attributes)\n", len(rows))
	}

	return nil
}

// describeEntity handles DESCRIBE ENTITY command.
func describeEntity(ctx *ExecContext, name ast.QualifiedName) error {
	return describeEntityGen(ctx, name)
}

// describeEntityToString generates MDL source for an entity and returns it as a string.
func describeEntityToString(ctx *ExecContext, name ast.QualifiedName) (string, error) {
	var buf strings.Builder
	origOutput := ctx.Output
	ctx.Output = &buf
	err := describeEntity(ctx, name)
	ctx.Output = origOutput
	if err != nil {
		return "", err
	}
	return buf.String(), nil
}

// extractAttrNameFromQualified extracts the attribute name from a 3-part
// qualified name ("Module.Entity.Attribute" → "Attribute").
// Returns "" for shorter names so callers can use the raw value as-is.
// For association names (2-part "Module.Assoc"), use memberAccessLocalName.
func extractAttrNameFromQualified(qualifiedName string) string {
	parts := strings.Split(qualifiedName, ".")
	if len(parts) >= 3 {
		return parts[len(parts)-1]
	}
	return ""
}

// resolveMicroflowByName resolves a qualified microflow name to its ID.
// It checks both microflows created during this session and existing microflows in the project.
func resolveMicroflowByName(ctx *ExecContext, qualifiedName string) (model.ID, error) {
	parts := strings.Split(qualifiedName, ".")
	if len(parts) < 2 {
		return "", mdlerrors.NewValidationf("invalid microflow name: %s (expected Module.Name)", qualifiedName)
	}
	moduleName := parts[0]
	mfName := strings.Join(parts[1:], ".")

	// Check microflows created during this session
	if ctx.Cache != nil && ctx.Cache.createdMicroflows != nil {
		if info, ok := ctx.Cache.createdMicroflows[qualifiedName]; ok {
			return info.ID, nil
		}
	}

	// Search existing microflows
	allMicroflows, err := listMicroflowsWithContainerGen(ctx)
	if err != nil {
		return "", mdlerrors.NewBackend("list microflows", err)
	}

	h, err := getHierarchy(ctx)
	if err != nil {
		return "", mdlerrors.NewBackend("build hierarchy", err)
	}

	for _, item := range allMicroflows {
		modID := h.FindModuleID(item.ContainerUUID)
		modName := h.GetModuleName(modID)
		if modName == moduleName && item.MF.Name() == mfName {
			return model.ID(item.MF.ID()), nil
		}
	}

	return "", mdlerrors.NewNotFound("microflow", qualifiedName)
}

// lookupMicroflowName reverse-looks up a microflow ID to its qualified name.
func lookupMicroflowName(ctx *ExecContext, mfID model.ID) string {
	allMicroflows, err := listMicroflowsWithContainerGen(ctx)
	if err != nil {
		return ""
	}

	h, err := getHierarchy(ctx)
	if err != nil {
		return ""
	}

	for _, item := range allMicroflows {
		if model.ID(item.MF.ID()) == mfID {
			modID := h.FindModuleID(item.ContainerUUID)
			modName := h.GetModuleName(modID)
			if modName != "" {
				return modName + "." + item.MF.Name()
			}
			return item.MF.Name()
		}
	}
	return ""
}

// --- Executor method wrappers for callers not yet migrated ---

func listEntityDeps(ctx context.Context, deps *HandlerDeps, name *ast.QualifiedName) error {
	return listEntityFuture(ctx, deps.Output, deps.ModuleLister, deps.DomainModels, name)
}

