// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"log"
	"strings"

	"github.com/mendixlabs/mxcli/mdl/ast"
	mdlerrors "github.com/mendixlabs/mxcli/mdl/errors"
	"github.com/mendixlabs/mxcli/model"
	genDt "github.com/mendixlabs/mxcli/modelsdk/gen/datatypes"
	genDm "github.com/mendixlabs/mxcli/modelsdk/gen/domainmodels"
	genPg "github.com/mendixlabs/mxcli/modelsdk/gen/pages"
)

// ============================================================================
// V3 Page Builder
// ============================================================================

// buildPageV3 creates a *genPg.Page from a CreatePageStmtV3. The builder's
// lastContainerID field is set to the resolved container (folder or module).
func (pb *pageBuilder) buildPageV3(s *ast.CreatePageStmtV3) (*genPg.Page, error) {
	// Resolve folder if specified
	containerID := pb.moduleID
	if s.Folder != "" {
		folderID, err := pb.resolveFolder(s.Folder)
		if err != nil {
			return nil, mdlerrors.NewBackend("resolve folder "+s.Folder, err)
		}
		containerID = folderID
	}
	pb.lastContainerID = containerID

	page := genPg.NewPage()
	assignFreshID(page)
	page.SetName(s.Name.Name)
	page.SetDocumentation(s.Documentation)
	page.SetExcluded(s.Excluded)
	page.SetMarkAsUsed(false)
	page.SetExportLevel("Hidden")
	page.SetAppearance(newDefaultAppearance())
	page.SetCanvasWidth(800)
	page.SetCanvasHeight(600)
	if s.URL != "" {
		page.SetUrl(s.URL)
	}

	// Set title: Mendix stores page Title as Texts$Text (not Forms$ClientTemplate).
	// Using genSimpleLabel (which wraps in ClientTemplate) would cause a
	// StorageLoadException in Studio Pro 11 ("ClientTemplate cannot be converted to Text").
	if s.Title != "" {
		page.SetTitle(genSimpleText(s.Title))
	}

	// Build parameters FIRST so paramScope/paramEntityNames are populated
	// before widget building (widgets may reference parameters as datasources).
	for _, param := range s.Parameters {
		pp := genPg.NewPageParameter()
		assignFreshID(pp)
		pp.SetName(param.Name)
		pp.SetIsRequired(true)

		if param.EntityType.Name != "" {
			// Entity-typed parameter: build a proper DataTypes$ObjectType nested element.
			// Previously used setRawBSONField("ParameterType_type/entity") which wrote flat
			// keys instead of a nested ParameterType document, causing DataTypes$UnknownType
			// on read-back (CE0170 / CE0566).
			entityID, err := pb.resolveEntity(param.EntityType)
			if err != nil {
				return nil, mdlerrors.NewBackend("resolve entity "+param.EntityType.String(), err)
			}
			entityName := param.EntityType.String()
			pb.paramScope[param.Name] = entityID
			pb.paramEntityNames[param.Name] = entityName
			objType := genDt.NewObjectType()
			assignFreshID(objType)
			objType.SetEntityQualifiedName(entityName)
			pp.SetParameterType(objType)
		} else if bsonType := pageParamBSONType(param.Type); bsonType != "" {
			// Primitive type: build the corresponding gen DataType element.
			primType := newPrimPageParamType(bsonType)
			if primType != nil {
				if withID, ok := primType.(genElementWithID); ok {
					assignFreshID(withID)
				}
				pp.SetParameterType(primType)
			}
		}

		page.AddParameters(pp)
	}

	// Build variables
	for _, v := range s.Variables {
		lv := genPg.NewLocalVariable()
		assignFreshID(lv)
		lv.SetName(v.Name)
		lv.SetVariableType(mdlTypeToDataTypeElement(v.DataType))
		if v.DefaultValue != "" {
			setRawBSONField(lv, "DefaultValue", v.DefaultValue)
		}
		page.AddVariables(lv)
	}

	// Resolve layout and build LayoutCall (after parameters so widgets can use paramScope)
	if s.Layout != "" {
		layoutID, err := pb.resolveLayout(s.Layout)
		if err != nil {
			log.Printf("warning: layout %s not found", s.Layout)
		} else {
			_ = layoutID

			lc := genPg.NewLayoutCall()
			assignFreshID(lc)
			// Mendix stores the layout name under the "Form" BSON key (historic naming).
			// After the types.go fix, initLayoutCall() uses "Form" as the property name,
			// so SetLayoutQualifiedName now writes "Form" directly.
			lc.SetLayoutQualifiedName(s.Layout)

			mainPlaceholderRef := pb.getMainPlaceholderRef(s.Layout)

			arg := genPg.NewLayoutCallArgument()
			assignFreshID(arg)
			// The gen type is "Forms$LayoutCallArgument" but Mendix uses "Forms$FormCallArgument"
			// as the storage type. Override so Studio Pro can parse the argument correctly.
			arg.SetTypeName("Forms$FormCallArgument")
			arg.SetParameterQualifiedName(mainPlaceholderRef)

			if len(s.Widgets) > 0 {
				// SP11.6.6 reads FormCallArgument.Widgets (plural flat list).
				// The old pattern wrapped content in a DivContainer and stored it via
				// SetWidget (singular) — SP11.6.6 cannot find this content.
				// Use AddWidgets to write a flat Widgets list instead.
				expanded, err := pb.expandFragments(s.Widgets)
				if err != nil {
					return nil, err
				}
				for _, astWidget := range expanded {
					w, err := pb.buildWidgetV3(astWidget)
					if err != nil {
						return nil, mdlerrors.NewBackend("build widget", err)
					}
					arg.AddWidgets(w)
				}
			}

			lc.AddArguments(arg)
			page.SetLayoutCall(lc)
		}
	}

	return page, nil
}

// buildSnippetV3 creates a *genPg.Snippet from a CreateSnippetStmtV3.
// The builder's lastContainerID field is set to the resolved container.
func (pb *pageBuilder) buildSnippetV3(s *ast.CreateSnippetStmtV3) (*genPg.Snippet, error) {
	// Resolve folder if specified
	containerID := pb.moduleID
	if s.Folder != "" {
		folderID, err := pb.resolveFolder(s.Folder)
		if err != nil {
			return nil, mdlerrors.NewBackend("resolve folder "+s.Folder, err)
		}
		containerID = folderID
	}
	pb.lastContainerID = containerID

	snippet := genPg.NewSnippet()
	assignFreshID(snippet)
	snippet.SetName(s.Name.Name)
	snippet.SetDocumentation(s.Documentation)

	// Build parameters
	for _, param := range s.Parameters {
		sp := genPg.NewSnippetParameter()
		assignFreshID(sp)
		sp.SetName(param.Name)

		if param.EntityType.Name != "" {
			entityID, err := pb.resolveEntity(param.EntityType)
			if err != nil {
				return nil, mdlerrors.NewBackend("resolve entity "+param.EntityType.String(), err)
			}
			entityName := param.EntityType.String()
			pb.paramScope[param.Name] = entityID
			pb.paramEntityNames[param.Name] = entityName
			// Build a proper DataTypes$ObjectType nested element (same fix as page params).
			objType := genDt.NewObjectType()
			assignFreshID(objType)
			objType.SetEntityQualifiedName(entityName)
			sp.SetParameterType(objType)
		}

		snippet.AddParameters(sp)
	}

	// Build variables
	for _, v := range s.Variables {
		lv := genPg.NewLocalVariable()
		assignFreshID(lv)
		lv.SetName(v.Name)
		lv.SetVariableType(mdlTypeToDataTypeElement(v.DataType))
		if v.DefaultValue != "" {
			setRawBSONField(lv, "DefaultValue", v.DefaultValue)
		}
		snippet.AddVariables(lv)
	}

	// Build widgets (expanding fragments)
	pb.isSnippet = true
	defer func() { pb.isSnippet = false }()

	expanded, err := pb.expandFragments(s.Widgets)
	if err != nil {
		return nil, err
	}
	for _, astWidget := range expanded {
		w, err := pb.buildWidgetV3(astWidget)
		if err != nil {
			return nil, mdlerrors.NewBackend("build widget", err)
		}
		snippet.AddWidgets(w)
	}

	return snippet, nil
}

// =============================================================================
// Helper functions
// =============================================================================

// entityQNByID returns the qualified name (Module.Entity) for a given entity ID
// by scanning all domain models. Returns "" if not found.
func (pb *pageBuilder) entityQNByID(entityID model.ID) string {
	if entityID == "" {
		return ""
	}
	pairs, err := pb.getDomainModelsWithContainer()
	if err != nil {
		return ""
	}
	for _, pair := range pairs {
		if pair.DM == nil {
			continue
		}
		for _, elem := range pair.DM.EntitiesItems() {
			e, ok := elem.(*genDm.Entity)
			if !ok {
				continue
			}
			if model.ID(e.ID()) == entityID {
				modName := pb.moduleNameByID(pair.ContainerID)
				if modName == "" {
					return e.Name()
				}
				return modName + "." + e.Name()
			}
		}
	}
	return ""
}

// moduleNameByID returns the module name for a given module ID. Cached via hierarchy.
func (pb *pageBuilder) moduleNameByID(moduleID model.ID) string {
	if moduleID == "" {
		return ""
	}
	modules := pb.getModules()
	for _, m := range modules {
		if m.ID == moduleID {
			return m.Name
		}
	}
	return ""
}

// getMicroflowReturnEntityName looks up a microflow and returns its return type entity name.
func (pb *pageBuilder) getMicroflowReturnEntityName(qualifiedName string) string {
	if pb.execCache != nil && pb.execCache.createdMicroflows != nil {
		if info, ok := pb.execCache.createdMicroflows[qualifiedName]; ok && info.ReturnEntityName != "" {
			return info.ReturnEntityName
		}
	}

	parts := strings.Split(qualifiedName, ".")
	if len(parts) < 2 {
		return ""
	}
	moduleName := parts[0]
	mfName := strings.Join(parts[1:], ".")

	mfs, err := pb.getMicroflows()
	if err != nil {
		return ""
	}

	h, err := pb.getHierarchy()
	if err != nil {
		return ""
	}

	for _, mf := range mfs {
		if mf == nil || pb.microflowsRepo == nil {
			continue
		}
		containerID, _ := pb.microflowsRepo.GetContainerUUID(model.ID(mf.ID()))
		modName := h.GetModuleName(h.FindModuleID(containerID))
		if modName == moduleName && mf.Name() == mfName {
			if qn := extractEntityFromGenReturnType(mf.ReturnType()); qn != "" {
				return qn
			}
			if rt := mf.MicroflowReturnType(); rt != nil {
				switch t := rt.(type) {
				case *genDt.ObjectType:
					if qn := t.EntityQualifiedName(); qn != "" {
						return qn
					}
				case *genDt.ListType:
					if qn := t.EntityQualifiedName(); qn != "" {
						return qn
					}
				}
			}
			return ""
		}
	}

	return ""
}

// getNanoflowReturnEntityName looks up a nanoflow and returns its return type entity name.
func (pb *pageBuilder) getNanoflowReturnEntityName(qualifiedName string) string {
	// Check session-local cache first (mirrors getMicroflowReturnEntityName).
	if pb.execCache != nil && pb.execCache.createdNanoflows != nil {
		if info, ok := pb.execCache.createdNanoflows[qualifiedName]; ok && info.ReturnEntityName != "" {
			return info.ReturnEntityName
		}
	}

	parts := strings.Split(qualifiedName, ".")
	var moduleName, name string
	if len(parts) >= 2 {
		moduleName = parts[0]
		name = parts[1]
	} else {
		moduleName = pb.moduleName
		name = qualifiedName
	}

	if pb.nanoflowsRepo == nil {
		return ""
	}
	nanoflows, err := pb.nanoflowsRepo.List("")
	if err != nil {
		return ""
	}

	h, err := pb.getHierarchy()
	if err != nil {
		return ""
	}

	for _, nf := range nanoflows {
		if nf == nil {
			continue
		}
		containerID, _ := pb.microflowsRepo.GetContainerUUID(model.ID(nf.ID()))
		modName := h.GetModuleName(h.FindModuleID(containerID))
		if modName == moduleName && nf.Name() == name {
			if qn := extractEntityFromGenReturnType(nf.ReturnType()); qn != "" {
				return qn
			}
			if rt := nf.MicroflowReturnType(); rt != nil {
				switch t := rt.(type) {
				case *genDt.ObjectType:
					if qn := t.EntityQualifiedName(); qn != "" {
						return qn
					}
				case *genDt.ListType:
					if qn := t.EntityQualifiedName(); qn != "" {
						return qn
					}
				}
			}
			return ""
		}
	}

	return ""
}

func (pb *pageBuilder) extractModule(qualifiedName string) string {
	qualifiedName = unquoteQualifiedName(qualifiedName)
	parts := strings.Split(qualifiedName, ".")
	if len(parts) >= 2 {
		return parts[0]
	}
	return pb.moduleName
}

func (pb *pageBuilder) extractName(qualifiedName string) string {
	qualifiedName = unquoteQualifiedName(qualifiedName)
	parts := strings.Split(qualifiedName, ".")
	if len(parts) >= 2 {
		return parts[1]
	}
	return qualifiedName
}

func (pb *pageBuilder) getEntityNameByID(entityID model.ID) (string, error) {
	pairs, err := pb.getDomainModelsWithContainer()
	if err != nil {
		return "", err
	}

	modules := pb.getModules()
	moduleNames := make(map[model.ID]string)
	for _, m := range modules {
		moduleNames[m.ID] = m.Name
	}

	for _, pair := range pairs {
		for _, elem := range pair.DM.EntitiesItems() {
			e, ok := elem.(*genDm.Entity)
			if !ok {
				continue
			}
			if model.ID(e.ID()) == entityID {
				moduleName := moduleNames[pair.ContainerID]
				return moduleName + "." + e.Name(), nil
			}
		}
	}
	return "", mdlerrors.NewNotFound("entity", string(entityID))
}

// resolveNanoflowByName resolves a nanoflow qualified name to its ID.
func (pb *pageBuilder) resolveNanoflowByName(nfName string) (model.ID, error) {
	if pb.execCache != nil && pb.execCache.createdNanoflows != nil {
		if info, ok := pb.execCache.createdNanoflows[nfName]; ok {
			return info.ID, nil
		}
	}

	parts := strings.Split(nfName, ".")
	var moduleName, name string
	if len(parts) >= 2 {
		moduleName = parts[0]
		name = parts[1]
	} else {
		moduleName = pb.moduleName
		name = nfName
	}

	if pb.nanoflowsRepo == nil {
		return "", mdlerrors.NewNotFound("nanoflow", nfName)
	}
	nanoflows, err := pb.nanoflowsRepo.List("")
	if err != nil {
		return "", mdlerrors.NewBackend("list nanoflows", err)
	}

	h, err := pb.getHierarchy()
	if err != nil {
		return "", err
	}

	for _, nf := range nanoflows {
		if nf == nil {
			continue
		}
		containerID, _ := pb.microflowsRepo.GetContainerUUID(model.ID(nf.ID()))
		modName := h.GetModuleName(h.FindModuleID(containerID))
		if modName == moduleName && nf.Name() == name {
			return model.ID(nf.ID()), nil
		}
	}

	return "", mdlerrors.NewNotFound("nanoflow", nfName)
}
