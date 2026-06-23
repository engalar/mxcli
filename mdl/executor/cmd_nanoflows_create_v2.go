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
	"context"
	"fmt"
	"strings"

	"github.com/mendixlabs/mxcli/mdl/ast"
	mdlerrors "github.com/mendixlabs/mxcli/mdl/errors"
	"github.com/mendixlabs/mxcli/model"
	"github.com/mendixlabs/mxcli/modelsdk/element"
	genMf "github.com/mendixlabs/mxcli/modelsdk/gen/microflows"
	"github.com/mendixlabs/mxcli/modelsdk/version"
)

// execCreateNanoflowGen handles CREATE NANOFLOW statements via the
// gen-typed write path. Behaviour parity with execCreateNanoflow for
// the cases it supports (header-only / empty-body); rejects compound
// bodies with an actionable error pending Stage 3.2.3.
func execCreateNanoflowGen(ctx *ExecContext, s *ast.CreateNanoflowStmt) error {
	return execCreateNanoflowGenFn(ctx, s, execContextToDeps(ctx))
}

// execCreateNanoflowGenFn is the HandlerDeps version of execCreateNanoflowGen.
func execCreateNanoflowGenFn(ctx context.Context, s *ast.CreateNanoflowStmt, deps *HandlerDeps) error {
	if deps.ConnectionManager == nil || !deps.ConnectionManager.IsConnected() {
		return mdlerrors.NewNotConnectedWrite()
	}
	if deps.NanoflowRepo == nil {
		return mdlerrors.NewBackend("nanoflows repo unavailable", nil)
	}

	if strings.TrimSpace(s.Name.Name) == "" {
		return mdlerrors.NewValidation("nanoflow name must not be empty")
	}

	ectx := phase3d2bNewExecContext(ctx, deps)
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

	if errMsg := validateNanoflow(qualifiedName, s.Body, s.ReturnType); errMsg != "" {
		return fmt.Errorf("%s", errMsg)
	}

	var (
		existing             *genMf.Nanoflow
		existingContainerID  model.ID
		existingAllowedRoles []string
		preserveAllowedRoles bool
	)
	all, err := deps.NanoflowRepo.List("")
	if err != nil {
		return mdlerrors.NewBackend("check existing nanoflows", err)
	}
	h, _ := getHierarchy(ectx)
	for _, nf := range all {
		if nf == nil {
			continue
		}
		modName := genFlowContainerModule(ectx, h, model.ID(nf.ID()))
		if modName == s.Name.Module && nf.Name() == s.Name.Name {
			if !s.CreateOrModify {
				return mdlerrors.NewAlreadyExistsMsg("nanoflow", qualifiedName,
					"nanoflow '"+qualifiedName+"' already exists (use create or modify to overwrite)")
			}
			existing = nf
			if deps.MicroflowRepo != nil {
				if cid, err := deps.MicroflowRepo.GetContainerUUID(model.ID(nf.ID())); err == nil && cid != "" {
					existingContainerID = cid
				}
			}
			existingAllowedRoles = append([]string{}, nf.AllowedModuleRolesQualifiedNames()...)
			preserveAllowedRoles = true
			break
		}
	}

	if existing != nil && s.Folder == "" && existingContainerID != "" {
		containerID = existingContainerID
	}

	nf := genMf.NewNanoflow()
	if existing != nil {
		nf.SetID(existing.ID())
	} else if dropped := consumeDroppedNanoflow(ectx, qualifiedName); dropped != nil {
		nf.SetID(element.ID(dropped.ID))
		if s.Folder == "" && dropped.ContainerID != "" {
			containerID = dropped.ContainerID
		}
		if len(dropped.AllowedRoles) > 0 {
			preserveAllowedRoles = false
		}
	}

	nf.SetName(s.Name.Name)
	nf.SetDocumentation(s.Documentation)
	nf.SetExcluded(s.Excluded)

	if preserveAllowedRoles {
		nf.SetAllowedModuleRolesQualifiedNames(existingAllowedRoles)
	} else {
		nf.SetAllowedModuleRolesQualifiedNames(defaultDocumentAccessRoleQNames(ectx, module))
	}

	paramElements := make([]*genMf.MicroflowParameter, 0, len(s.Parameters))
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
		paramElements = append(paramElements, param)
	}

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

		flow := newHorizontalFlowGen(start.ID(), end.ID())
		nf.AddFlows(flow)
	} else {
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
			isNanoflow:        true,
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

		oc = fb.buildFlowGraphGen(s.Body, s.ReturnType)

		if errs := fb.GetErrors(); len(errs) > 0 {
			return mdlerrors.NewValidationf(
				"nanoflow '%s' has validation errors:\n  - %s",
				qualifiedName, strings.Join(errs, "\n  - "))
		}

		for _, param := range paramElements {
			oc.AddObjects(param)
		}

		for _, flow := range fb.flows {
			nf.AddFlows(flow)
		}
		for _, af := range fb.annotationFlows {
			nf.AddFlows(af)
		}
	}

	nf.SetObjectCollection(oc)

	if s.ReturnType != nil {
		if t := paramASTToShortType(s.ReturnType.Type); t != "" {
			nf.SetReturnType(t)
		}
		if dt := convertASTToGenDataType(s.ReturnType.Type); dt != nil {
			nf.SetMicroflowReturnType(dt)
		}
		if s.ReturnType.Variable != "" {
			nf.SetReturnVariableName(s.ReturnType.Variable)
		}
	}

	if existing != nil {
		if err := deps.NanoflowRepo.Update(nf); err != nil {
			return mdlerrors.NewBackend("update nanoflow", err)
		}
		fmt.Fprintf(deps.Output, "Replaced nanoflow: %s\n", qualifiedName)
	} else {
		if err := deps.NanoflowRepo.Create(string(containerID), "Documents", nf); err != nil {
			return mdlerrors.NewBackend("create nanoflow", err)
		}
		fmt.Fprintf(deps.Output, "Created nanoflow: %s\n", qualifiedName)
	}

	returnEntityName := ""
	if s.ReturnType != nil {
		if ref := paramEntityRef(s.ReturnType.Type); ref != nil && ref.Module != "" {
			returnEntityName = ref.Module + "." + ref.Name
		}
	}
	ectx.trackCreatedNanoflow(s.Name.Module, s.Name.Name, model.ID(nf.ID()), containerID, returnEntityName)

	invalidateHierarchyFn(deps)
	invalidateMicroflowsCacheFn(deps)
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
	switch v := e.(type) {
	case *ast.LiteralExpr:
		switch v.Kind {
		case ast.LiteralBoolean:
			if b, ok := v.Value.(bool); ok {
				if b {
					return "true"
				}
				return "false"
			}
		case ast.LiteralString:
			if s, ok := v.Value.(string); ok {
				return "'" + s + "'"
			}
		case ast.LiteralInteger, ast.LiteralDecimal:
			return fmt.Sprintf("%v", v.Value)
		case ast.LiteralEmpty:
			return "empty"
		case ast.LiteralNull:
			return "null"
		}
	case *ast.VariableExpr:
		return "$" + v.Name
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
	case ast.TypeVoid:
		return "Void"
	case ast.TypeEntity:
		if t.EntityRef != nil {
			return t.EntityRef.Module + "." + t.EntityRef.Name
		}
	case ast.TypeEnumeration:
		// Bare qualified names parse as TypeEnumeration but may be entity
		// types (CLAUDE.md — TypeEnumeration vs TypeEntity Ambiguity).
		// Return the qualified name so extractEntityFromGenReturnType
		// can parse it downstream.
		if t.EnumRef != nil {
			return t.EnumRef.Module + "." + t.EnumRef.Name
		}
	case ast.TypeListOf:
		// Format matches extractEntityFromGenReturnType's expected prefix.
		if t.EntityRef != nil {
			return "List of " + t.EntityRef.Module + "." + t.EntityRef.Name
		}
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

// defaultDocumentAccessRoleQNamesFn is the HandlerDeps version of defaultDocumentAccessRoleQNames.
func defaultDocumentAccessRoleQNamesFn(deps *HandlerDeps, module *model.Module) []string {
	ectx := &ExecContext{Backend: deps.Backend}
	ids := defaultDocumentAccessRoles(ectx, module)
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
