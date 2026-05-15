// SPDX-License-Identifier: Apache-2.0

// Package executor - Autocomplete support for LSP and REPL.
// Returns qualified names for modules, entities, microflows, pages, etc.
package executor

import (
	"context"

	"github.com/mendixlabs/mxcli/model"
	genDm "github.com/mendixlabs/mxcli/modelsdk/gen/domainmodels"
)

// getModuleNames returns a list of all module names for autocomplete.
func getModuleNames(ctx *ExecContext) []string {
	if !ctx.Connected() {
		return nil
	}
	modules, err := ctx.Backend.ListModules()
	if err != nil {
		return nil
	}
	names := make([]string, 0, len(modules))
	for _, m := range modules {
		names = append(names, m.Name)
	}
	return names
}

// getMicroflowNamesAC returns qualified microflow names, optionally filtered by module.
func getMicroflowNamesAC(ctx *ExecContext, moduleFilter string) []string {
	if !ctx.Connected() {
		return nil
	}
	h, err := getHierarchy(ctx)
	if err != nil {
		return nil
	}
	mfs, err := listMicroflowsWithContainerGen(ctx)
	if err != nil {
		return nil
	}
	names := make([]string, 0)
	for _, item := range mfs {
		modID := h.FindModuleID(item.ContainerUUID)
		modName := h.GetModuleName(modID)
		if moduleFilter == "" || modName == moduleFilter {
			names = append(names, modName+"."+item.MF.Name())
		}
	}
	return names
}

// getEntityNamesAC returns qualified entity names, optionally filtered by module.
func getEntityNamesAC(ctx *ExecContext, moduleFilter string) []string {
	if !ctx.Connected() {
		return nil
	}
	h, err := getHierarchy(ctx)
	if err != nil {
		return nil
	}
	dms, err := cachedDomainModelsGen(ctx)
	if err != nil {
		return nil
	}
	names := make([]string, 0)
	for _, dm := range dms {
		if dm == nil {
			continue
		}
		modID := h.FindModuleID(model.ID(dm.ID()))
		modName := h.GetModuleName(modID)
		if moduleFilter == "" || modName == moduleFilter {
			for _, entityElem := range dm.EntitiesItems() {
				ent, ok := entityElem.(*genDm.Entity)
				if !ok {
					continue
				}
				names = append(names, modName+"."+ent.Name())
			}
		}
	}
	return names
}

// getPageNamesAC returns qualified page names, optionally filtered by module.
func getPageNamesAC(ctx *ExecContext, moduleFilter string) []string {
	if !ctx.Connected() {
		return nil
	}
	h, err := getHierarchy(ctx)
	if err != nil {
		return nil
	}
	pairs, err := listPagesWithContainerGen(ctx)
	if err != nil {
		return nil
	}
	names := make([]string, 0)
	for _, pair := range pairs {
		p := pair.Elem
		modID := h.FindModuleID(model.ID(pair.ContainerID))
		modName := h.GetModuleName(modID)
		if moduleFilter == "" || modName == moduleFilter {
			names = append(names, modName+"."+p.Name())
		}
	}
	return names
}

// getSnippetNamesAC returns qualified snippet names, optionally filtered by module.
func getSnippetNamesAC(ctx *ExecContext, moduleFilter string) []string {
	if !ctx.Connected() {
		return nil
	}
	h, err := getHierarchy(ctx)
	if err != nil {
		return nil
	}
	snippetPairs, err := listSnippetsWithContainerGen(ctx)
	if err != nil {
		return nil
	}
	names := make([]string, 0)
	for _, pair := range snippetPairs {
		s := pair.Elem
		modID := h.FindModuleID(model.ID(pair.ContainerID))
		modName := h.GetModuleName(modID)
		if moduleFilter == "" || modName == moduleFilter {
			names = append(names, modName+"."+s.Name())
		}
	}
	return names
}

// getAssociationNamesAC returns qualified association names, optionally filtered by module.
func getAssociationNamesAC(ctx *ExecContext, moduleFilter string) []string {
	if !ctx.Connected() {
		return nil
	}
	h, err := getHierarchy(ctx)
	if err != nil {
		return nil
	}
	dms, err := cachedDomainModelsGen(ctx)
	if err != nil {
		return nil
	}
	names := make([]string, 0)
	for _, dm := range dms {
		if dm == nil {
			continue
		}
		modID := h.FindModuleID(model.ID(dm.ID()))
		modName := h.GetModuleName(modID)
		if moduleFilter == "" || modName == moduleFilter {
			for _, assocElem := range dm.AssociationsItems() {
				assoc, ok := assocElem.(*genDm.Association)
				if !ok {
					continue
				}
				names = append(names, modName+"."+assoc.Name())
			}
		}
	}
	return names
}

// getEnumerationNamesAC returns qualified enumeration names, optionally filtered by module.
func getEnumerationNamesAC(ctx *ExecContext, moduleFilter string) []string {
	if !ctx.Connected() {
		return nil
	}
	h, err := getHierarchy(ctx)
	if err != nil {
		return nil
	}
	enums, err := ctx.Backend.ListEnumerations()
	if err != nil {
		return nil
	}
	names := make([]string, 0)
	for _, enum := range enums {
		modID := h.FindModuleID(enum.ContainerID)
		modName := h.GetModuleName(modID)
		if moduleFilter == "" || modName == moduleFilter {
			names = append(names, modName+"."+enum.Name)
		}
	}
	return names
}

// getLayoutNamesAC returns qualified layout names, optionally filtered by module.
func getLayoutNamesAC(ctx *ExecContext, moduleFilter string) []string {
	if !ctx.Connected() {
		return nil
	}
	h, err := getHierarchy(ctx)
	if err != nil {
		return nil
	}
	layoutPairs, err := listLayoutsWithContainerGen(ctx)
	if err != nil {
		return nil
	}
	names := make([]string, 0)
	for _, pair := range layoutPairs {
		layout := pair.Elem
		modID := h.FindModuleID(model.ID(pair.ContainerID))
		modName := h.GetModuleName(modID)
		if moduleFilter == "" || modName == moduleFilter {
			names = append(names, modName+"."+layout.Name())
		}
	}
	return names
}

// getJavaActionNamesAC returns qualified Java action names, optionally filtered by module.
func getJavaActionNamesAC(ctx *ExecContext, moduleFilter string) []string {
	if !ctx.Connected() {
		return nil
	}
	h, err := getHierarchy(ctx)
	if err != nil {
		return nil
	}
	pairs, err := listJavaActionsWithContainerGen(ctx)
	if err != nil {
		return nil
	}
	names := make([]string, 0, len(pairs))
	for _, p := range pairs {
		if p.Elem == nil {
			continue
		}
		modID := h.FindModuleID(model.ID(p.ContainerID))
		modName := h.GetModuleName(modID)
		if moduleFilter == "" || modName == moduleFilter {
			names = append(names, modName+"."+p.Elem.Name())
		}
	}
	return names
}

// getODataClientNamesAC returns qualified consumed OData service names, optionally filtered by module.
func getODataClientNamesAC(ctx *ExecContext, moduleFilter string) []string {
	if !ctx.Connected() {
		return nil
	}
	h, err := getHierarchy(ctx)
	if err != nil {
		return nil
	}
	services, err := ctx.Backend.ListConsumedODataServices()
	if err != nil {
		return nil
	}
	names := make([]string, 0)
	for _, svc := range services {
		modID := h.FindModuleID(svc.ContainerID)
		modName := h.GetModuleName(modID)
		if moduleFilter == "" || modName == moduleFilter {
			names = append(names, modName+"."+svc.Name)
		}
	}
	return names
}

// getODataServiceNamesAC returns qualified published OData service names, optionally filtered by module.
func getODataServiceNamesAC(ctx *ExecContext, moduleFilter string) []string {
	if !ctx.Connected() {
		return nil
	}
	h, err := getHierarchy(ctx)
	if err != nil {
		return nil
	}
	services, err := ctx.Backend.ListPublishedODataServices()
	if err != nil {
		return nil
	}
	names := make([]string, 0)
	for _, svc := range services {
		modID := h.FindModuleID(svc.ContainerID)
		modName := h.GetModuleName(modID)
		if moduleFilter == "" || modName == moduleFilter {
			names = append(names, modName+"."+svc.Name)
		}
	}
	return names
}

// getRestClientNamesAC returns qualified consumed REST service names, optionally filtered by module.
func getRestClientNamesAC(ctx *ExecContext, moduleFilter string) []string {
	if !ctx.Connected() {
		return nil
	}
	h, err := getHierarchy(ctx)
	if err != nil {
		return nil
	}
	services, err := ctx.Backend.ListConsumedRestServices()
	if err != nil {
		return nil
	}
	names := make([]string, 0)
	for _, svc := range services {
		modID := h.FindModuleID(svc.ContainerID)
		modName := h.GetModuleName(modID)
		if moduleFilter == "" || modName == moduleFilter {
			names = append(names, modName+"."+svc.Name)
		}
	}
	return names
}

// getDatabaseConnectionNamesAC returns qualified database connection names, optionally filtered by module.
func getDatabaseConnectionNamesAC(ctx *ExecContext, moduleFilter string) []string {
	if !ctx.Connected() {
		return nil
	}
	h, err := getHierarchy(ctx)
	if err != nil {
		return nil
	}
	connections, err := ctx.Backend.ListDatabaseConnections()
	if err != nil {
		return nil
	}
	names := make([]string, 0)
	for _, conn := range connections {
		modID := h.FindModuleID(conn.ContainerID)
		modName := h.GetModuleName(modID)
		if moduleFilter == "" || modName == moduleFilter {
			names = append(names, modName+"."+conn.Name)
		}
	}
	return names
}

// getBusinessEventServiceNamesAC returns qualified business event service names, optionally filtered by module.
func getBusinessEventServiceNamesAC(ctx *ExecContext, moduleFilter string) []string {
	if !ctx.Connected() {
		return nil
	}
	h, err := getHierarchy(ctx)
	if err != nil {
		return nil
	}
	services, err := ctx.Backend.ListBusinessEventServices()
	if err != nil {
		return nil
	}
	names := make([]string, 0)
	for _, svc := range services {
		modID := h.FindModuleID(svc.ContainerID)
		modName := h.GetModuleName(modID)
		if moduleFilter == "" || modName == moduleFilter {
			names = append(names, modName+"."+svc.Name)
		}
	}
	return names
}

// getJsonStructureNamesAC returns qualified JSON structure names, optionally filtered by module.
func getJsonStructureNamesAC(ctx *ExecContext, moduleFilter string) []string {
	if !ctx.Connected() {
		return nil
	}
	h, err := getHierarchy(ctx)
	if err != nil {
		return nil
	}
	structures, err := ctx.Backend.ListJsonStructures()
	if err != nil {
		return nil
	}
	names := make([]string, 0)
	for _, js := range structures {
		modID := h.FindModuleID(js.ContainerID)
		modName := h.GetModuleName(modID)
		if moduleFilter == "" || modName == moduleFilter {
			names = append(names, modName+"."+js.Name)
		}
	}
	return names
}

// ----------------------------------------------------------------------------
// Exported Executor method wrappers (public API for external callers)
// ----------------------------------------------------------------------------

// GetModuleNames returns a list of all module names for autocomplete.
func (e *Executor) GetModuleNames() []string {
	return getModuleNames(e.newExecContext(context.Background()))
}

// GetMicroflowNames returns qualified microflow names, optionally filtered by module.
func (e *Executor) GetMicroflowNames(moduleFilter string) []string {
	return getMicroflowNamesAC(e.newExecContext(context.Background()), moduleFilter)
}

// GetEntityNames returns qualified entity names, optionally filtered by module.
func (e *Executor) GetEntityNames(moduleFilter string) []string {
	return getEntityNamesAC(e.newExecContext(context.Background()), moduleFilter)
}

// GetPageNames returns qualified page names, optionally filtered by module.
func (e *Executor) GetPageNames(moduleFilter string) []string {
	return getPageNamesAC(e.newExecContext(context.Background()), moduleFilter)
}

// GetSnippetNames returns qualified snippet names, optionally filtered by module.
func (e *Executor) GetSnippetNames(moduleFilter string) []string {
	return getSnippetNamesAC(e.newExecContext(context.Background()), moduleFilter)
}

// GetAssociationNames returns qualified association names, optionally filtered by module.
func (e *Executor) GetAssociationNames(moduleFilter string) []string {
	return getAssociationNamesAC(e.newExecContext(context.Background()), moduleFilter)
}

// GetEnumerationNames returns qualified enumeration names, optionally filtered by module.
func (e *Executor) GetEnumerationNames(moduleFilter string) []string {
	return getEnumerationNamesAC(e.newExecContext(context.Background()), moduleFilter)
}

// GetLayoutNames returns qualified layout names, optionally filtered by module.
func (e *Executor) GetLayoutNames(moduleFilter string) []string {
	return getLayoutNamesAC(e.newExecContext(context.Background()), moduleFilter)
}

// GetJavaActionNames returns qualified Java action names, optionally filtered by module.
func (e *Executor) GetJavaActionNames(moduleFilter string) []string {
	return getJavaActionNamesAC(e.newExecContext(context.Background()), moduleFilter)
}

// GetODataClientNames returns qualified consumed OData service names, optionally filtered by module.
func (e *Executor) GetODataClientNames(moduleFilter string) []string {
	return getODataClientNamesAC(e.newExecContext(context.Background()), moduleFilter)
}

// GetODataServiceNames returns qualified published OData service names, optionally filtered by module.
func (e *Executor) GetODataServiceNames(moduleFilter string) []string {
	return getODataServiceNamesAC(e.newExecContext(context.Background()), moduleFilter)
}

// GetRestClientNames returns qualified consumed REST service names, optionally filtered by module.
func (e *Executor) GetRestClientNames(moduleFilter string) []string {
	return getRestClientNamesAC(e.newExecContext(context.Background()), moduleFilter)
}

// GetDatabaseConnectionNames returns qualified database connection names, optionally filtered by module.
func (e *Executor) GetDatabaseConnectionNames(moduleFilter string) []string {
	return getDatabaseConnectionNamesAC(e.newExecContext(context.Background()), moduleFilter)
}

// GetBusinessEventServiceNames returns qualified business event service names, optionally filtered by module.
func (e *Executor) GetBusinessEventServiceNames(moduleFilter string) []string {
	return getBusinessEventServiceNamesAC(e.newExecContext(context.Background()), moduleFilter)
}

// GetJsonStructureNames returns qualified JSON structure names, optionally filtered by module.
func (e *Executor) GetJsonStructureNames(moduleFilter string) []string {
	return getJsonStructureNamesAC(e.newExecContext(context.Background()), moduleFilter)
}
