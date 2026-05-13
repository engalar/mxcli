// SPDX-License-Identifier: Apache-2.0

// Stage 3.2.3.a — gen-typed flow-graph builder foundation.
//
// This file defines `flowBuilderGen`, the modelsdk/gen-native counterpart
// to `flowBuilder` (cmd_microflows_builder.go). Subsequent 3.2.3.b…j
// commits add the per-statement emitters, control-flow plumbing,
// sequence-flow constructors, validators, and the create-microflow
// entry point that drive the gen write path end-to-end.
//
// Scope of this commit (a):
//   - Struct definition mirroring the legacy `flowBuilder` state shape
//     but typed against `*genMf.*` and `element.Element`.
//   - Variable-state snapshot/restore + clone helpers (so subsequent
//     control-flow files can scope branch-local variable visibility).
//   - exprToString-equivalent that routes through `fb.execAdapter` when
//     wired, falling back to the same `expressionToString` helper used
//     by the legacy path (keeps text output 1:1 across paths).
//   - Layout-string helper `layoutPos` — gen activities store position
//     and size as a single space-separated string (e.g. "100 200")
//     instead of legacy `model.Point` / `model.Size` structs.
//   - Validation error collection (mirrors `addError` /
//     `addErrorWithExample` / `GetErrors`).
//   - Microflow / nanoflow existence + return-type lookups via
//     `ctx.Microflows` and `ctx.Nanoflows` repos rather than the SDK
//     backend (matches Stage 3.2.5a's helpers_gen.go pattern).
//
// What's intentionally absent — added in later 3.2.3.* commits:
//   - addStatement dispatcher and per-statement adders (graph,
//     actions, calls, control, workflow files).
//   - sequenceFlow constructors and error-handler queue logic
//     (flows file).
//   - annotations / EndEvent / ErrorEvent (annotations file).
//   - execCreateMicroflowGen entry point (create_gen file).
//
// The struct deliberately omits a handful of legacy fields that turned
// out to be dead in the gen path (e.g. legacy uses `*microflows.Microflow`
// to look up rule/microflow return types via the sdk types' `ReturnType`
// field; the gen path resolves return-type strings from gen objects).

package executor

import (
	"fmt"
	"strings"

	"github.com/mendixlabs/mxcli/mdl/ast"
	"github.com/mendixlabs/mxcli/mdl/backend"
	"github.com/mendixlabs/mxcli/mdl/exprcheck/adapters"
	"github.com/mendixlabs/mxcli/mdl/repos"
	"github.com/mendixlabs/mxcli/mdl/types"
	"github.com/mendixlabs/mxcli/model"
	"github.com/mendixlabs/mxcli/modelsdk/element"
	genMf "github.com/mendixlabs/mxcli/modelsdk/gen/microflows"
	msdkprop "github.com/mendixlabs/mxcli/modelsdk/property"
)

// flowBuilderGen builds the flow graph from AST statements using
// modelsdk/gen-native types. It is the eventual replacement for
// `flowBuilder`; the two will run in parallel until Stage 3.2.6 deletes
// the legacy entry point.
//
// Field ordering mirrors the legacy struct so reviewers can diff the
// two layouts side-by-side; comments explain only divergences.
type flowBuilderGen struct {
	// execAdapter / currentSlot / microflowQN — same role as on the
	// legacy builder (route expressions through the exprcheck robust
	// parser). P3 wires per-slot context.
	execAdapter *adapters.ExecAdapter
	currentSlot string
	microflowQN string

	// objects holds gen elements that ultimately land on the
	// MicroflowObjectCollection. The legacy slice was typed against
	// the sdk `microflows.MicroflowObject` interface; the gen
	// equivalent is the (untyped) `element.Element` because gen does
	// not expose a unifying interface yet (codec dispatch is by
	// concrete type).
	objects []element.Element

	// flows / annotationFlows hold gen sequence-flow and annotation-flow
	// elements that land on the Microflow's Flows array.
	flows           []*genMf.SequenceFlow
	annotationFlows []*genMf.AnnotationFlow

	posX            int
	posY            int
	baseY           int
	spacing         int
	returnValue     string
	returnType      *ast.MicroflowReturnType
	endsWithReturn  bool
	lastReturnEndID element.ID
	varTypes        map[string]string
	declaredVars    map[string]string
	errors          []string
	measurer        *layoutMeasurer

	// nextConnectionPoint is the exit ID that the next emitted flow
	// should originate from (used for compound statements where the
	// entry and exit IDs differ).
	nextConnectionPoint element.ID
	// incomingRedirect — see legacy flowBuilder.incomingRedirect.
	incomingRedirect element.ID
	nextFlowCase     string
	nextFlowAnchor   *ast.FlowAnchors

	backend            backend.FullBackend
	microflowsRepo     repos.MicroflowRepository
	nanoflowsRepo      repos.NanoflowRepository
	hierarchy          *ContainerHierarchy
	pendingAnnotations *ast.ActivityAnnotations

	// restServices is cached from the executor entry so each call into
	// the builder doesn't re-query the backend.
	restServices []*model.ConsumedRestService

	// listInputVariables / objectInputVariables — pre-collected sets
	// the per-statement adders consult to decide whether an output
	// variable is later used as a list or as an object/attribute root.
	listInputVariables   map[string]bool
	objectInputVariables map[string]bool

	previousStmtAnchor *ast.FlowAnchors

	// Cached gen flow lists. The gen path resolves return types and
	// existence via `ctx.Microflows` / `ctx.Nanoflows` rather than the
	// SDK backend (helpers_gen.go pattern). Cache loads on first use.
	microflowsCache       []*genMf.Microflow
	microflowsCacheLoaded bool
	nanoflowsCache        []*genMf.Nanoflow
	nanoflowsCacheLoaded  bool

	manualLoopBackTarget element.ID
	isNanoflow           bool

	// Pending custom error-handler routing. Same shape as the legacy
	// builder; subsequent commits add the queue helpers.
	emptyErrorHandlerFrom    element.ID
	errorHandlerTailFrom     element.ID
	errorHandlerSource       element.ID
	errorHandlerSkipVar      string
	errorHandlerTailCase     string
	errorHandlerTailAnchor   *ast.FlowAnchors
	errorHandlerTailIsSource bool
	errorHandlerReturnValue  string
	pendingErrorHandlers     []pendingErrorHandlerStateGen
}

// pendingErrorHandlerStateGen is the gen-typed equivalent of
// pendingErrorHandlerState (cmd_microflows_builder_flows.go). The shape
// is identical — only the ID type changes from `model.ID` to
// `element.ID`. Both alias to `string`, but keeping them distinct
// surfaces accidental mixing through the type checker.
type pendingErrorHandlerStateGen struct {
	emptyFrom    element.ID
	tailFrom     element.ID
	source       element.ID
	skipVar      string
	tailCase     string
	tailAnchor   *ast.FlowAnchors
	tailIsSource bool
	returnValue  string
}

// flowBuilderGenVariableState is the gen counterpart of
// flowBuilderVariableState. snapshotVariableState / restoreVariableState
// let scoped-statement helpers (if/loop/case bodies) clone variable
// visibility so branch-local declarations don't leak into siblings.
type flowBuilderGenVariableState struct {
	varTypes     map[string]string
	declaredVars map[string]string
}

func (fb *flowBuilderGen) snapshotVariableState() flowBuilderGenVariableState {
	return flowBuilderGenVariableState{
		varTypes:     cloneStringMap(fb.varTypes),
		declaredVars: cloneStringMap(fb.declaredVars),
	}
}

func (fb *flowBuilderGen) restoreVariableState(state flowBuilderGenVariableState) {
	fb.varTypes = state.varTypes
	fb.declaredVars = state.declaredVars
}

// addError appends a validation error. Same surface as the legacy
// builder so test expectations carry over.
func (fb *flowBuilderGen) addError(format string, args ...any) {
	fb.errors = append(fb.errors, fmt.Sprintf(format, args...))
}

// addErrorWithExample appends a validation error followed by a code
// example illustrating the fix. Mirrors legacy semantics including the
// indented "Example:" prefix.
func (fb *flowBuilderGen) addErrorWithExample(message, example string) {
	fb.errors = append(fb.errors, fmt.Sprintf("%s\n\n  Example:\n%s", message, example))
}

// GetErrors returns all validation errors collected during build.
func (fb *flowBuilderGen) GetErrors() []string {
	return fb.errors
}

// hasDeclaredReturnValue mirrors flowBuilder.hasDeclaredReturnValue.
func (fb *flowBuilderGen) hasDeclaredReturnValue() bool {
	return fb.returnType != nil && fb.returnType.Type.Kind != ast.TypeVoid
}

// isVariableDeclared reports whether varName has been declared either
// as an entity / list-of-entity (varTypes) or as a primitive
// (declaredVars). Strips the leading `$` and any trailing path so that
// `$Foo/Bar` and `$Foo` both resolve via `Foo`.
func (fb *flowBuilderGen) isVariableDeclared(varName string) bool {
	name := strings.TrimPrefix(varName, "$")
	if idx := strings.IndexByte(name, '/'); idx >= 0 {
		name = name[:idx]
	}
	if _, ok := fb.varTypes[name]; ok {
		return true
	}
	if _, ok := fb.declaredVars[name]; ok {
		return true
	}
	return false
}

// layoutPos formats an (x, y) pair as the single space-separated string
// that gen activities store under their RelativeMiddlePoint field. The
// gen codec serializes layout coordinates as text, not as nested {X,Y}
// documents — this helper is the single source of truth for that
// format so the per-statement adders never hand-roll it.
func layoutPos(x, y int) string {
	return fmt.Sprintf("%d %d", x, y)
}

// layoutSize formats a (width, height) pair as the single
// space-separated string that gen activities store under their Size
// field. Companion to layoutPos.
func layoutSize(width, height int) string {
	return fmt.Sprintf("%d %d", width, height)
}

// genElementWithID is the minimal interface every gen element exposes
// for ID assignment. Used by newGenID to keep the per-statement
// adders free of explicit `SetID(element.ID(types.GenerateID()))`
// boilerplate at every emission site.
type genElementWithID interface {
	ID() element.ID
	SetID(element.ID)
}

// assignFreshID stamps a freshly-generated UUID on a gen element if it
// doesn't already carry one. Returns the resulting ID. The legacy
// builder relied on `model.BaseElement{ID: model.ID(types.GenerateID())}`
// at every construction site; the gen path centralises that pattern
// here so SequenceFlows, AnnotationFlows, and downstream lookups can
// reference the element immediately rather than waiting for the codec
// to backfill empty IDs at encode time.
func assignFreshID(e genElementWithID) element.ID {
	if id := e.ID(); id != "" {
		return id
	}
	id := element.ID(types.GenerateID())
	e.SetID(id)
	return id
}

// genBaseHolder is the minimal interface every concrete gen element
// satisfies for ad-hoc property injection. element.Base implements
// AddProperty (see modelsdk/element/element.go:138 — explicitly intended
// for "inherited or ad-hoc properties that the codegen doesn't produce").
type genBaseHolder interface {
	AddProperty(p element.Property, bit uint)
}

// setExtraStringField injects a string-valued ad-hoc field onto a gen
// element so the codec emits it on encode. Used to bridge gen-schema
// gaps where the reflection-data carries a field but codegen didn't
// produce a typed setter (e.g. CastAction.ObjectVariableName — gen
// reads it via raw BSON in show_gen but exposes no setter).
//
// The injected Property is a fresh `*property.Primitive[string]` with
// the supplied value pre-Set (so it reports Dirty and the encoder
// overlays it onto the BSON output document via setField).
//
// The bit argument is forwarded to Base.AddProperty for dirty
// tracking; gen elements use bits 0..N for codegen-bound properties
// and bit 63 for "is new". Bit 62 is reserved here for ad-hoc fields
// (no codegen Property uses it, so collisions are impossible).
//
// No-op when value is empty — Mendix BSON omits empty optional
// strings, and adding a dirty empty Property would cause the codec
// to emit a stray empty key.
func setExtraStringField(e element.Element, key, value string) {
	if value == "" {
		return
	}
	holder, ok := e.(genBaseHolder)
	if !ok {
		return
	}
	prop := msdkprop.NewPrimitive[string](key, msdkprop.DecodeString)
	prop.Set(value)
	holder.AddProperty(prop, 62)
}

// exprToString converts an AST Expression to a Mendix expression string
// using the same `expressionToString` helper the legacy builder uses,
// after first walking the tree to resolve association navigation paths
// (so `$Order/Mod.Order_Customer/Name` becomes
// `$Order/Mod.Order_Customer/Mod.Customer/Name`).
//
// When fb.execAdapter is non-nil the expression is also routed through
// the exprcheck robust parser, which prints any hints to the adapter's
// writer and skips the BSON write for error-level hints.
func (fb *flowBuilderGen) exprToString(expr ast.Expression) string {
	if fb.execAdapter != nil {
		if out := fb.execAdapter.ExprToBSON(fb.currentSlot, expr, fb.microflowQN); out != "" {
			return out
		}
	}
	resolved := fb.resolveAssociationPaths(expr)
	return expressionToString(resolved)
}

// resolveAssociationPaths walks an expression tree and, for every
// AttributePathExpr whose path contains an association (qualified name
// like `Module.AssocName`), inserts the association's target entity
// after the association segment. Same algorithm as
// flowBuilder.resolveAssociationPaths — duplicated here rather than
// promoted to a shared helper because of the receiver-type difference,
// and because Stage 3.2.6 will delete the legacy method outright.
func (fb *flowBuilderGen) resolveAssociationPaths(expr ast.Expression) ast.Expression {
	if expr == nil {
		return nil
	}

	switch e := expr.(type) {
	case *ast.AttributePathExpr:
		resolved := fb.resolvePathSegments(e.Path)
		return &ast.AttributePathExpr{
			Variable: e.Variable,
			Path:     resolved,
			Segments: e.Segments,
		}
	case *ast.BinaryExpr:
		return &ast.BinaryExpr{
			Left:     fb.resolveAssociationPaths(e.Left),
			Operator: e.Operator,
			Right:    fb.resolveAssociationPaths(e.Right),
		}
	case *ast.UnaryExpr:
		return &ast.UnaryExpr{
			Operator: e.Operator,
			Operand:  fb.resolveAssociationPaths(e.Operand),
		}
	case *ast.FunctionCallExpr:
		args := make([]ast.Expression, len(e.Arguments))
		for i, arg := range e.Arguments {
			args[i] = fb.resolveAssociationPaths(arg)
		}
		return &ast.FunctionCallExpr{
			Name:      e.Name,
			Arguments: args,
		}
	case *ast.ParenExpr:
		return &ast.ParenExpr{Inner: fb.resolveAssociationPaths(e.Inner)}
	case *ast.IfThenElseExpr:
		return &ast.IfThenElseExpr{
			Condition: fb.resolveAssociationPaths(e.Condition),
			ThenExpr:  fb.resolveAssociationPaths(e.ThenExpr),
			ElseExpr:  fb.resolveAssociationPaths(e.ElseExpr),
		}
	case *ast.SourceExpr:
		if e.Source != "" {
			return e
		}
		return fb.resolveAssociationPaths(e.Expression)
	default:
		return expr
	}
}

// resolvePathSegments processes path segments in an attribute-path
// expression. For each segment that is a qualified association name
// (`Module.AssocName`), it looks up the association's target entity
// and inserts it after the association.
//
// The lookup itself currently still goes through the legacy
// flowBuilder.lookupAssociation (which queries the SDK backend), since
// the gen domain-model repository is out of scope for Stage 3.2.3
// (Stage 3.1 only ships microflows + nanoflows). Once a gen
// DomainModelRepository lands, this will switch over.
func (fb *flowBuilderGen) resolvePathSegments(path []string) []string {
	if fb.backend == nil || len(path) == 0 {
		return path
	}

	var resolved []string
	for i, segment := range path {
		resolved = append(resolved, segment)

		if !strings.Contains(segment, ".") {
			continue
		}
		if i+1 < len(path) && strings.Contains(path[i+1], ".") {
			continue
		}
		if i == len(path)-1 {
			continue
		}

		parts := strings.SplitN(segment, ".", 2)
		if len(parts) != 2 {
			continue
		}
		// Reuse the legacy lookup. assocLookupResult is shared.
		legacy := &flowBuilder{backend: fb.backend, hierarchy: fb.hierarchy}
		result := legacy.lookupAssociation(parts[0], parts[1])
		if result != nil && result.childEntityQN != "" {
			resolved = append(resolved, result.childEntityQN)
		}
	}
	return resolved
}

// microflowExistsGen returns true if qualifiedName refers to a
// microflow present in the connected project. Same fall-through-to-true
// semantics as the legacy variant: returns true on any error or when
// no repo is available so offline / syntax-check mode is unaffected.
func (fb *flowBuilderGen) microflowExistsGen(qualifiedName string) bool {
	if fb.microflowsRepo == nil {
		return true
	}
	if mf, err := fb.microflowsRepo.FindByQualifiedName(qualifiedName); err == nil && mf != nil {
		return true
	}
	if !fb.microflowsCacheLoaded {
		list, err := fb.microflowsRepo.ListAll()
		if err != nil {
			return true
		}
		fb.microflowsCache = list
		fb.microflowsCacheLoaded = true
	}
	moduleName, mfName, ok := strings.Cut(qualifiedName, ".")
	if !ok {
		return true
	}
	for _, mf := range fb.microflowsCache {
		if mf == nil || mf.Name() != mfName {
			continue
		}
		if fb.containerModuleName(model.ID(mf.ID())) == moduleName {
			return true
		}
	}
	return false
}

// nanoflowExistsGen mirrors microflowExistsGen for nanoflows.
func (fb *flowBuilderGen) nanoflowExistsGen(qualifiedName string) bool {
	if fb.nanoflowsRepo == nil {
		return true
	}
	if !fb.nanoflowsCacheLoaded {
		list, err := fb.nanoflowsRepo.List("")
		if err != nil {
			return true
		}
		fb.nanoflowsCache = list
		fb.nanoflowsCacheLoaded = true
	}
	moduleName, nfName, ok := strings.Cut(qualifiedName, ".")
	if !ok {
		return true
	}
	for _, nf := range fb.nanoflowsCache {
		if nf == nil || nf.Name() != nfName {
			continue
		}
		if fb.containerModuleName(model.ID(nf.ID())) == moduleName {
			return true
		}
	}
	return false
}

// containerModuleName resolves a flow's owning module name via the
// hierarchy. Returns "" when the resolution chain is incomplete.
// Used by microflowExistsGen / nanoflowExistsGen and by future
// per-statement adders that need to display qualified names.
func (fb *flowBuilderGen) containerModuleName(id model.ID) string {
	if id == "" || fb.hierarchy == nil || fb.microflowsRepo == nil {
		return ""
	}
	cid, err := fb.microflowsRepo.GetContainerUUID(id)
	if err != nil || cid == "" {
		return ""
	}
	modID := fb.hierarchy.FindModuleID(cid)
	return fb.hierarchy.GetModuleName(modID)
}
