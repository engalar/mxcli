// SPDX-License-Identifier: Apache-2.0

// Package executor — RENAME commands (entity, module)
package executor

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/mendixlabs/mxcli/mdl/ast"
	mdlerrors "github.com/mendixlabs/mxcli/mdl/errors"
	"github.com/mendixlabs/mxcli/mdl/types"
	"github.com/mendixlabs/mxcli/model"
	genDm "github.com/mendixlabs/mxcli/modelsdk/gen/domainmodels"
	genMf "github.com/mendixlabs/mxcli/modelsdk/gen/microflows"
)

func printRenameReport(ctx *ExecContext, oldName, newName string, hits []types.RenameHit) {
	printRenameReportFn(ctx.Output, oldName, newName, hits)
}

func totalRefCount(hits []types.RenameHit) int {
	total := 0
	for _, h := range hits {
		total += h.Count
	}
	return total
}

func mergeHits(a, b []types.RenameHit) []types.RenameHit {
	seen := make(map[string]int) // unitID → index in result
	result := make([]types.RenameHit, len(a))
	copy(result, a)
	for i := range result {
		seen[result[i].UnitID] = i
	}
	for _, h := range b {
		if idx, ok := seen[h.UnitID]; ok {
			result[idx].Count += h.Count
		} else {
			seen[h.UnitID] = len(result)
			result = append(result, h)
		}
	}
	return result
}

// ────────────────────────────────────────────────────────────
// Phase 3d-5h: Fn (HandlerDeps) versions of rename functions
// ────────────────────────────────────────────────────────────

func execRenameFn(ctx context.Context, s *ast.RenameStmt, deps *HandlerDeps) error {
	if !deps.ConnectionManager.IsConnected() {
		return mdlerrors.NewNotConnected()
	}

	switch s.ObjectType {
	case "entity":
		return execRenameEntityFn(ctx, s, deps)
	case "microflow":
		return execRenameDocumentFn(ctx, s, deps, "microflow")
	case "nanoflow":
		return execRenameDocumentFn(ctx, s, deps, "nanoflow")
	case "page":
		return execRenameDocumentFn(ctx, s, deps, "page")
	case "enumeration":
		return execRenameEnumerationFn(ctx, s, deps)
	case "association":
		return execRenameAssociationFn(ctx, s, deps)
	case "constant":
		return execRenameDocumentFn(ctx, s, deps, "constant")
	case "workflow":
		return execRenameDocumentFn(ctx, s, deps, "workflow")
	case "javaaction":
		return execRenameJavaActionFn(ctx, s, deps)
	case "module":
		return execRenameModuleFn(ctx, s, deps)
	default:
		return mdlerrors.NewUnsupported(fmt.Sprintf("rename not supported for %s", s.ObjectType))
	}
}

func execRenameEntityFn(ctx context.Context, s *ast.RenameStmt, deps *HandlerDeps) error {
	module, err := findModuleDeps(ctx, deps, s.Name.Module)
	if err != nil {
		return err
	}

	dm, err := getDomainModelGenCachedDeps(ctx, deps, module.ID)
	if err != nil {
		return mdlerrors.NewBackend("get domain model", err)
	}
	if dm == nil {
		return mdlerrors.NewNotFound("entity", s.Name.String())
	}

	found := false
	collision := false
	for _, entityElem := range dm.EntitiesItems() {
		ent, ok := entityElem.(*genDm.Entity)
		if !ok {
			continue
		}
		if ent.Name() == s.Name.Name {
			found = true
		} else if ent.Name() == s.NewName {
			collision = true
		}
	}
	if !found {
		return mdlerrors.NewNotFound("entity", s.Name.String())
	}
	if collision {
		return mdlerrors.NewValidationf("entity %s already exists in module %s", s.NewName, s.Name.Module)
	}

	oldQualifiedName := s.Name.Module + "." + s.Name.Name
	newQualifiedName := s.Name.Module + "." + s.NewName

	hits, err := deps.RenameManager.RenameReferences(oldQualifiedName, newQualifiedName, s.DryRun)
	if err != nil {
		return mdlerrors.NewBackend("scan references", err)
	}

	if s.DryRun {
		printRenameReportFn(deps.Output, oldQualifiedName, newQualifiedName, hits)
		return nil
	}

	for _, entityElem := range dm.EntitiesItems() {
		ent, ok := entityElem.(*genDm.Entity)
		if !ok {
			continue
		}
		if ent.Name() == s.Name.Name {
			ent.SetName(s.NewName)
			break
		}
	}
	if err := deps.DomainModelWriter.UpdateDomainModelGen(dm); err != nil {
		return mdlerrors.NewBackend("update entity name", err)
	}
	setDomainModelGenCachedDeps(deps, module.ID, dm)

	invalidateHierarchyDeps(deps)
	invalidateDomainModelsCacheDeps(deps)

	fmt.Fprintf(deps.Output, "Renamed entity: %s → %s\n", oldQualifiedName, newQualifiedName)
	if len(hits) > 0 {
		fmt.Fprintf(deps.Output, "Updated %d reference(s) in %d document(s)\n", totalRefCount(hits), len(hits))
	}
	return nil
}

func execRenameModuleFn(ctx context.Context, s *ast.RenameStmt, deps *HandlerDeps) error {
	oldModuleName := s.Name.Module
	newModuleName := s.NewName

	module, err := findModuleDeps(ctx, deps, oldModuleName)
	if err != nil {
		return err
	}

	hits, err := deps.RenameManager.RenameReferences(oldModuleName+".", newModuleName+".", s.DryRun)
	if err != nil {
		return mdlerrors.NewBackend("scan references", err)
	}

	exactHits, err := deps.RenameManager.RenameReferences(oldModuleName, newModuleName, s.DryRun)
	if err != nil {
		return mdlerrors.NewBackend("scan exact module references", err)
	}

	allHits := mergeHits(hits, exactHits)

	if s.DryRun {
		printRenameReportFn(deps.Output, oldModuleName, newModuleName, allHits)
		return nil
	}

	module.Name = newModuleName
	if err := deps.ModuleWriter.UpdateModule(module); err != nil {
		return mdlerrors.NewBackend("update module name", err)
	}

	invalidateHierarchyDeps(deps)
	invalidateDomainModelsCacheDeps(deps)

	fmt.Fprintf(deps.Output, "Renamed module: %s → %s\n", oldModuleName, newModuleName)
	if len(allHits) > 0 {
		fmt.Fprintf(deps.Output, "Updated %d reference(s) in %d document(s)\n", totalRefCount(allHits), len(allHits))
	}
	return nil
}

func execRenameDocumentFn(ctx context.Context, s *ast.RenameStmt, deps *HandlerDeps, docType string) error {
	oldQualifiedName := s.Name.Module + "." + s.Name.Name
	newQualifiedName := s.Name.Module + "." + s.NewName

	h, err := GetOrBuildHierarchy(deps)
	if err != nil {
		return err
	}

	found := false
	collision := false
	switch docType {
	case "microflow":
		var mfs []*genMf.Microflow
		if deps.MicroflowRepo != nil {
			mfs, _ = deps.MicroflowRepo.ListAll()
		}
		for _, mf := range mfs {
			if mf == nil {
				continue
			}
			modName := genFlowContainerModuleDeps(deps, h, model.ID(mf.ID()))
			if modName != s.Name.Module {
				continue
			}
			if mf.Name() == s.Name.Name {
				found = true
			} else if mf.Name() == s.NewName {
				collision = true
			}
		}
	case "nanoflow":
		var nfs []*genMf.Nanoflow
		if deps.NanoflowRepo != nil {
			nfs, _ = deps.NanoflowRepo.List("")
		}
		for _, nf := range nfs {
			if nf == nil {
				continue
			}
			modName := genFlowContainerModuleDeps(deps, h, model.ID(nf.ID()))
			if modName != s.Name.Module {
				continue
			}
			if nf.Name() == s.Name.Name {
				found = true
			} else if nf.Name() == s.NewName {
				collision = true
			}
		}
	case "page":
		pairs, _ := listPagesWithContainerGenDeps(ctx, deps)
		for _, p := range pairs {
			if p.Elem == nil {
				continue
			}
			modID := h.FindModuleID(model.ID(p.ContainerID))
			if h.GetModuleName(modID) != s.Name.Module {
				continue
			}
			if p.Elem.Name() == s.Name.Name {
				found = true
			} else if p.Elem.Name() == s.NewName {
				collision = true
			}
		}
	case "constant":
		cs, _ := deps.ConstantReader.ListConstants()
		for _, c := range cs {
			modID := h.FindModuleID(c.ContainerID)
			if h.GetModuleName(modID) != s.Name.Module {
				continue
			}
			if c.Name == s.Name.Name {
				found = true
			} else if c.Name == s.NewName {
				collision = true
			}
		}
	case "workflow":
		pairs, _ := listWorkflowsWithContainerGenDeps(deps)
		for _, p := range pairs {
			if p.Elem == nil {
				continue
			}
			modID := h.FindModuleID(model.ID(p.ContainerID))
			if h.GetModuleName(modID) != s.Name.Module {
				continue
			}
			if p.Elem.Name() == s.Name.Name {
				found = true
			} else if p.Elem.Name() == s.NewName {
				collision = true
			}
		}
	}

	if !found {
		return mdlerrors.NewNotFound(s.ObjectType, oldQualifiedName)
	}
	if collision {
		return mdlerrors.NewValidationf("%s %s already exists in module %s", docType, s.NewName, s.Name.Module)
	}

	hits, err := deps.RenameManager.RenameReferences(oldQualifiedName, newQualifiedName, s.DryRun)
	if err != nil {
		return mdlerrors.NewBackend("scan references", err)
	}

	if s.DryRun {
		printRenameReportFn(deps.Output, oldQualifiedName, newQualifiedName, hits)
		return nil
	}

	if err := deps.RenameManager.RenameDocumentByName(s.Name.Module, s.Name.Name, s.NewName); err != nil {
		return mdlerrors.NewBackend(fmt.Sprintf("rename %s", docType), err)
	}

	invalidateHierarchyDeps(deps)
	deps.Cache.microflowNames = nil
	deps.Cache.Invalidate(
		CacheDomainMicroflows, CacheDomainNanoflows,
		CacheDomainPages, CacheDomainLayouts, CacheDomainSnippets,
		CacheDomainWorkflows,
		CacheDomainJavaActions, CacheDomainJavaScriptActions,
	)

	fmt.Fprintf(deps.Output, "Renamed %s: %s → %s\n", docType, oldQualifiedName, newQualifiedName)
	if len(hits) > 0 {
		fmt.Fprintf(deps.Output, "Updated %d reference(s) in %d document(s)\n", totalRefCount(hits), len(hits))
	}
	return nil
}

func execRenameEnumerationFn(ctx context.Context, s *ast.RenameStmt, deps *HandlerDeps) error {
	oldQualifiedName := s.Name.Module + "." + s.Name.Name
	newQualifiedName := s.Name.Module + "." + s.NewName

	enums, err := deps.EnumerationReader.ListEnumerations()
	if err != nil {
		return mdlerrors.NewBackend("list enumerations", err)
	}
	h, err := GetOrBuildHierarchy(deps)
	if err != nil {
		return err
	}
	found := false
	collision := false
	for _, en := range enums {
		modID := h.FindModuleID(en.ContainerID)
		if h.GetModuleName(modID) != s.Name.Module {
			continue
		}
		if en.Name == s.Name.Name {
			found = true
		} else if en.Name == s.NewName {
			collision = true
		}
	}
	if !found {
		return mdlerrors.NewNotFound("enumeration", oldQualifiedName)
	}
	if collision {
		return mdlerrors.NewValidationf("enumeration %s already exists in module %s", s.NewName, s.Name.Module)
	}

	hits, err := deps.RenameManager.RenameReferences(oldQualifiedName, newQualifiedName, s.DryRun)
	if err != nil {
		return mdlerrors.NewBackend("scan references", err)
	}

	if s.DryRun {
		printRenameReportFn(deps.Output, oldQualifiedName, newQualifiedName, hits)
		return nil
	}

	if err := deps.RenameManager.RenameDocumentByName(s.Name.Module, s.Name.Name, s.NewName); err != nil {
		return mdlerrors.NewBackend("rename enumeration", err)
	}

	if err := deps.DomainModelWriter.UpdateEnumerationRefsInAllDomainModels(oldQualifiedName, newQualifiedName); err != nil {
		fmt.Fprintf(deps.Output, "Warning: failed to update enumeration references in domain models: %v\n", err)
	}

	invalidateHierarchyDeps(deps)
	invalidateDomainModelsCacheDeps(deps)

	fmt.Fprintf(deps.Output, "Renamed enumeration: %s → %s\n", oldQualifiedName, newQualifiedName)
	if len(hits) > 0 {
		fmt.Fprintf(deps.Output, "Updated %d reference(s) in %d document(s)\n", totalRefCount(hits), len(hits))
	}
	return nil
}

func execRenameAssociationFn(ctx context.Context, s *ast.RenameStmt, deps *HandlerDeps) error {
	oldQualifiedName := s.Name.Module + "." + s.Name.Name
	newQualifiedName := s.Name.Module + "." + s.NewName

	module, err := findModuleDeps(ctx, deps, s.Name.Module)
	if err != nil {
		return err
	}

	dm, err := getDomainModelGenCachedDeps(ctx, deps, module.ID)
	if err != nil {
		return mdlerrors.NewBackend("get domain model", err)
	}
	if dm == nil {
		return mdlerrors.NewNotFound("association", oldQualifiedName)
	}

	found := false
	collision := false
	for _, assocElem := range dm.AssociationsItems() {
		assoc, ok := assocElem.(*genDm.Association)
		if !ok {
			continue
		}
		if assoc.Name() == s.Name.Name {
			found = true
		} else if assoc.Name() == s.NewName {
			collision = true
		}
	}
	if !found {
		return mdlerrors.NewNotFound("association", oldQualifiedName)
	}
	if collision {
		return mdlerrors.NewValidationf("association %s already exists in module %s", s.NewName, s.Name.Module)
	}

	hits, err := deps.RenameManager.RenameReferences(oldQualifiedName, newQualifiedName, s.DryRun)
	if err != nil {
		return mdlerrors.NewBackend("scan references", err)
	}

	if s.DryRun {
		printRenameReportFn(deps.Output, oldQualifiedName, newQualifiedName, hits)
		return nil
	}

	for _, assocElem := range dm.AssociationsItems() {
		assoc, ok := assocElem.(*genDm.Association)
		if !ok {
			continue
		}
		if assoc.Name() == s.Name.Name {
			assoc.SetName(s.NewName)
			break
		}
	}
	if err := deps.DomainModelWriter.UpdateDomainModelGen(dm); err != nil {
		return mdlerrors.NewBackend("update association name", err)
	}
	setDomainModelGenCachedDeps(deps, module.ID, dm)

	invalidateHierarchyDeps(deps)
	invalidateDomainModelsCacheDeps(deps)

	fmt.Fprintf(deps.Output, "Renamed association: %s → %s\n", oldQualifiedName, newQualifiedName)
	if len(hits) > 0 {
		fmt.Fprintf(deps.Output, "Updated %d reference(s) in %d document(s)\n", totalRefCount(hits), len(hits))
	}
	return nil
}

func execRenameJavaActionFn(ctx context.Context, s *ast.RenameStmt, deps *HandlerDeps) error {
	oldQualifiedName := s.Name.Module + "." + s.Name.Name
	newQualifiedName := s.Name.Module + "." + s.NewName

	pairs, err := listJavaActionsWithContainerGenDeps(deps)
	if err != nil {
		return mdlerrors.NewBackend("list java actions", err)
	}
	h, err := GetOrBuildHierarchy(deps)
	if err != nil {
		return err
	}
	found := false
	collision := false
	for _, p := range pairs {
		if p.Elem == nil {
			continue
		}
		startID := model.ID(p.ContainerID)
		if startID == "" {
			startID = model.ID(p.Elem.ID())
		}
		modID := h.FindModuleID(startID)
		if h.GetModuleName(modID) != s.Name.Module {
			continue
		}
		name := p.Elem.Name()
		if name == s.Name.Name {
			found = true
		} else if name == s.NewName {
			collision = true
		}
	}
	if !found {
		return mdlerrors.NewNotFound("java action", oldQualifiedName)
	}
	if collision {
		return mdlerrors.NewValidationf("java action %s already exists in module %s", s.NewName, s.Name.Module)
	}

	hits, err := deps.RenameManager.RenameReferences(oldQualifiedName, newQualifiedName, s.DryRun)
	if err != nil {
		return mdlerrors.NewBackend("scan references", err)
	}

	if s.DryRun {
		printRenameReportFn(deps.Output, oldQualifiedName, newQualifiedName, hits)
		return nil
	}

	if err := deps.RenameManager.RenameDocumentByName(s.Name.Module, s.Name.Name, s.NewName); err != nil {
		return mdlerrors.NewBackend("rename java action", err)
	}
	if err := deps.JavaActionWriter.RenameJavaSourceFile(s.Name.Module, s.Name.Name, s.NewName); err != nil {
		return mdlerrors.NewBackend("rename java source file", err)
	}

	invalidateHierarchyDeps(deps)
	deps.Cache.microflowNames = nil
	deps.Cache.Invalidate(
		CacheDomainMicroflows, CacheDomainNanoflows,
		CacheDomainPages, CacheDomainLayouts, CacheDomainSnippets,
		CacheDomainWorkflows,
		CacheDomainJavaActions, CacheDomainJavaScriptActions,
	)

	fmt.Fprintf(deps.Output, "Renamed java action: %s → %s\n", oldQualifiedName, newQualifiedName)
	if len(hits) > 0 {
		fmt.Fprintf(deps.Output, "Updated %d reference(s) in %d document(s)\n", totalRefCount(hits), len(hits))
	}
	return nil
}

func printRenameReportFn(output io.Writer, oldName, newName string, hits []types.RenameHit) {
	fmt.Fprintf(output, "Would rename: %s → %s\n", oldName, newName)
	fmt.Fprintf(output, "References found: %d in %d document(s)\n", totalRefCount(hits), len(hits))

	for _, h := range hits {
		label := h.Name
		if label == "" {
			label = h.UnitID
		}
		typeName := h.UnitType
		if idx := strings.Index(typeName, "$"); idx >= 0 {
			typeName = typeName[idx+1:]
		}
		fmt.Fprintf(output, "  %s (%s) — %d reference(s)\n", label, typeName, h.Count)
	}
}


