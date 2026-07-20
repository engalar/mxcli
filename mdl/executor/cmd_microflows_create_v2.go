// SPDX-License-Identifier: Apache-2.0

// Stage 3.2.3.k — gen-typed CREATE MICROFLOW entry point.
//
// Persists microflows through ctx.Microflows (the modelsdk-native
// repository wired in Stage 2) instead of ctx.Backend.CreateMicroflow,
// so the BSON encode goes through the gen codec — no sdk/microflows
// types touch the write path.
//
// Wiring:
//
//   1. Validation: name must be non-empty, must be connected for write.
//   2. Find / create the owning module (legacy findOrCreateModule).
//   3. Resolve folder when supplied.
//   4. Existence check via ctx.Microflows.ListAll() + module-name match
//      — refuses on non-modify create when a name collision exists,
//      reuses the existing element ID for `create or modify`.
//   5. Build the gen Microflow:
//        - SetName, SetDocumentation, SetExcluded
//        - allowed module roles (preserved on replace, defaults
//          otherwise)
//        - parameters as MicroflowParameter elements inside the
//          ObjectCollection
//        - return type (primitive shorthand only at this stage —
//          entity / list-of-entity returns will be wired in a
//          follow-up commit)
//   6. Build the flow graph via flowBuilderGen.buildFlowGraphGen
//      (j) — that emits StartEvent + body activities + EndEvent
//      and threads sequence flows through fb.flows.
//   7. Append flows from fb.flows to mf.AddFlows(...).
//   8. Persist via ctx.Microflows.Create / Update.
//   9. Track the created microflow + invalidate caches.
//
// Stage 3.2.6 (final cleanup) will replace the legacy
// execCreateMicroflow dispatch with this entry and delete the
// sdk/microflows write path.

package executor

import (
	"context"
	"fmt"
	"strings"

	"github.com/mendixlabs/mxcli/mdl/ast"
	mdlerrors "github.com/mendixlabs/mxcli/mdl/errors"
	"github.com/mendixlabs/mxcli/model"
	"github.com/mendixlabs/mxcli/modelsdk/element"
	genDt "github.com/mendixlabs/mxcli/modelsdk/gen/datatypes"
	genMf "github.com/mendixlabs/mxcli/modelsdk/gen/microflows"
	"github.com/mendixlabs/mxcli/modelsdk/version"
)

// execCreateMicroflowGen handles CREATE MICROFLOW via the gen-typed
// write path.

// ExecCreateMicroflowGenFn is the HandlerDeps version of execCreateMicroflowGen.
func ExecCreateMicroflowGenFn(ctx context.Context, s *ast.CreateMicroflowStmt, deps *HandlerDeps) error {
	if deps.ConnectionManager == nil || !deps.ConnectionManager.IsConnected() {
		return mdlerrors.NewNotConnectedWrite()
	}
	if deps.MicroflowRepo == nil {
		return mdlerrors.NewBackend("microflows repo unavailable", nil)
	}

	if strings.TrimSpace(s.Name.Name) == "" {
		return mdlerrors.NewValidation("microflow name must not be empty")
	}

	ectx := newMinimalExecCtx(ctx, deps)
	module, err := findOrCreateModule(ectx, s.Name.Module)
	if err != nil {
		return err
	}

	containerID := module.ID
	if s.Folder != "" {
		h, _ := getHierarchy(ectx)
		folderID, err := resolveFolder(ectx, module.ID, s.Folder, h)
		if err != nil {
			return mdlerrors.NewBackend("resolve folder "+s.Folder, err)
		}
		containerID = folderID
	}

	qualifiedName := s.Name.Module + "." + s.Name.Name

	var (
		existing             *genMf.Microflow
		existingContainerID  model.ID
		existingAllowedRoles []string
		preserveAllowedRoles bool
	)
	items, err := listMicroflowsWithContainerGen(ectx)
	if err != nil {
		return mdlerrors.NewBackend("check existing microflows", err)
	}
	h, _ := getHierarchy(ectx)
	for _, item := range items {
		if item.MF == nil {
			continue
		}
		modName := containerModuleName(h, item.ContainerUUID)
		if modName == s.Name.Module && item.MF.Name() == s.Name.Name {
			if !s.CreateOrModify {
				return mdlerrors.NewAlreadyExistsMsg("microflow", qualifiedName,
					"microflow '"+qualifiedName+"' already exists (use create or modify to overwrite)")
			}
			existing = item.MF
			existingContainerID = item.ContainerUUID
			existingAllowedRoles = append([]string{}, item.MF.AllowedModuleRolesQualifiedNames()...)
			preserveAllowedRoles = true
			break
		}
	}

	var existingParamNames []string
	if existing != nil {
		existingParamNames = microflowParamNamesFromOC(existing)
	}

	if existing != nil && s.Folder == "" && existingContainerID != "" {
		containerID = existingContainerID
	}

	mf := genMf.NewMicroflow()
	if existing != nil {
		mf.SetID(existing.ID())
	} else if dropped := consumeDroppedMicroflow(ectx, qualifiedName); dropped != nil {
		mf.SetID(element.ID(dropped.ID))
		if s.Folder == "" && dropped.ContainerID != "" {
			containerID = dropped.ContainerID
		}
		preserveAllowedRoles = false
	}

	mf.SetName(s.Name.Name)
	mf.SetDocumentation(s.Documentation)
	mf.SetExcluded(s.Excluded)
	mf.SetExportLevel("Hidden")
	mf.SetMarkAsUsed(false)
	mf.SetAllowConcurrentExecution(true)

	if preserveAllowedRoles {
		mf.SetAllowedModuleRolesQualifiedNames(existingAllowedRoles)
	} else {
		mf.SetAllowedModuleRolesQualifiedNames(defaultDocumentAccessRoleQNames(ectx, module))
	}

	if s.ReturnType != nil {
		if dt := convertASTToGenDataType(s.ReturnType.Type); dt != nil {
			mf.SetMicroflowReturnType(dt)
		}
		if s.ReturnType.Variable != "" {
			mf.SetReturnVariableName(s.ReturnType.Variable)
		}
	} else {
		vt := genDt.NewVoidType()
		assignFreshID(vt)
		mf.SetMicroflowReturnType(vt)
	}

	hierarchy, _ := getHierarchy(ectx)
	restServices, _ := loadRestServices(ectx)

	var mendixVer version.Version
	if rpv := deps.ConnectionManager.ProjectVersion(); rpv != nil {
		mendixVer = version.Parse(rpv.ProductVersion)
	}
	fb := &flowBuilderGen{
		posX:              200,
		posY:              200,
		baseY:             200,
		spacing:           HorizontalSpacing,
		varTypes:          map[string]string{},
		declaredVars:      map[string]string{},
		measurer:          &layoutMeasurer{varTypes: map[string]string{}},
		moduleLister:      deps.ModuleLister,
		domainModelReader: deps.DomainModelReader,
		microflowsRepo:    deps.MicroflowRepo,
		nanoflowsRepo:     deps.NanoflowRepo,
		hierarchy:         hierarchy,
		restServices:      restServices,
		version:           mendixVer,
	}

	for _, p := range s.Parameters {
		if ref := paramEntityRef(p.Type); ref != nil {
			entityQN := ref.Module + "." + ref.Name
			if p.Type.Kind == ast.TypeListOf {
				fb.varTypes[p.Name] = "List of " + entityQN
			} else {
				fb.varTypes[p.Name] = entityQN
			}
		} else {
			fb.declaredVars[p.Name] = p.Type.Kind.String()
		}
	}

	if s.ReturnType != nil && s.ReturnType.Variable != "" {
		fb.declaredVars[s.ReturnType.Variable] = "Unknown"
	}

	oc := fb.buildFlowGraphGen(s.Body, s.ReturnType)

	if errs := fb.GetErrors(); len(errs) > 0 {
		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("microflow '%s' has validation errors:\n", qualifiedName))
		for _, e := range errs {
			sb.WriteString(fmt.Sprintf("  - %s\n", e))
		}
		return fmt.Errorf("%s", sb.String())
	}

	for i, p := range s.Parameters {
		param := genMf.NewMicroflowParameter()
		assignFreshID(param)
		param.SetName(p.Name)
		param.SetRelativeMiddlePoint(layoutPos(ParameterStartX+i*ParameterSpacingX, ParameterStartY))
		param.SetSize(layoutSize(ParameterWidth, ParameterHeight))
		paramType := resolveAmbiguousDataType(ectx, deps.ModuleLister, deps.DomainModelReader, p.Type)
		if dt := convertASTToGenDataType(paramType); dt != nil {
			param.SetParameterType(dt)
		}
		oc.AddObjects(param)
	}

	mf.SetObjectCollection(oc)

	for _, flow := range fb.flows {
		mf.AddFlows(flow)
	}
	for _, af := range fb.annotationFlows {
		mf.AddFlows(af)
	}

	if existing != nil && len(existingParamNames) > 0 {
		newParamSet := make(map[string]bool, len(s.Parameters))
		for _, p := range s.Parameters {
			newParamSet[p.Name] = true
		}
		var removed []string
		for _, old := range existingParamNames {
			if !newParamSet[old] {
				removed = append(removed, old)
			}
		}
		warnBrokenCallerRefs(ectx, qualifiedName, removed)
	}

	if existing != nil {
		if err := deps.MicroflowRepo.Update(mf); err != nil {
			return mdlerrors.NewBackend("update microflow", err)
		}
		fmt.Fprintf(deps.Output, "Replaced microflow: %s\n", qualifiedName)
	} else {
		if err := deps.MicroflowRepo.Create(string(containerID), "Documents", mf); err != nil {
			return mdlerrors.NewBackend("create microflow", err)
		}
		fmt.Fprintf(deps.Output, "Created microflow: %s\n", qualifiedName)
	}

	returnEntityName := ""
	if s.ReturnType != nil {
		if ref := paramEntityRef(s.ReturnType.Type); ref != nil && ref.Module != "" {
			returnEntityName = ref.Module + "." + ref.Name
		}
	}
	ectx.trackCreatedMicroflow(s.Name.Module, s.Name.Name, model.ID(mf.ID()), containerID, returnEntityName)

	invalidateHierarchyFn(deps)
	if existing == nil {
		invalidateMicroflowsCacheFn(deps)
	}
	return nil
}
