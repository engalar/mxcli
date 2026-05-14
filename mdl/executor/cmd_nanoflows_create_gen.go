// SPDX-License-Identifier: Apache-2.0

// Stage 3.2.5c — gen-typed CREATE NANOFLOW write path.
//
// Persists nanoflows through ctx.Nanoflows (the modelsdk-native
// repository wired in Stage 2) instead of ctx.Backend.CreateNanoflow,
// so the BSON encode goes through the gen codec — no sdk/microflows
// types touch the write path.
//
// Scope of this commit:
//   - Header construction (Name, Documentation, Excluded, allowed
//     roles, parameter list, return type) is fully gen-native.
//   - Trivial bodies — `begin end;` with at most a single bare
//     `return;` / `return $expr;` — are emitted as a gen Start → End
//     SequenceFlow pair. This covers nanoflow-skeleton workflows used
//     by `mxcli init` templates and the test surface here.
//   - Compound bodies (if/loop/calls/activities) are deferred to Stage
//     3.2.3 (the `flowBuilder` family rewrite). Encountering one
//     returns a clear error directing the caller to the legacy
//     executor entry until that lands.
//
// The dispatch layer (Stage 3.2.6) will route execCreateNanoflow to
// execCreateNanoflowGen and delete the sdk-typed original — at which
// point Stage 3.2.3 must already be merged so compound bodies don't
// regress.

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

// execCreateNanoflowGen handles CREATE NANOFLOW statements via the
// gen-typed write path. Behaviour parity with execCreateNanoflow for
// the cases it supports (header-only / empty-body); rejects compound
// bodies with an actionable error pending Stage 3.2.3.
func execCreateNanoflowGen(ctx *ExecContext, s *ast.CreateNanoflowStmt) error {
	if !ctx.ConnectedForWrite() {
		return mdlerrors.NewNotConnectedWrite()
	}
	if ctx.Nanoflows == nil {
		return mdlerrors.NewBackend("nanoflows repo unavailable", nil)
	}

	// ── Validation ───────────────────────────────────────────
	if strings.TrimSpace(s.Name.Name) == "" {
		return mdlerrors.NewValidation("nanoflow name must not be empty")
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

	// Nanoflow-specific body restrictions (no DB / Java / workflow / etc).
	// Pure AST check — no SDK types involved.
	if errMsg := validateNanoflow(qualifiedName, s.Body, s.ReturnType); errMsg != "" {
		return fmt.Errorf("%s", errMsg)
	}

	// Stage 3.2.3 lifted the trivial-body restriction: compound
	// bodies route through the shared flowBuilderGen / buildFlowGraphGen
	// path below (see "Compound body emission" branch). The
	// validateNanoflow guard above already rejects nanoflow-disallowed
	// statement types before we reach this point.

	// ── Existence + replace handling ──────────────────────────
	var (
		existing             *genMf.Nanoflow
		existingContainerID  model.ID
		existingAllowedRoles []string
		preserveAllowedRoles bool
	)
	all, err := ctx.Nanoflows.List("")
	if err != nil {
		return mdlerrors.NewBackend("check existing nanoflows", err)
	}
	h, _ := getHierarchy(ctx)
	for _, nf := range all {
		if nf == nil {
			continue
		}
		modName := genFlowContainerModule(ctx, h, model.ID(nf.ID()))
		if modName == s.Name.Module && nf.Name() == s.Name.Name {
			if !s.CreateOrModify {
				return mdlerrors.NewAlreadyExistsMsg("nanoflow", qualifiedName,
					"nanoflow '"+qualifiedName+"' already exists (use create or modify to overwrite)")
			}
			existing = nf
			if cid, err := ctx.Microflows.GetContainerUUID(model.ID(nf.ID())); err == nil && cid != "" {
				existingContainerID = cid
			}
			existingAllowedRoles = append([]string{}, nf.AllowedModuleRolesQualifiedNames()...)
			preserveAllowedRoles = true
			break
		}
	}

	// Folder-omission preserves the existing container on replace.
	if existing != nil && s.Folder == "" && existingContainerID != "" {
		containerID = existingContainerID
	}

	// Build the gen Nanoflow.
	nf := genMf.NewNanoflow()
	if existing != nil {
		// Reuse the existing element ID so refs and dirty bits don't
		// shift across replace.
		nf.SetID(existing.ID())
	} else if dropped := consumeDroppedNanoflow(ctx, qualifiedName); dropped != nil {
		nf.SetID(element.ID(dropped.ID))
		if s.Folder == "" && dropped.ContainerID != "" {
			containerID = dropped.ContainerID
		}
		if len(dropped.AllowedRoles) > 0 {
			// Convert sdk role IDs back to qualified names. The dropped
			// info captured raw model.IDs; the gen path stores qualified
			// names. We can't recover names from IDs here without a
			// security-roles query — fall through to module defaults.
			//
			// This is a known gap that 3.2.3 / 3.2.6 will close when the
			// drop-tracker switches to recording qualified names too.
			preserveAllowedRoles = false
		}
	}

	nf.SetName(s.Name.Name)
	nf.SetDocumentation(s.Documentation)
	nf.SetExcluded(s.Excluded)

	if preserveAllowedRoles {
		nf.SetAllowedModuleRolesQualifiedNames(existingAllowedRoles)
	} else {
		nf.SetAllowedModuleRolesQualifiedNames(defaultDocumentAccessRoleQNames(ctx, module))
	}

	// Parameter elements — emitted into the ObjectCollection so the
	// gen describer's walk over the collection finds them
	// (MicroflowParameter / NanoflowParameter tags).
	paramElements := make([]*genMf.MicroflowParameter, 0, len(s.Parameters))
	for _, p := range s.Parameters {
		param := genMf.NewMicroflowParameter()
		param.SetName(p.Name)
		if t := paramASTToShortType(p.Type); t != "" {
			param.SetType(t)
		}
		paramElements = append(paramElements, param)
	}

	// Body emission — two paths:
	//
	//  - Trivial (empty / bare `return`): inline Start→End pair so the
	//    output exactly matches the original Stage 3.2.5c shape that
	//    fixtures + roundtrip tests assert against.
	//
	//  - Compound (any statement that's not a bare return): route
	//    through flowBuilderGen.buildFlowGraphGen — Stage 3.2.3 shipped
	//    the full flow-graph builder and made compound nanoflow bodies
	//    feasible without legacy.
	var oc *genMf.MicroflowObjectCollection
	if isTrivialNanoflowBody(s.Body) {
		oc = genMf.NewMicroflowObjectCollection()
		for _, param := range paramElements {
			oc.AddObjects(param)
		}
		start := genMf.NewStartEvent()
		assignFreshID(start)
		end := genMf.NewEndEvent()
		assignFreshID(end)
		if rv := returnExpressionFromBody(s.Body); rv != "" {
			end.SetReturnValue(rv)
		}
		oc.AddObjects(start)
		oc.AddObjects(end)

		flow := genMf.NewSequenceFlow()
		assignFreshID(flow)
		flow.SetOriginID(start.ID())
		flow.SetDestinationID(end.ID())
		nf.AddFlows(flow)
	} else {
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
			isNanoflow:     true, // EH defaults to "Abort" instead of "Rollback"
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

		oc = fb.buildFlowGraphGen(s.Body, s.ReturnType)

		// Surface validation errors collected during build.
		if errs := fb.GetErrors(); len(errs) > 0 {
			return mdlerrors.NewValidationf(
				"nanoflow '%s' has validation errors:\n  - %s",
				qualifiedName, strings.Join(errs, "\n  - "))
		}

		// Inject parameters alongside body activities (gen describer
		// walks the collection looking for both kinds).
		for _, param := range paramElements {
			oc.AddObjects(param)
		}

		// Append all sequence flows + annotation flows onto the
		// nanoflow's Flows array (top-level placement, not nested in
		// the ObjectCollection — matches the microflow path).
		for _, flow := range fb.flows {
			nf.AddFlows(flow)
		}
		for _, af := range fb.annotationFlows {
			nf.AddFlows(af)
		}
	}

	nf.SetObjectCollection(oc)

	// Return type — primitive shorthand only at this stage; entity /
	// list-of-entity returns will be wired alongside the rest of
	// flowBuilder in 3.2.3.
	if s.ReturnType != nil {
		if t := paramASTToShortType(s.ReturnType.Type); t != "" {
			nf.SetReturnType(t)
		}
	}

	// ── Persist ──────────────────────────────────────────────
	if existing != nil {
		if err := ctx.Nanoflows.Update(nf); err != nil {
			return mdlerrors.NewBackend("update nanoflow", err)
		}
		fmt.Fprintf(ctx.Output, "Replaced nanoflow: %s\n", qualifiedName)
	} else {
		if err := ctx.Nanoflows.Create(string(containerID), "Documents", nf); err != nil {
			return mdlerrors.NewBackend("create nanoflow", err)
		}
		fmt.Fprintf(ctx.Output, "Created nanoflow: %s\n", qualifiedName)
	}

	// Track for dispatch-layer tests + invalidate caches so subsequent
	// reads see the new unit.
	returnEntityName := ""
	if s.ReturnType != nil && s.ReturnType.Type.EntityRef != nil {
		returnEntityName = s.ReturnType.Type.EntityRef.Module + "." + s.ReturnType.Type.EntityRef.Name
	}
	ctx.trackCreatedNanoflow(s.Name.Module, s.Name.Name, model.ID(nf.ID()), containerID, returnEntityName)

	invalidateHierarchy(ctx)
	invalidateMicroflowsCache(ctx)
	return nil
}

// isTrivialNanoflowBody reports whether the body is one of:
//   - empty (no statements)
//   - a single bare `return;` or `return $expr;`
//
// Anything else falls outside Stage 3.2.5c's gen-write scope.
func isTrivialNanoflowBody(body []ast.MicroflowStatement) bool {
	switch len(body) {
	case 0:
		return true
	case 1:
		_, isReturn := body[0].(*ast.ReturnStmt)
		return isReturn
	}
	return false
}

// returnExpressionFromBody extracts the return expression from a
// trivial body (the second case of isTrivialNanoflowBody). Returns ""
// for empty bodies / bare `return;`.
//
// ReturnStmt.Value is an ast.Expression interface; we convert it to a
// raw expression string via the same helper microflow code paths use
// when emitting return values to BSON.
func returnExpressionFromBody(body []ast.MicroflowStatement) string {
	if len(body) != 1 {
		return ""
	}
	rs, ok := body[0].(*ast.ReturnStmt)
	if !ok || rs.Value == nil {
		return ""
	}
	return strings.TrimSpace(expressionToStringForGen(rs.Value))
}

// expressionToStringForGen renders an ast.Expression as the literal
// MDL/BSON string the runtime expects. We can't use the legacy
// `expressionToString` helper here because it lives on flowBuilder and
// needs the full type-resolution context (which Stage 3.2.3 will
// supply in the gen flowBuilder). Until then, fall back to the
// expression's String() form when it implements Stringer, or "" so
// the return rendering stays harmless.
func expressionToStringForGen(e ast.Expression) string {
	if e == nil {
		return ""
	}
	if s, ok := e.(interface{ String() string }); ok {
		return s.String()
	}
	return ""
}

// paramASTToShortType maps an ast.DataType to the gen short type tag
// (the string `MicroflowParameter.Type` / `Microflow.ReturnType`
// stores). For primitives this is the type name. For entity / list /
// enum types we return "" — those need DataType part elements that
// 3.2.3's flowBuilder rewrite will introduce.
func paramASTToShortType(t ast.DataType) string {
	switch t.Kind {
	case ast.TypeBoolean:
		return "Boolean"
	case ast.TypeInteger:
		return "Integer"
	case ast.TypeLong:
		return "Long"
	case ast.TypeDecimal:
		return "Decimal"
	case ast.TypeString:
		return "String"
	case ast.TypeDateTime:
		return "DateTime"
	case ast.TypeBinary:
		return "Binary"
	}
	return ""
}

// defaultDocumentAccessRoleQNames returns the module's default
// allowed-role qualified names. defaultDocumentAccessRoles already
// returns model.IDs that are themselves qualified-name strings
// (`Module.RoleName`), so we just convert the slice. Returns nil for
// modules with no auto-role configured.
func defaultDocumentAccessRoleQNames(ctx *ExecContext, module *model.Module) []string {
	ids := defaultDocumentAccessRoles(ctx, module)
	if len(ids) == 0 {
		return nil
	}
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		if s := string(id); s != "" {
			out = append(out, s)
		}
	}
	return out
}
