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
	return getModuleNamesDeps(ctx.Deps)
}

// getMicroflowNamesAC returns qualified microflow names, optionally filtered by module.
func getMicroflowNamesAC(ctx *ExecContext, moduleFilter string) []string {
	return getMicroflowNamesACDeps(ctx.Deps, moduleFilter)
}

// getEntityNamesAC returns qualified entity names, optionally filtered by module.
func getEntityNamesAC(ctx *ExecContext, moduleFilter string) []string {
	return getEntityNamesACDeps(ctx.Deps, moduleFilter)
}

// getPageNamesAC returns qualified page names, optionally filtered by module.
func getPageNamesAC(ctx *ExecContext, moduleFilter string) []string {
	return getPageNamesACDeps(ctx.Deps, moduleFilter)
}

// getSnippetNamesAC returns qualified snippet names, optionally filtered by module.
func getSnippetNamesAC(ctx *ExecContext, moduleFilter string) []string {
	return getSnippetNamesACDeps(ctx.Deps, moduleFilter)
}

// getAssociationNamesAC returns qualified association names, optionally filtered by module.
func getAssociationNamesAC(ctx *ExecContext, moduleFilter string) []string {
	return getAssociationNamesACDeps(ctx.Deps, moduleFilter)
}

// getEnumerationNamesAC returns qualified enumeration names, optionally filtered by module.
func getEnumerationNamesAC(ctx *ExecContext, moduleFilter string) []string {
	return getEnumerationNamesACDeps(ctx.Deps, moduleFilter)
}

// getLayoutNamesAC returns qualified layout names, optionally filtered by module.
func getLayoutNamesAC(ctx *ExecContext, moduleFilter string) []string {
	return getLayoutNamesACDeps(ctx.Deps, moduleFilter)
}

// getJavaActionNamesAC returns qualified Java action names, optionally filtered by module.
func getJavaActionNamesAC(ctx *ExecContext, moduleFilter string) []string {
	return getJavaActionNamesACDeps(ctx.Deps, moduleFilter)
}

// getODataClientNamesAC returns qualified consumed OData service names, optionally filtered by module.
func getODataClientNamesAC(ctx *ExecContext, moduleFilter string) []string {
	return getODataClientNamesACDeps(ctx.Deps, moduleFilter)
}

// getODataServiceNamesAC returns qualified published OData service names, optionally filtered by module.
func getODataServiceNamesAC(ctx *ExecContext, moduleFilter string) []string {
	return getODataServiceNamesACDeps(ctx.Deps, moduleFilter)
}

// getRestClientNamesAC returns qualified consumed REST service names, optionally filtered by module.
func getRestClientNamesAC(ctx *ExecContext, moduleFilter string) []string {
	return getRestClientNamesACDeps(ctx.Deps, moduleFilter)
}

// getDatabaseConnectionNamesAC returns qualified database connection names, optionally filtered by module.
func getDatabaseConnectionNamesAC(ctx *ExecContext, moduleFilter string) []string {
	return getDatabaseConnectionNamesACDeps(ctx.Deps, moduleFilter)
}

// getBusinessEventServiceNamesAC returns qualified business event service names, optionally filtered by module.
func getBusinessEventServiceNamesAC(ctx *ExecContext, moduleFilter string) []string {
	return getBusinessEventServiceNamesACDeps(ctx.Deps, moduleFilter)
}

// getJsonStructureNamesAC returns qualified JSON structure names, optionally filtered by module.
func getJsonStructureNamesAC(ctx *ExecContext, moduleFilter string) []string {
	return getJsonStructureNamesACDeps(ctx.Deps, moduleFilter)
}

// ----------------------------------------------------------------------------
// Deps-based autocomplete helpers (use *HandlerDeps directly)
// ----------------------------------------------------------------------------

func getModuleNamesDeps(deps *HandlerDeps) []string {
	if deps.ConnectionManager == nil || !deps.ConnectionManager.IsConnected() {
		return nil
	}
	modules, err := deps.ModuleLister.ListModules()
	if err != nil {
		return nil
	}
	names := make([]string, 0, len(modules))
	for _, m := range modules {
		names = append(names, m.Name)
	}
	return names
}

func getMicroflowNamesACDeps(deps *HandlerDeps, moduleFilter string) []string {
	if deps.ConnectionManager == nil || !deps.ConnectionManager.IsConnected() {
		return nil
	}
	h, err := getHierarchyDeps(deps)
	if err != nil {
		return nil
	}
	mfs, err := deps.MicroflowRepo.ListAll()
	if err != nil {
		return nil
	}
	names := make([]string, 0)
	for _, mf := range mfs {
		if mf == nil {
			continue
		}
		cid, err := deps.MicroflowRepo.GetContainerUUID(model.ID(mf.ID()))
		if err != nil {
			continue
		}
		modID := h.FindModuleID(cid)
		modName := h.GetModuleName(modID)
		if moduleFilter == "" || modName == moduleFilter {
			names = append(names, modName+"."+mf.Name())
		}
	}
	return names
}

func getEntityNamesACDeps(deps *HandlerDeps, moduleFilter string) []string {
	if deps.ConnectionManager == nil || !deps.ConnectionManager.IsConnected() {
		return nil
	}
	h, err := getHierarchyDeps(deps)
	if err != nil {
		return nil
	}
	dms, err := cachedDomainModelsGenDeps(deps)
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

func getPageNamesACDeps(deps *HandlerDeps, moduleFilter string) []string {
	if deps.ConnectionManager == nil || !deps.ConnectionManager.IsConnected() {
		return nil
	}
	h, err := getHierarchyDeps(deps)
	if err != nil {
		return nil
	}
	pairs, err := listPagesWithContainerGenDeps(context.Background(), deps)
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

func getSnippetNamesACDeps(deps *HandlerDeps, moduleFilter string) []string {
	if deps.ConnectionManager == nil || !deps.ConnectionManager.IsConnected() {
		return nil
	}
	h, err := getHierarchyDeps(deps)
	if err != nil {
		return nil
	}
	snippetPairs, err := listSnippetsWithContainerGenDeps(context.Background(), deps)
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

func getAssociationNamesACDeps(deps *HandlerDeps, moduleFilter string) []string {
	if deps.ConnectionManager == nil || !deps.ConnectionManager.IsConnected() {
		return nil
	}
	h, err := getHierarchyDeps(deps)
	if err != nil {
		return nil
	}
	dms, err := cachedDomainModelsGenDeps(deps)
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

func getEnumerationNamesACDeps(deps *HandlerDeps, moduleFilter string) []string {
	if deps.ConnectionManager == nil || !deps.ConnectionManager.IsConnected() {
		return nil
	}
	h, err := getHierarchyDeps(deps)
	if err != nil {
		return nil
	}
	enums, err := deps.EnumerationReader.ListEnumerations()
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

func getLayoutNamesACDeps(deps *HandlerDeps, moduleFilter string) []string {
	if deps.ConnectionManager == nil || !deps.ConnectionManager.IsConnected() {
		return nil
	}
	h, err := getHierarchyDeps(deps)
	if err != nil {
		return nil
	}
	layoutPairs, err := listLayoutsWithContainerGenDeps(deps)
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

func getJavaActionNamesACDeps(deps *HandlerDeps, moduleFilter string) []string {
	if deps.ConnectionManager == nil || !deps.ConnectionManager.IsConnected() {
		return nil
	}
	h, err := getHierarchyDeps(deps)
	if err != nil {
		return nil
	}
	pairs, err := listJavaActionsWithContainerGenDeps(deps)
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

func getODataClientNamesACDeps(deps *HandlerDeps, moduleFilter string) []string {
	if deps.ConnectionManager == nil || !deps.ConnectionManager.IsConnected() {
		return nil
	}
	h, err := getHierarchyDeps(deps)
	if err != nil {
		return nil
	}
	services, err := deps.ServiceLister.ListConsumedODataServices()
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

func getODataServiceNamesACDeps(deps *HandlerDeps, moduleFilter string) []string {
	if deps.ConnectionManager == nil || !deps.ConnectionManager.IsConnected() {
		return nil
	}
	h, err := getHierarchyDeps(deps)
	if err != nil {
		return nil
	}
	services, err := deps.ServiceLister.ListPublishedODataServices()
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

func getRestClientNamesACDeps(deps *HandlerDeps, moduleFilter string) []string {
	if deps.ConnectionManager == nil || !deps.ConnectionManager.IsConnected() {
		return nil
	}
	h, err := getHierarchyDeps(deps)
	if err != nil {
		return nil
	}
	services, err := deps.ServiceLister.ListConsumedRestServices()
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

func getDatabaseConnectionNamesACDeps(deps *HandlerDeps, moduleFilter string) []string {
	if deps.ConnectionManager == nil || !deps.ConnectionManager.IsConnected() {
		return nil
	}
	h, err := getHierarchyDeps(deps)
	if err != nil {
		return nil
	}
	connections, err := deps.ServiceLister.ListDatabaseConnections()
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

func getBusinessEventServiceNamesACDeps(deps *HandlerDeps, moduleFilter string) []string {
	if deps.ConnectionManager == nil || !deps.ConnectionManager.IsConnected() {
		return nil
	}
	h, err := getHierarchyDeps(deps)
	if err != nil {
		return nil
	}
	services, err := deps.ServiceLister.ListBusinessEventServices()
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

func getJsonStructureNamesACDeps(deps *HandlerDeps, moduleFilter string) []string {
	if deps.ConnectionManager == nil || !deps.ConnectionManager.IsConnected() {
		return nil
	}
	h, err := getHierarchyDeps(deps)
	if err != nil {
		return nil
	}
	structures, err := deps.MapperReader.ListJsonStructures()
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
	return getModuleNamesDeps(e.buildHandlerDeps())
}

// GetMicroflowNames returns qualified microflow names, optionally filtered by module.
func (e *Executor) GetMicroflowNames(moduleFilter string) []string {
	return getMicroflowNamesACDeps(e.buildHandlerDeps(), moduleFilter)
}

// GetEntityNames returns qualified entity names, optionally filtered by module.
func (e *Executor) GetEntityNames(moduleFilter string) []string {
	return getEntityNamesACDeps(e.buildHandlerDeps(), moduleFilter)
}

// GetPageNames returns qualified page names, optionally filtered by module.
func (e *Executor) GetPageNames(moduleFilter string) []string {
	return getPageNamesACDeps(e.buildHandlerDeps(), moduleFilter)
}

// GetSnippetNames returns qualified snippet names, optionally filtered by module.
func (e *Executor) GetSnippetNames(moduleFilter string) []string {
	return getSnippetNamesACDeps(e.buildHandlerDeps(), moduleFilter)
}

// GetAssociationNames returns qualified association names, optionally filtered by module.
func (e *Executor) GetAssociationNames(moduleFilter string) []string {
	return getAssociationNamesACDeps(e.buildHandlerDeps(), moduleFilter)
}

// GetEnumerationNames returns qualified enumeration names, optionally filtered by module.
func (e *Executor) GetEnumerationNames(moduleFilter string) []string {
	return getEnumerationNamesACDeps(e.buildHandlerDeps(), moduleFilter)
}

// GetLayoutNames returns qualified layout names, optionally filtered by module.
func (e *Executor) GetLayoutNames(moduleFilter string) []string {
	return getLayoutNamesACDeps(e.buildHandlerDeps(), moduleFilter)
}

// GetJavaActionNames returns qualified Java action names, optionally filtered by module.
func (e *Executor) GetJavaActionNames(moduleFilter string) []string {
	return getJavaActionNamesACDeps(e.buildHandlerDeps(), moduleFilter)
}

// GetODataClientNames returns qualified consumed OData service names, optionally filtered by module.
func (e *Executor) GetODataClientNames(moduleFilter string) []string {
	return getODataClientNamesACDeps(e.buildHandlerDeps(), moduleFilter)
}

// GetODataServiceNames returns qualified published OData service names, optionally filtered by module.
func (e *Executor) GetODataServiceNames(moduleFilter string) []string {
	return getODataServiceNamesACDeps(e.buildHandlerDeps(), moduleFilter)
}

// GetRestClientNames returns qualified consumed REST service names, optionally filtered by module.
func (e *Executor) GetRestClientNames(moduleFilter string) []string {
	return getRestClientNamesACDeps(e.buildHandlerDeps(), moduleFilter)
}

// GetDatabaseConnectionNames returns qualified database connection names, optionally filtered by module.
func (e *Executor) GetDatabaseConnectionNames(moduleFilter string) []string {
	return getDatabaseConnectionNamesACDeps(e.buildHandlerDeps(), moduleFilter)
}

// GetBusinessEventServiceNames returns qualified business event service names, optionally filtered by module.
func (e *Executor) GetBusinessEventServiceNames(moduleFilter string) []string {
	return getBusinessEventServiceNamesACDeps(e.buildHandlerDeps(), moduleFilter)
}

// GetJsonStructureNames returns qualified JSON structure names, optionally filtered by module.
func (e *Executor) GetJsonStructureNames(moduleFilter string) []string {
	return getJsonStructureNamesACDeps(e.buildHandlerDeps(), moduleFilter)
}
