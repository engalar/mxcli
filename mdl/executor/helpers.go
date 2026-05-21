// SPDX-License-Identifier: Apache-2.0

// Package executor - Shared helper functions for module/folder resolution,
// reference validation, and data type conversion.
package executor

import (
	"fmt"
	"strings"

	"github.com/mendixlabs/mxcli/mdl/ast"
	mdlerrors "github.com/mendixlabs/mxcli/mdl/errors"
	"github.com/mendixlabs/mxcli/mdl/types"
	"github.com/mendixlabs/mxcli/model"
	genDm "github.com/mendixlabs/mxcli/modelsdk/gen/domainmodels"
)

// ----------------------------------------------------------------------------
// Module and Folder Resolution
// ----------------------------------------------------------------------------

// getModulesFromCache returns cached modules or loads them.
func getModulesFromCache(ctx *ExecContext) ([]*model.Module, error) {
	if ctx.Cache != nil && ctx.Cache.modules != nil {
		return ctx.Cache.modules, nil
	}
	modules, err := ctx.Backend.ListModules()
	if err != nil {
		return nil, err
	}
	if ctx.Cache != nil {
		ctx.Cache.modules = modules
	}
	return modules, nil
}

// invalidateModuleCache clears the module cache so next lookup gets fresh data.
// Also invalidates the hierarchy cache since new modules affect hierarchy.
func invalidateModuleCache(ctx *ExecContext) {
	if ctx.Cache != nil {
		ctx.Cache.modules = nil
		ctx.Cache.hierarchy = nil
	}
}

func findModule(ctx *ExecContext, name string) (*model.Module, error) {
	// Module name is required - objects must always belong to a module
	if name == "" {
		return nil, mdlerrors.NewValidation("module name is required: objects must be created within a module (use ModuleName.ObjectName syntax)")
	}

	modules, err := getModulesFromCache(ctx)
	if err != nil {
		return nil, mdlerrors.NewBackend("list modules", err)
	}

	for _, m := range modules {
		if m.Name == name {
			return m, nil
		}
	}

	return nil, mdlerrors.NewNotFound("module", name)
}

// findOrCreateModule looks up a module by name, auto-creating it if it doesn't exist
// and the executor has write access. Used by CREATE operations to avoid requiring
// manual module creation.
func findOrCreateModule(ctx *ExecContext, name string) (*model.Module, error) {
	m, err := findModule(ctx, name)
	if err == nil {
		return m, nil
	}
	if !ctx.ConnectedForWrite() || name == "" {
		return nil, err
	}
	// Auto-create the module
	if createErr := execCreateModule(ctx, &ast.CreateModuleStmt{Name: name}); createErr != nil {
		return nil, mdlerrors.NewBackend("auto-create module "+name, createErr)
	}
	return findModule(ctx, name)
}

func findModuleByID(ctx *ExecContext, id model.ID) (*model.Module, error) {
	modules, err := getModulesFromCache(ctx)
	if err != nil {
		return nil, mdlerrors.NewBackend("list modules", err)
	}

	for _, m := range modules {
		if m.ID == id {
			return m, nil
		}
	}

	return nil, mdlerrors.NewNotFoundMsg("module", string(id), "module not found with ID: "+string(id))
}

func findEntity(ctx *ExecContext, moduleName, entityName string) (*genDm.Entity, error) {
	module, err := findModule(ctx, moduleName)
	if err != nil {
		return nil, err
	}
	dm, err := ctx.Backend.GetDomainModelGen(module.ID)
	if err != nil {
		return nil, mdlerrors.NewBackend("get domain model", err)
	}
	for _, entityElem := range dm.EntitiesItems() {
		entity, ok := entityElem.(*genDm.Entity)
		if ok && entity != nil && entity.Name() == entityName {
			return entity, nil
		}
	}
	return nil, mdlerrors.NewNotFound("entity", moduleName+"."+entityName)
}

// resolveFolder resolves a folder path (e.g., "Resources/Images") to a folder ID.
// The path is relative to the given module. If the folder doesn't exist, it creates it.
func resolveFolder(ctx *ExecContext, moduleID model.ID, folderPath string) (model.ID, error) {
	if folderPath == "" {
		return moduleID, nil
	}

	folders, err := ctx.Backend.ListFolders()
	if err != nil {
		return "", mdlerrors.NewBackend("list folders", err)
	}

	// Split path into parts
	parts := strings.Split(folderPath, "/")
	currentContainerID := moduleID

	for _, part := range parts {
		if part == "" {
			continue
		}

		// Find folder with this name under current container
		var foundFolder *types.FolderInfo
		for _, f := range folders {
			if f.ContainerID == currentContainerID && f.Name == part {
				foundFolder = f
				break
			}
		}

		if foundFolder != nil {
			currentContainerID = foundFolder.ID
		} else {
			// Create the folder
			parentID := currentContainerID
			newFolderID, err := createFolder(ctx, part, parentID)
			if err != nil {
				return "", mdlerrors.NewBackend("create folder "+part, err)
			}
			currentContainerID = newFolderID

			// Add to the list so subsequent lookups find it
			folders = append(folders, &types.FolderInfo{
				ID:          newFolderID,
				ContainerID: parentID,
				Name:        part,
			})
		}
	}

	return currentContainerID, nil
}

// createFolder creates a new folder in the project.
func createFolder(ctx *ExecContext, name string, containerID model.ID) (model.ID, error) {
	folder := &model.Folder{
		BaseElement: model.BaseElement{
			ID:       model.ID(types.GenerateID()),
			TypeName: "Projects$Folder",
		},
		ContainerID: containerID,
		Name:        name,
	}

	if err := ctx.Backend.CreateFolder(folder); err != nil {
		return "", err
	}

	return folder.ID, nil
}

// ----------------------------------------------------------------------------
// Reference Existence Checks
// ----------------------------------------------------------------------------

// enumerationExists checks if an enumeration exists in the project.
func enumerationExists(ctx *ExecContext, qualifiedName string) bool {
	if !ctx.Connected() {
		return false
	}

	// Parse the qualified name to get module and enum name
	parts := strings.Split(qualifiedName, ".")
	if len(parts) != 2 {
		return false
	}
	moduleName, enumName := parts[0], parts[1]

	// Find the module to get its ID
	module, err := findModule(ctx, moduleName)
	if err != nil {
		return false
	}

	// Get all enumerations and check if one matches
	enums, err := ctx.Backend.ListEnumerations()
	if err != nil {
		return false
	}

	for _, enum := range enums {
		if enum.ContainerID == module.ID && enum.Name == enumName {
			return true
		}
	}
	return false
}

// ----------------------------------------------------------------------------
// Widget Reference Validation
// ----------------------------------------------------------------------------

// validateWidgetReferences validates all qualified name references in a widget tree.
// It checks DataSource (microflow/nanoflow/entity), Action (page/microflow/nanoflow),
// and Snippet references.
func validateWidgetReferences(ctx *ExecContext, widgets []*ast.WidgetV3, sc *scriptContext) []string {
	if !ctx.Connected() || len(widgets) == 0 {
		return nil
	}

	// Collect all references from the widget tree
	refs := &widgetRefCollector{}
	refs.collectFromWidgets(widgets)

	if refs.empty() {
		return nil
	}

	// Build lookup maps lazily (only for reference types that are actually used)
	var errors []string

	if len(refs.microflows) > 0 {
		known := buildMicroflowQualifiedNamesGen(ctx)
		for _, ref := range refs.microflows {
			if !known[ref] && !sc.microflows[ref] {
				errors = append(errors, fmt.Sprintf("microflow not found: %s", ref))
			}
		}
	}

	if len(refs.nanoflows) > 0 {
		known := buildNanoflowQualifiedNamesGen(ctx)
		for _, ref := range refs.nanoflows {
			if !known[ref] {
				errors = append(errors, fmt.Sprintf("nanoflow not found: %s", ref))
			}
		}
	}

	if len(refs.pages) > 0 {
		known := buildPageQualifiedNames(ctx)
		for _, ref := range refs.pages {
			if !known[ref] && !sc.pages[ref] {
				errors = append(errors, fmt.Sprintf("page not found: %s", ref))
			}
		}
	}

	if len(refs.snippets) > 0 {
		known := buildSnippetQualifiedNames(ctx)
		for _, ref := range refs.snippets {
			if !known[ref] && !sc.snippets[ref] {
				errors = append(errors, fmt.Sprintf("snippet not found: %s", ref))
			}
		}
	}

	if len(refs.entities) > 0 {
		known := buildEntityQualifiedNames(ctx)
		for _, ref := range refs.entities {
			if !known[ref] && !sc.entities[ref] {
				errors = append(errors, fmt.Sprintf("entity not found: %s", ref))
			}
		}
	}

	return errors
}

// widgetRefCollector collects qualified name references from a widget tree.
type widgetRefCollector struct {
	microflows []string
	nanoflows  []string
	pages      []string
	snippets   []string
	entities   []string
}

func (c *widgetRefCollector) empty() bool {
	return len(c.microflows) == 0 && len(c.nanoflows) == 0 &&
		len(c.pages) == 0 && len(c.snippets) == 0 && len(c.entities) == 0
}

func (c *widgetRefCollector) collectFromWidgets(widgets []*ast.WidgetV3) {
	for _, w := range widgets {
		c.collectFromWidget(w)
	}
}

func (c *widgetRefCollector) collectFromWidget(w *ast.WidgetV3) {
	// Check DataSource
	if ds := w.GetDataSource(); ds != nil {
		switch ds.Type {
		case "microflow":
			if ds.Reference != "" {
				c.microflows = append(c.microflows, ds.Reference)
			}
		case "nanoflow":
			if ds.Reference != "" {
				c.nanoflows = append(c.nanoflows, ds.Reference)
			}
		case "database":
			if ds.Reference != "" {
				c.entities = append(c.entities, ds.Reference)
			}
		}
	}

	// Check Action
	if action := w.GetAction(); action != nil {
		c.collectFromAction(action)
	}

	// Check Snippet reference
	if snippet := w.GetSnippet(); snippet != "" {
		c.snippets = append(c.snippets, snippet)
	}

	// Recurse into children
	c.collectFromWidgets(w.Children)
}

func (c *widgetRefCollector) collectFromAction(action *ast.ActionV3) {
	switch action.Type {
	case "showPage":
		if action.Target != "" {
			c.pages = append(c.pages, action.Target)
		}
	case "microflow":
		if action.Target != "" {
			c.microflows = append(c.microflows, action.Target)
		}
	case "nanoflow":
		if action.Target != "" {
			c.nanoflows = append(c.nanoflows, action.Target)
		}
	case "create":
		if action.Target != "" {
			c.entities = append(c.entities, action.Target)
		}
	}
	// Check chained ThenAction
	if action.ThenAction != nil {
		c.collectFromAction(action.ThenAction)
	}
}

// ----------------------------------------------------------------------------
// Qualified Name Builders (used by validation and autocomplete)
// ----------------------------------------------------------------------------

// buildPageQualifiedNames returns a set of all page qualified names in the project.
func buildPageQualifiedNames(ctx *ExecContext) map[string]bool {
	result := make(map[string]bool)
	h, err := getHierarchy(ctx)
	if err != nil {
		return result
	}
	pgPairs, err := listPagesWithContainerGen(ctx)
	if err != nil {
		return result
	}
	for _, pair := range pgPairs {
		pg := pair.Elem
		qn := h.GetQualifiedName(model.ID(pair.ContainerID), pg.Name())
		result[qn] = true
	}
	return result
}

// buildSnippetQualifiedNames returns a set of all snippet qualified names in the project.
func buildSnippetQualifiedNames(ctx *ExecContext) map[string]bool {
	result := make(map[string]bool)
	h, err := getHierarchy(ctx)
	if err != nil {
		return result
	}
	snippetPairs, err := listSnippetsWithContainerGen(ctx)
	if err != nil {
		return result
	}
	for _, pair := range snippetPairs {
		s := pair.Elem
		qn := h.GetQualifiedName(model.ID(pair.ContainerID), s.Name())
		result[qn] = true
	}
	return result
}

// buildEntityQualifiedNames returns a set of all entity qualified names in the project.
func buildEntityQualifiedNames(ctx *ExecContext) map[string]bool {
	result := make(map[string]bool)
	h, err := getHierarchy(ctx)
	if err != nil {
		return result
	}
	dms, err := cachedDomainModelsGen(ctx)
	if err != nil {
		return result
	}
	for _, dm := range dms {
		if dm == nil {
			continue
		}
		modName := h.GetModuleName(h.FindModuleID(model.ID(dm.ID())))
		if modName == "" {
			continue
		}
		for _, entityElem := range dm.EntitiesItems() {
			ent, ok := entityElem.(*genDm.Entity)
			if !ok {
				continue
			}
			result[modName+"."+ent.Name()] = true
		}
	}
	return result
}

// buildNonPersistentEntityQualifiedNames returns the set of non-persistent entity
// qualified names in the project. Used for CE0053 detection in validateFlowBodyReferences.
func buildNonPersistentEntityQualifiedNames(ctx *ExecContext) map[string]bool {
	result := make(map[string]bool)
	h, err := getHierarchy(ctx)
	if err != nil {
		return result
	}
	dms, err := cachedDomainModelsGen(ctx)
	if err != nil {
		return result
	}
	for _, dm := range dms {
		if dm == nil {
			continue
		}
		modName := h.GetModuleName(h.FindModuleID(model.ID(dm.ID())))
		if modName == "" {
			continue
		}
		for _, entityElem := range dm.EntitiesItems() {
			ent, ok := entityElem.(*genDm.Entity)
			if !ok {
				continue
			}
			if !entityPersistableGen(ent) {
				result[modName+"."+ent.Name()] = true
			}
		}
	}
	return result
}

// buildEntityEnumAttrMap returns a map of bare attribute name → enumeration qualified name
// for all enum-typed attributes on the given entity (e.g. "Status" → "Module.OrderStatus").
// Returns an empty map if the entity is not found or has no enum attributes.
func buildEntityEnumAttrMap(ctx *ExecContext, entityQN string) map[string]string {
	result := make(map[string]string)
	if !ctx.Connected() || entityQN == "" {
		return result
	}
	parts := strings.SplitN(entityQN, ".", 2)
	if len(parts) != 2 {
		return result
	}
	h, err := getHierarchy(ctx)
	if err != nil {
		return result
	}
	dms, err := cachedDomainModelsGen(ctx)
	if err != nil {
		return result
	}
	for _, dm := range dms {
		if dm == nil {
			continue
		}
		modName := h.GetModuleName(h.FindModuleID(model.ID(dm.ID())))
		if modName != parts[0] {
			continue
		}
		for _, entityElem := range dm.EntitiesItems() {
			ent, ok := entityElem.(*genDm.Entity)
			if !ok || ent.Name() != parts[1] {
				continue
			}
			for _, attrElem := range ent.AttributesItems() {
				attr, ok := attrElem.(*genDm.Attribute)
				if !ok {
					continue
				}
				if enumType, ok := attr.Type().(*genDm.EnumerationAttributeType); ok {
					result[attr.Name()] = enumType.EnumerationQualifiedName()
				}
			}
			return result
		}
	}
	return result
}

// buildJavaActionQualifiedNames returns a set of all java action qualified names in the project.
func buildJavaActionQualifiedNames(ctx *ExecContext) map[string]bool {
	result := make(map[string]bool)
	h, err := getHierarchy(ctx)
	if err != nil {
		return result
	}
	pairs, err := listJavaActionsWithContainerGen(ctx)
	if err != nil {
		return result
	}
	for _, p := range pairs {
		if p.Elem == nil {
			continue
		}
		qn := h.GetQualifiedName(model.ID(p.ContainerID), p.Elem.Name())
		result[qn] = true
	}
	return result
}

func buildJavaScriptActionQualifiedNames(ctx *ExecContext) map[string]bool {
	result := make(map[string]bool)
	h, err := getHierarchy(ctx)
	if err != nil {
		return result
	}
	// Stage 3.3.2.C1: source from gen-typed pairs via the cache helper.
	// ContainerID comes from the MPR Unit table (codec strips Container linkage).
	pairs, err := listJavaScriptActionsWithContainerGen(ctx)
	if err != nil {
		return result
	}
	for _, p := range pairs {
		if p.Elem == nil {
			continue
		}
		qn := h.GetQualifiedName(model.ID(p.ContainerID), p.Elem.Name())
		result[qn] = true
	}
	return result
}

// ----------------------------------------------------------------------------
// Executor method wrappers (for callers in unmigrated files)
// ----------------------------------------------------------------------------
