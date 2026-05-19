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
	"fmt"
	"strings"

	"github.com/mendixlabs/mxcli/mdl/ast"
	mdlerrors "github.com/mendixlabs/mxcli/mdl/errors"
	"github.com/mendixlabs/mxcli/model"
	"github.com/mendixlabs/mxcli/modelsdk/element"
	genMf "github.com/mendixlabs/mxcli/modelsdk/gen/microflows"
)

// execCreateMicroflowGen handles CREATE MICROFLOW via the gen-typed
// write path.
func execCreateMicroflowGen(ctx *ExecContext, s *ast.CreateMicroflowStmt) error {
	if !ctx.ConnectedForWrite() {
		return mdlerrors.NewNotConnectedWrite()
	}
	if ctx.Microflows == nil {
		return mdlerrors.NewBackend("microflows repo unavailable", nil)
	}

	// ── Validation ───────────────────────────────────────────
	if strings.TrimSpace(s.Name.Name) == "" {
		return mdlerrors.NewValidation("microflow name must not be empty")
	}

	module, err := findOrCreateModule(ctx, s.Name.Module)
	if err != nil {
		return err
	}

	containerID := module.ID
	if s.Folder != "" {
		folderID, err := resolveFolder(ctx, module.ID, s.Folder)
		if err != nil {
			return mdlerrors.NewBackend("resolve folder "+s.Folder, err)
		}
		containerID = folderID
	}

	qualifiedName := s.Name.Module + "." + s.Name.Name

	// ── Existence + replace handling ──────────────────────────
	var (
		existing             *genMf.Microflow
		existingContainerID  model.ID
		existingAllowedRoles []string
		preserveAllowedRoles bool
	)
	all, err := ctx.Microflows.ListAll()
	if err != nil {
		return mdlerrors.NewBackend("check existing microflows", err)
	}
	h, _ := getHierarchy(ctx)
	for _, mf := range all {
		if mf == nil {
			continue
		}
		modName := genFlowContainerModule(ctx, h, model.ID(mf.ID()))
		if modName == s.Name.Module && mf.Name() == s.Name.Name {
			if !s.CreateOrModify {
				return mdlerrors.NewAlreadyExistsMsg("microflow", qualifiedName,
					"microflow '"+qualifiedName+"' already exists (use create or modify to overwrite)")
			}
			existing = mf
			if cid, err := ctx.Microflows.GetContainerUUID(model.ID(mf.ID())); err == nil && cid != "" {
				existingContainerID = cid
			}
			existingAllowedRoles = append([]string{}, mf.AllowedModuleRolesQualifiedNames()...)
			preserveAllowedRoles = true
			break
		}
	}

	// Folder-omission preserves the existing container on replace.
	if existing != nil && s.Folder == "" && existingContainerID != "" {
		containerID = existingContainerID
	}

	// ── Build the gen Microflow ──────────────────────────────
	mf := genMf.NewMicroflow()
	if existing != nil {
		// Reuse existing element ID so refs and dirty bits don't
		// shift across replace.
		mf.SetID(existing.ID())
	} else if dropped := consumeDroppedMicroflow(ctx, qualifiedName); dropped != nil {
		mf.SetID(element.ID(dropped.ID))
		if s.Folder == "" && dropped.ContainerID != "" {
			containerID = dropped.ContainerID
		}
		// Allowed role preservation across drop+create — same gap
		// as nanoflow path: gen stores qualified names, dropped
		// info captured raw model.IDs. Fall through to defaults.
		preserveAllowedRoles = false
	}

	mf.SetName(s.Name.Name)
	mf.SetDocumentation(s.Documentation)
	mf.SetExcluded(s.Excluded)
	mf.SetAllowConcurrentExecution(true)

	if preserveAllowedRoles {
		mf.SetAllowedModuleRolesQualifiedNames(existingAllowedRoles)
	} else {
		mf.SetAllowedModuleRolesQualifiedNames(defaultDocumentAccessRoleQNames(ctx, module))
	}

	// Return type — set both the shorthand string (ReturnType) and the
	// nested DataType element (MicroflowReturnType) that mx check requires
	// for caller return-variable type resolution.
	if s.ReturnType != nil {
		if t := paramASTToShortType(s.ReturnType.Type); t != "" {
			mf.SetReturnType(t)
		}
		if dt := convertASTToGenDataType(s.ReturnType.Type); dt != nil {
			mf.SetMicroflowReturnType(dt)
		}
		if s.ReturnType.Variable != "" {
			mf.SetReturnVariableName(s.ReturnType.Variable)
		}
	}

	// ── Build the flow graph ─────────────────────────────────
	hierarchy, _ := getHierarchy(ctx)
	restServices, _ := loadRestServices(ctx)

	fb := &flowBuilderGen{
		posX:           200,
		posY:           200,
		baseY:          200,
		spacing:        HorizontalSpacing,
		varTypes:       map[string]string{},
		declaredVars:   map[string]string{},
		measurer:       &layoutMeasurer{varTypes: map[string]string{}},
		backend:        ctx.Backend,
		microflowsRepo: ctx.Microflows,
		nanoflowsRepo:  ctx.Nanoflows,
		hierarchy:      hierarchy,
		restServices:   restServices,
	}

	// Initialise variable types from parameters so body statements
	// can resolve member access on entity-typed params.
	for _, p := range s.Parameters {
		if p.Type.EntityRef != nil {
			entityQN := p.Type.EntityRef.Module + "." + p.Type.EntityRef.Name
			if p.Type.Kind == ast.TypeListOf {
				fb.varTypes[p.Name] = "List of " + entityQN
			} else {
				fb.varTypes[p.Name] = entityQN
			}
		} else {
			fb.declaredVars[p.Name] = p.Type.Kind.String()
		}
	}

	// Build the flow graph (StartEvent + body activities + EndEvent
	// + sequence flows). The collection ends up on the microflow's
	// ObjectCollection; flows go onto the microflow's Flows array
	// (top-level, not nested in the collection).
	oc := fb.buildFlowGraphGen(s.Body, s.ReturnType)

	// Surface collected validation errors.
	if errs := fb.GetErrors(); len(errs) > 0 {
		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("microflow '%s' has validation errors:\n", qualifiedName))
		for _, e := range errs {
			sb.WriteString(fmt.Sprintf("  - %s\n", e))
		}
		return fmt.Errorf("%s", sb.String())
	}

	// Inject parameters into the ObjectCollection so they round-trip
	// alongside the body activities (gen describer walks the collection
	// looking for MicroflowParameter elements).
	for _, p := range s.Parameters {
		param := genMf.NewMicroflowParameter()
		assignFreshID(param)
		param.SetName(p.Name)
		// Studio Pro stores the type exclusively in VariableType (a
		// DataTypes child element). The Type() string field is never set.
		if dt := convertASTToGenDataType(p.Type); dt != nil {
			param.SetParameterType(dt)
		}
		oc.AddObjects(param)
	}

	mf.SetObjectCollection(oc)

	// Append all sequence flows onto the microflow's Flows array.
	for _, flow := range fb.flows {
		mf.AddFlows(flow)
	}
	for _, af := range fb.annotationFlows {
		mf.AddFlows(af)
	}

	// ── Persist ──────────────────────────────────────────────
	if existing != nil {
		if err := ctx.Microflows.Update(mf); err != nil {
			return mdlerrors.NewBackend("update microflow", err)
		}
		fmt.Fprintf(ctx.Output, "Replaced microflow: %s\n", qualifiedName)
	} else {
		if err := ctx.Microflows.Create(string(containerID), "Documents", mf); err != nil {
			return mdlerrors.NewBackend("create microflow", err)
		}
		fmt.Fprintf(ctx.Output, "Created microflow: %s\n", qualifiedName)
	}

	// Track for downstream lookups + invalidate caches so subsequent
	// reads see the new unit.
	returnEntityName := ""
	if s.ReturnType != nil && s.ReturnType.Type.EntityRef != nil {
		returnEntityName = s.ReturnType.Type.EntityRef.Module + "." + s.ReturnType.Type.EntityRef.Name
	}
	ctx.trackCreatedMicroflow(s.Name.Module, s.Name.Name, model.ID(mf.ID()), containerID, returnEntityName)

	invalidateHierarchy(ctx)
	invalidateMicroflowsCache(ctx)
	return nil
}
