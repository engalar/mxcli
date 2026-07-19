// SPDX-License-Identifier: Apache-2.0

// Stage 3.2.3.c — gen-typed sequence-flow constructors.
//
// This file is the gen-typed counterpart of the SequenceFlow helpers in
// `cmd_microflows_builder_flows.go`. It owns the primitives used by
// the per-statement adders (next commits) to wire activities together:
//
//   - newHorizontalFlowGen / newDownwardFlowWithCaseGen / newUpwardFlowGen
//     — direction-specific factories that hand back *genMf.SequenceFlow
//     with the right OriginConnectionIndex / DestinationConnectionIndex.
//   - newHorizontalFlowWithCaseGen / newHorizontalFlowWithEnumCaseGen /
//     newHorizontalFlowWithInheritanceCaseGen — variants that decorate
//     the flow with a CaseValue element (Mendix ExclusiveSplit branch
//     tag).
//   - newErrorHandlerFlowGen — vertically-anchored flow flagged
//     IsErrorHandler=true.
//   - caseValueForFlowGen / convertErrorHandlingTypeGen — translation
//     from MDL strings/AST to the gen EnumerationCase / ErrorHandlingType
//     vocabularies.
//   - applyUserAnchorsGen — overrides the builder-chosen anchors with
//     user-specified @anchor sides.
//
// Important divergence from the legacy SDK shape: gen has **no
// ExpressionCase** type. Studio Pro stores boolean true/false branch
// tags as EnumerationCase entries with Value "true" / "false" — and
// the gen describer (cmd_microflows_show_gen.go) already reads them
// back this way. So the gen builder follows suit: every boolean case
// is emitted as an EnumerationCase regardless of the legacy code's
// ExpressionCase choice.
//
// The error-handler queue logic (pendingErrorHandlerStateGen helpers,
// finishCustomErrorHandler, retry-loop emission) is intentionally not
// in this file — it lands in commit (d) once the per-statement adders
// it serves are also drafted.

package executor

import (
	"github.com/mendixlabs/mxcli/mdl/ast"
	"github.com/mendixlabs/mxcli/modelsdk/element"
	genMf "github.com/mendixlabs/mxcli/modelsdk/gen/microflows"
)

// newHorizontalFlowGen creates a SequenceFlow whose visual line runs
// left-to-right (origin's right anchor → destination's left anchor).
// The default for ordinary main-flow connections.
//
// Every unconditional flow carries a NoCase in CaseValues to match the
// BSON Studio Pro writes. Without it Mendix cannot traverse the flow
// graph and reports CE0108 "Variable not in scope".
// setSequenceFlowDefaults populates fields that Studio Pro always
// writes on every SequenceFlow (IsErrorHandler, Line BezierCurve).
func setSequenceFlowDefaults(flow *genMf.SequenceFlow) {
	flow.SetIsErrorHandler(false)
	curve := genMf.NewBezierCurve()
	assignFreshID(curve)
	curve.SetOriginControlVector("0;0")
	curve.SetDestinationControlVector("0;0")
	flow.SetLine(curve)
}

func newHorizontalFlowGen(originID, destinationID element.ID) *genMf.SequenceFlow {
	flow := genMf.NewSequenceFlow()
	assignFreshID(flow)
	flow.SetOriginID(originID)
	flow.SetDestinationID(destinationID)
	flow.SetOriginConnectionIndex(int32(AnchorRight))
	flow.SetDestinationConnectionIndex(int32(AnchorLeft))
	setSequenceFlowDefaults(flow)
	addNoCaseToFlow(flow)
	return flow
}

// newHorizontalFlowWithCaseGen wraps newHorizontalFlowGen and decorates
// the flow with a boolean case value. caseValue must be "true" or
// "false"; anything else is treated as an enum-style raw value via
// caseValueForFlowGen.
func newHorizontalFlowWithCaseGen(originID, destinationID element.ID, caseValue string) *genMf.SequenceFlow {
	flow := genMf.NewSequenceFlow()
	assignFreshID(flow)
	flow.SetOriginID(originID)
	flow.SetDestinationID(destinationID)
	flow.SetOriginConnectionIndex(int32(AnchorRight))
	flow.SetDestinationConnectionIndex(int32(AnchorLeft))
	setSequenceFlowDefaults(flow)
	if cv := caseValueForFlowGen(caseValue); cv != nil {
		flow.AddCaseValues(cv)
	}
	return flow
}

// newHorizontalFlowWithEnumCaseGen creates a horizontal flow whose
// CaseValues entry is an explicit EnumerationCase with the given raw
// value. Used by enum-split branch wiring.
func newHorizontalFlowWithEnumCaseGen(originID, destinationID element.ID, caseValue string) *genMf.SequenceFlow {
	flow := genMf.NewSequenceFlow()
	assignFreshID(flow)
	flow.SetOriginID(originID)
	flow.SetDestinationID(destinationID)
	flow.SetOriginConnectionIndex(int32(AnchorRight))
	flow.SetDestinationConnectionIndex(int32(AnchorLeft))
	setSequenceFlowDefaults(flow)
	ec := genMf.NewEnumerationCase()
	assignFreshID(ec)
	ec.SetValue(caseValue)
	flow.AddCaseValues(ec)
	return flow
}

// newHorizontalFlowWithInheritanceCaseGen creates a horizontal flow
// whose CaseValues entry is an InheritanceCase pointing at the given
// entity qualified name. Used by inheritance-split branch wiring.
func newHorizontalFlowWithInheritanceCaseGen(originID, destinationID element.ID, entityQN string) *genMf.SequenceFlow {
	flow := genMf.NewSequenceFlow()
	assignFreshID(flow)
	flow.SetOriginID(originID)
	flow.SetDestinationID(destinationID)
	flow.SetOriginConnectionIndex(int32(AnchorRight))
	flow.SetDestinationConnectionIndex(int32(AnchorLeft))
	setSequenceFlowDefaults(flow)
	ic := genMf.NewInheritanceCase()
	assignFreshID(ic)
	ic.SetValueQualifiedName(entityQN)
	flow.AddCaseValues(ic)
	return flow
}

// newDownwardFlowWithCaseGen creates a flow whose visual line goes
// down (origin's bottom anchor → destination's left anchor). Used
// when an IF's true branch dives below the main line.
func newDownwardFlowWithCaseGen(originID, destinationID element.ID, caseValue string) *genMf.SequenceFlow {
	flow := genMf.NewSequenceFlow()
	assignFreshID(flow)
	flow.SetOriginID(originID)
	flow.SetDestinationID(destinationID)
	flow.SetOriginConnectionIndex(int32(AnchorBottom))
	flow.SetDestinationConnectionIndex(int32(AnchorLeft))
	setSequenceFlowDefaults(flow)
	if cv := caseValueForFlowGen(caseValue); cv != nil {
		flow.AddCaseValues(cv)
	}
	return flow
}

// newDownwardFlowWithInheritanceCaseGen mirrors
// newHorizontalFlowWithInheritanceCaseGen but with downward visual
// direction.
func newDownwardFlowWithInheritanceCaseGen(originID, destinationID element.ID, entityQN string) *genMf.SequenceFlow {
	flow := genMf.NewSequenceFlow()
	assignFreshID(flow)
	flow.SetOriginID(originID)
	flow.SetDestinationID(destinationID)
	flow.SetOriginConnectionIndex(int32(AnchorBottom))
	flow.SetDestinationConnectionIndex(int32(AnchorLeft))
	setSequenceFlowDefaults(flow)
	ic := genMf.NewInheritanceCase()
	assignFreshID(ic)
	ic.SetValueQualifiedName(entityQN)
	flow.AddCaseValues(ic)
	return flow
}

// newUpwardFlowGen creates a flow whose visual line goes up (origin's
// right anchor → destination's bottom anchor). Used when a lower
// branch's tail merges back into the main line above.
func newUpwardFlowGen(originID, destinationID element.ID) *genMf.SequenceFlow {
	flow := genMf.NewSequenceFlow()
	assignFreshID(flow)
	flow.SetOriginID(originID)
	flow.SetDestinationID(destinationID)
	flow.SetOriginConnectionIndex(int32(AnchorRight))
	flow.SetDestinationConnectionIndex(int32(AnchorBottom))
	setSequenceFlowDefaults(flow)
	addNoCaseToFlow(flow)
	return flow
}

// newErrorHandlerFlowGen creates the dedicated flow that connects a
// failing source activity to the first activity of an error-handler
// body. Vertically anchored (bottom → top) and flagged
// IsErrorHandler=true so Studio Pro renders it with the red-stripe
// style and skips it in normal-control validation.
func newErrorHandlerFlowGen(originID, destinationID element.ID) *genMf.SequenceFlow {
	flow := genMf.NewSequenceFlow()
	assignFreshID(flow)
	flow.SetOriginID(originID)
	flow.SetDestinationID(destinationID)
	flow.SetOriginConnectionIndex(int32(AnchorBottom))
	flow.SetDestinationConnectionIndex(int32(AnchorTop))
	flow.SetIsErrorHandler(true)
	curve := genMf.NewBezierCurve()
	assignFreshID(curve)
	curve.SetOriginControlVector("0;0")
	curve.SetDestinationControlVector("0;0")
	flow.SetLine(curve)
	addNoCaseToFlow(flow)
	return flow
}

// addNoCaseToFlow appends a NoCase element to the CaseValues list of a
// SequenceFlow. Studio Pro always writes a NoCase on unconditional flows
// so Mendix can traverse the flow graph; omitting it causes CE0108.
func addNoCaseToFlow(flow *genMf.SequenceFlow) {
	nc := genMf.NewNoCase()
	assignFreshID(nc)
	flow.AddCaseValues(nc)
}

// caseValueForFlowGen translates an MDL-level case literal into a gen
// EnumerationCase element for use in CaseValues (plural).
//
// Returns nil for an empty caseValue — the caller must then use
// addNoCaseToFlow for unconditional flows.
func caseValueForFlowGen(caseValue string) element.Element {
	if caseValue == "" {
		return nil
	}
	ec := genMf.NewEnumerationCase()
	assignFreshID(ec)
	ec.SetValue(caseValue)
	return ec
}

// convertErrorHandlingTypeGen translates an AST ErrorHandlingClause
// into the gen ErrorHandlingType string Studio Pro expects. nil
// ErrorHandlingClause defaults to "Rollback" (the same default the
// legacy SDK constant ErrorHandlingTypeRollback maps to).
func convertErrorHandlingTypeGen(eh *ast.ErrorHandlingClause) string {
	if eh == nil {
		return "Rollback"
	}
	switch eh.Type {
	case ast.ErrorHandlingContinue:
		return "Continue"
	case ast.ErrorHandlingRollback:
		return "Rollback"
	case ast.ErrorHandlingCustom:
		return "Custom"
	case ast.ErrorHandlingCustomWithoutRollback:
		return "CustomWithoutRollBack"
	}
	return "Rollback"
}

// ehTypeGen returns the ErrorHandlingType for an action in this flow
// context. Microflows default to "Rollback"; nanoflows have no
// transactions so they default to "Abort". An explicit clause always
// overrides the default. Mirrors flowBuilder.ehType.
func (fb *flowBuilderGen) ehTypeGen(eh *ast.ErrorHandlingClause) string {
	if fb.isNanoflow && eh == nil {
		return "Abort"
	}
	return convertErrorHandlingTypeGen(eh)
}

// isEmptyCustomErrorHandlerGen reports whether eh is an empty `on
// error custom` (or `custom without rollback`) clause — i.e. the
// `custom` form without any body. The per-statement adders (next
// commits) use this to defer merge handling rather than emit a body.
func isEmptyCustomErrorHandlerGen(eh *ast.ErrorHandlingClause) bool {
	if eh == nil || len(eh.Body) != 0 {
		return false
	}
	return eh.Type == ast.ErrorHandlingCustom || eh.Type == ast.ErrorHandlingCustomWithoutRollback
}

// applyUserAnchorsGen overrides a SequenceFlow's connection indices
// with user-specified values from @anchor annotations. Either origin
// or destination can be nil (no annotation on that side); a non-nil
// anchor whose From/To is AnchorSideUnset is also a no-op so the
// builder default stays in place.
func applyUserAnchorsGen(flow *genMf.SequenceFlow, origin *ast.FlowAnchors, destination *ast.FlowAnchors) {
	if flow == nil {
		return
	}
	if origin != nil && origin.From != ast.AnchorSideUnset {
		flow.SetOriginConnectionIndex(int32(origin.From))
	}
	if destination != nil && destination.To != ast.AnchorSideUnset {
		flow.SetDestinationConnectionIndex(int32(destination.To))
	}
}
