// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"context"
	"fmt"
	"strings"

	"github.com/mendixlabs/mxcli/mdl/ast"
	mdlerrors "github.com/mendixlabs/mxcli/mdl/errors"
	"github.com/mendixlabs/mxcli/mdl/types"
	"github.com/mendixlabs/mxcli/model"
	"github.com/mendixlabs/mxcli/modelsdk/element"
	genCA "github.com/mendixlabs/mxcli/modelsdk/gen/codeactions"
	genJA "github.com/mendixlabs/mxcli/modelsdk/gen/javaactions"
	genJSA "github.com/mendixlabs/mxcli/modelsdk/gen/javascriptactions"
)

// ─────────────────────────────────────────────────────────────────────
// D3 — execDropJavaActionGen
// ─────────────────────────────────────────────────────────────────────

// execDropJavaActionGen handles DROP JAVA ACTION using gen-typed reads
// from listJavaActionsWithContainerGen and the gen-aware repo path.
// Mirrors execDropJavaAction (cmd_javaactions.go:249) but consumes
// container UUIDs from the cache helper rather than from a sdk-typed
// ContainerID field that gen objects don't carry.
func execDropJavaActionGen(ctx *ExecContext, s *ast.DropJavaActionStmt) error {
	return execDropJavaActionGenFn(ctx, s, execContextToDeps(ctx))
}

// execDropJavaActionGenFn is the HandlerDeps version of execDropJavaActionGen.
func execDropJavaActionGenFn(ctx context.Context, s *ast.DropJavaActionStmt, deps *HandlerDeps) error {
	if deps.ConnectionManager == nil || !deps.ConnectionManager.IsConnected() {
		return mdlerrors.NewNotConnectedWrite()
	}
	h, err := NewContainerHierarchyFromRoles(deps.ModuleLister, deps.MetadataReader, deps.FolderManager)
	if err != nil {
		return mdlerrors.NewBackend("build hierarchy", err)
	}
	ectx := phase3d2bNewExecContext(ctx, deps)
	pairs, err := listJavaActionsWithContainerGen(ectx)
	if err != nil {
		return mdlerrors.NewBackend("list java actions", err)
	}
	for _, p := range pairs {
		if p.Elem == nil {
			continue
		}
		modID := h.FindModuleID(model.ID(p.ContainerID))
		modName := h.GetModuleName(modID)
		if modName != s.Name.Module || p.Elem.Name() != s.Name.Name {
			continue
		}
		// Phase D Delete is not yet wired through the gen repo (Phase
		// D5 fills the writer); for now route through the legacy
		// backend Delete which targets the same MPR Unit row by ID.
		if err := deps.JavaActionWriter.DeleteJavaAction(model.ID(p.Elem.ID())); err != nil {
			return mdlerrors.NewBackend("delete java action", err)
		}
		if err := deps.JavaActionWriter.DeleteJavaSourceFile(modName, p.Elem.Name()); err != nil {
			return mdlerrors.NewBackend("delete java source file", err)
		}
		invalidateJavaActionsCacheFn(deps)
		fmt.Fprintf(deps.Output, "Dropped java action: %s.%s\n", s.Name.Module, s.Name.Name)
		return nil
	}
	return mdlerrors.NewNotFound("java action", s.Name.Module+"."+s.Name.Name)
}

// invalidateJavaActionsCacheFn is the HandlerDeps version of invalidateJavaActionsCache.
func invalidateJavaActionsCacheFn(deps *HandlerDeps) {
	if deps.Cache != nil {
		deps.Cache.javaActionsWithContainerGen = nil
	}
}

// ─────────────────────────────────────────────────────────────────────
// A6 — execCreateJavaScriptAction
// ─────────────────────────────────────────────────────────────────────

// execCreateJavaScriptAction handles CREATE [OR MODIFY] JAVASCRIPT ACTION.
// Creates the BSON unit and JavaScript source file from scratch, mirroring
// execCreateJavaActionGen — no Studio Pro pre-requisite.
func execCreateJavaScriptAction(ctx *ExecContext, s *ast.CreateJavaScriptActionStmt) error {
	return execCreateJavaScriptActionFn(ctx, s, execContextToDeps(ctx))
}

// execCreateJavaScriptActionFn is the HandlerDeps version of execCreateJavaScriptAction.
func execCreateJavaScriptActionFn(ctx context.Context, s *ast.CreateJavaScriptActionStmt, deps *HandlerDeps) error {
	if deps.ConnectionManager == nil || !deps.ConnectionManager.IsConnected() {
		return mdlerrors.NewNotConnectedWrite()
	}

	h, err := NewContainerHierarchyFromRoles(deps.ModuleLister, deps.MetadataReader, deps.FolderManager)
	if err != nil {
		return mdlerrors.NewBackend("build hierarchy", err)
	}

	modules, err := deps.ModuleLister.ListModules()
	if err != nil {
		return mdlerrors.NewBackend("get modules", err)
	}
	var (
		containerID model.ID
		moduleName  string
	)
	for _, mod := range modules {
		if mod.Name == s.Name.Module {
			containerID = mod.ID
			moduleName = mod.Name
			break
		}
	}
	if containerID == "" {
		return mdlerrors.NewNotFound("module", s.Name.Module)
	}

	ectx := phase3d2bNewExecContext(ctx, deps)
	pairs, err := listJavaScriptActionsWithContainerGen(ectx)
	if err != nil {
		return mdlerrors.NewBackend("list javascript actions", err)
	}
	var existingJSA *genJSA.JavaScriptAction
	for _, p := range pairs {
		if p.Elem == nil {
			continue
		}
		modID := h.FindModuleID(modelIDFromElementID(p.ContainerID))
		modName := h.GetModuleName(modID)
		if modName == s.Name.Module && p.Elem.Name() == s.Name.Name {
			if !s.CreateOrModify {
				return mdlerrors.NewAlreadyExists("javascript action", s.Name.Module+"."+s.Name.Name)
			}
			existingJSA = p.Elem
			break
		}
	}

	jsa := genJSA.NewJavaScriptAction()
	if existingJSA != nil {
		jsa.SetID(existingJSA.ID())
	} else {
		jsa.SetID(element.ID(types.GenerateID()))
	}
	jsa.SetName(s.Name.Name)
	jsa.SetDocumentation(s.Documentation)
	jsa.SetExportLevel("Public")
	jsa.SetExcluded(false)
	jsa.SetActionDefaultReturnName("ReturnValue")
	if s.Platform != "" {
		jsa.SetPlatform(s.Platform)
	}

	existingParams := make(map[string]element.ID)
	if existingJSA != nil {
		for _, p := range existingJSA.ParametersItems() {
			if pp, ok := p.(*genJSA.JavaScriptActionParameter); ok {
				existingParams[pp.Name()] = pp.ID()
			}
		}
	}
	for _, param := range s.Parameters {
		jsaParam := genJSA.NewJavaScriptActionParameter()
		if id, ok := existingParams[param.Name]; ok {
			jsaParam.SetID(id)
		} else {
			jsaParam.SetID(element.ID(types.GenerateID()))
		}
		jsaParam.SetName(strings.ToLower(param.Name[:1]) + param.Name[1:])
		jsaParam.SetCategory("")
		jsaParam.SetDescription("")
		jsaParam.SetIsRequired(param.IsRequired)
		innerType := astDataTypeToJavaActionParamTypeGen(param.Type, nil)
		bpt := genCA.NewBasicParameterType()
		bpt.SetID(element.ID(types.GenerateID()))
		bpt.SetType(innerType)
		jsaParam.SetActionParameterType(bpt)
		jsa.AddParameters(jsaParam)
	}

	if s.ReturnType.Kind != ast.TypeVoid {
		rt := astDataTypeToJavaActionReturnTypeGen(s.ReturnType, nil)
		jsa.SetActionReturnType(rt)
		jsa.SetJavaReturnType(rt)
	}

	if existingJSA != nil {
		if err := deps.JavaScriptActionWriter.UpdateJavaScriptActionGen(jsa); err != nil {
			return mdlerrors.NewBackend("update javascript action", err)
		}
	} else {
		if err := deps.JavaScriptActionWriter.CreateJavaScriptActionGen(string(containerID), "Documents", jsa); err != nil {
			return mdlerrors.NewBackend("create javascript action", err)
		}
	}

	if s.UserCode != "" || s.Imports != "" || s.ExtraCode != "" {
		paramNames := make([]string, len(s.Parameters))
		for i, p := range s.Parameters {
			if p.Name == "" {
				paramNames[i] = fmt.Sprintf("param%d", i)
			} else {
				paramNames[i] = strings.ToLower(p.Name[:1]) + p.Name[1:]
			}
		}
		if err := writeJavaScriptActionSource(deps.MprPath, moduleName, s.Name.Name,
			s.Imports, s.ExtraCode, s.UserCode, paramNames); err != nil {
			return mdlerrors.NewBackend("write javascript source file", err)
		}
	}

	invalidateJavaScriptActionsCacheFn(deps)
	invalidateHierarchyFn(deps)

	if existingJSA != nil {
		fmt.Fprintf(deps.Output, "Modified javascript action: %s.%s\n", s.Name.Module, s.Name.Name)
	} else {
		fmt.Fprintf(deps.Output, "Created javascript action: %s.%s\n", s.Name.Module, s.Name.Name)
	}
	return nil
}

// invalidateJavaScriptActionsCacheFn is the HandlerDeps version of invalidateJavaScriptActionsCache.
func invalidateJavaScriptActionsCacheFn(deps *HandlerDeps) {
	if deps.Cache != nil {
		deps.Cache.javaScriptActionsWithContainerGen = nil
	}
}

// ─────────────────────────────────────────────────────────────────────
// D2 — execCreateJavaActionGen
// ─────────────────────────────────────────────────────────────────────

// execCreateJavaActionGen handles CREATE JAVA ACTION via the gen-typed
// write path. Mirrors execCreateJavaAction (cmd_javaactions.go:285) but:
//
//   - Existence check goes through listJavaActionsWithContainerGen
//     (cache-aware, gen-typed) instead of ctx.Backend.ListJavaActions
//     (sdk-typed).
//   - The JavaAction object is built via genJA.New* constructors and
//     SetX accessors. Parameter / return types come from the D1
//     converters (astDataTypeToJavaActionParamTypeGen /
//     astDataTypeToJavaActionReturnTypeGen).
//   - Persistence routes through ctx.Backend.{CreateJavaActionGen,
//     UpdateJavaActionGen} (the gen-aware sibling of CreateJavaAction).
//   - Java source file IO routes through ctx.Backend.WriteJavaSourceFileGen.
//     This currently returns "not implemented (Phase D6)"; we tolerate
//     that error so the BSON unit is still created — D6 will replace
//     the stub with a real generator and source files will start
//     materialising automatically.
//
// schema gap: per the dual-accessor note (3) at top of file, Studio
// Pro fixtures populate the legacy Parameters/ReturnType BSON keys.
// For NEW elements emitted by the gen path, we use the canonical
// ActionParameters / ActionReturnType setters; downstream readers
// already dispatch on both flavours via javaActionParametersOf /
// javaActionReturnTypeElement (no schema-gap mismatch on roundtrip).
func execCreateJavaActionGen(ctx *ExecContext, s *ast.CreateJavaActionStmt) error {
	return execCreateJavaActionGenFn(ctx, s, execContextToDeps(ctx))
}

// execCreateJavaActionGenFn is the HandlerDeps version of execCreateJavaActionGen.
func execCreateJavaActionGenFn(ctx context.Context, s *ast.CreateJavaActionStmt, deps *HandlerDeps) error {
	if deps.ConnectionManager == nil || !deps.ConnectionManager.IsConnected() {
		return mdlerrors.NewNotConnectedWrite()
	}

	h, err := NewContainerHierarchyFromRoles(deps.ModuleLister, deps.MetadataReader, deps.FolderManager)
	if err != nil {
		return mdlerrors.NewBackend("build hierarchy", err)
	}

	modules, err := deps.ModuleLister.ListModules()
	if err != nil {
		return mdlerrors.NewBackend("get modules", err)
	}
	var (
		containerID model.ID
		moduleName  string
	)
	for _, mod := range modules {
		if mod.Name == s.Name.Module {
			containerID = mod.ID
			moduleName = mod.Name
			break
		}
	}
	if containerID == "" {
		return mdlerrors.NewNotFound("module", s.Name.Module)
	}

	ectx := phase3d2bNewExecContext(ctx, deps)
	pairs, err := listJavaActionsWithContainerGen(ectx)
	if err != nil {
		return mdlerrors.NewBackend("list java actions", err)
	}
	var existingJA *genJA.JavaAction
	for _, p := range pairs {
		if p.Elem == nil {
			continue
		}
		modID := h.FindModuleID(model.ID(p.ContainerID))
		modName := h.GetModuleName(modID)
		if modName == s.Name.Module && p.Elem.Name() == s.Name.Name {
			if !s.CreateOrModify {
				return mdlerrors.NewAlreadyExists("java action", s.Name.Module+"."+s.Name.Name)
			}
			existingJA = p.Elem
			break
		}
	}

	ja := genJA.NewJavaAction()
	if existingJA != nil {
		ja.SetID(existingJA.ID())
	} else {
		ja.SetID(element.ID(types.GenerateID()))
	}
	ja.SetName(s.Name.Name)
	ja.SetDocumentation(s.Documentation)
	ja.SetExportLevel("Public")
	ja.SetExcluded(false)
	ja.SetActionDefaultReturnName("ReturnValueName")

	typeParamIDs := make(map[string]element.ID, len(s.TypeParameters))
	for _, tpName := range s.TypeParameters {
		tp := genJA.NewTypeParameter()
		tp.SetID(element.ID(types.GenerateID()))
		tp.SetName(tpName)
		ja.AddTypeParameters(tp)
		typeParamIDs[tpName] = tp.ID()
	}
	ja.ForceWriteTypeParameters()

	typeParamNames := make(map[string]bool, len(s.TypeParameters))
	for _, tpName := range s.TypeParameters {
		typeParamNames[tpName] = true
	}

	for _, param := range s.Parameters {
		jaParam := genJA.NewJavaActionParameter()
		jaParam.SetID(element.ID(types.GenerateID()))
		jaParam.SetCategory("")
		jaParam.SetDescription("")
		jaParam.SetName(param.Name)
		jaParam.SetIsRequired(param.IsRequired)

		var innerType element.Element
		switch {
		case param.Type.Kind == ast.TypeEntityTypeParam:
			innerType = astDataTypeToJavaActionParamTypeGen(param.Type, typeParamIDs)
		case isTypeParamRef(param.Type, typeParamNames):
			tpName := getTypeParamRefName(param.Type)
			pet := genCA.NewParameterizedEntityType()
			if id, ok := typeParamIDs[tpName]; ok {
				pet.SetTypeParameterID(id)
			}
			innerType = pet
		default:
			innerType = astDataTypeToJavaActionParamTypeGen(param.Type, typeParamIDs)
		}
		bpt := genCA.NewBasicParameterType()
		bpt.SetID(element.ID(types.GenerateID()))
		bpt.SetType(innerType)
		jaParam.SetParameterType(bpt)
		ja.AddParameters(jaParam)
	}

	if isTypeParamRef(s.ReturnType, typeParamNames) {
		tpName := getTypeParamRefName(s.ReturnType)
		pet := genCA.NewParameterizedEntityType()
		if id, ok := typeParamIDs[tpName]; ok {
			pet.SetTypeParameterID(id)
		}
		ja.SetJavaReturnType(pet)
	} else {
		ja.SetJavaReturnType(astDataTypeToJavaActionReturnTypeGen(s.ReturnType, typeParamIDs))
	}

	if s.ExposedCaption != "" {
		mai := genCA.NewMicroflowActionInfo()
		mai.SetCaption(s.ExposedCaption)
		mai.SetCategory(s.ExposedCategory)
		ja.SetMicroflowActionInfo(mai)
	}

	if existingJA != nil {
		if err := deps.JavaActionWriter.UpdateJavaActionGen(ja); err != nil {
			return mdlerrors.NewBackend("update java action", err)
		}
	} else {
		if err := deps.JavaActionWriter.CreateJavaActionGen(string(containerID), "Documents", ja); err != nil {
			return mdlerrors.NewBackend("create java action", err)
		}
	}

	if s.JavaCode != "" || len(s.Imports) > 0 || s.ExtraCode != "" {
		if strings.Contains(s.JavaCode, "executeAction") {
			return mdlerrors.NewValidationf(
				"code block must contain only the method body, not the method signature\n" +
					"  hint: write only the statements inside executeAction(), e.g.:\n" +
					"    code $$\n" +
					"      return this.Input.trim();\n" +
					"    $$")
		}
		params := make([]*genJA.JavaActionParameter, 0, len(ja.ActionParametersItems()))
		for _, p := range ja.ActionParametersItems() {
			if pp, ok := p.(*genJA.JavaActionParameter); ok {
				params = append(params, pp)
			}
		}
		if werr := deps.JavaActionWriter.WriteJavaSourceFileGen(moduleName, s.Name.Name, s.JavaCode, params, ja.ActionReturnType(), s.Imports, s.ExtraCode); werr != nil {
			if !strings.Contains(werr.Error(), "not implemented") {
				return mdlerrors.NewBackend("write java source file", werr)
			}
		} else if len(s.Imports) > 0 {
			fmt.Fprintf(deps.Output, "note: %d import(s) merged into %s.java\n", len(s.Imports), s.Name.Name)
		}
	}

	invalidateJavaActionsCacheFn(deps)
	invalidateHierarchyFn(deps)

	if existingJA != nil {
		fmt.Fprintf(deps.Output, "Modified java action: %s.%s\n", s.Name.Module, s.Name.Name)
	} else {
		fmt.Fprintf(deps.Output, "Created java action: %s.%s\n", s.Name.Module, s.Name.Name)
	}
	return nil
}
