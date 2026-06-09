// SPDX-License-Identifier: Apache-2.0

// Stage 3.2.3.d — gen-typed error-handler queue + deferred-rejoin
// flow emission.
//
// This file is the gen-typed counterpart of the error-handler state
// machinery in `cmd_microflows_builder_flows.go` (lines ~163–447). It
// owns the queue / active-state / rewrite helpers
// (queueActivePendingErrorHandlerGen, rewritePendingErrorHandlersGen),
// the deferred per-statement rejoin emitter
// (addPendingErrorHandlerFlowForStatementGen,
// addPendingErrorHandlerFlowForStateGen), the after-the-fact rejoin
// emitter (addPendingErrorHandlerFlowToGen), the empty-handler
// register helper (registerEmptyCustomErrorHandlerWithSkipGen), and
// the merge-insertion primitives that splice an ExclusiveMerge into
// an existing horizontal flow (addEmptyErrorHandlerRejoinFlowFromGen,
// addErrorHandlerRejoinFlowForStateGen).
//
// What's deliberately NOT in this commit:
//
//   - addErrorHandlerFlow / buildRetryLoopErrorHandler — both
//     require recursing into addStatement (per-statement adders
//     don't land until commits e/g/h/i), so they live with the
//     dispatcher in commit (e) onward.
//
//   - finishCustomErrorHandler — same dependency on
//     addErrorHandlerFlow + buildRetryLoopErrorHandler, deferred
//     to commit (e).
//
// All pure-AST helpers used by this state machine (statementReferencesVar,
// errorHandlerStatementVarRefs, outputDerivedVariable,
// callArgumentVarRefs, templateParamVarRefs, exprVarRefs,
// isRetryLoopErrorHandler, bodyTerminates) live in the legacy file
// and are reused as-is — they take no SDK type references.

package executor

import (
	"github.com/mendixlabs/mxcli/mdl/ast"
	"github.com/mendixlabs/mxcli/modelsdk/element"
	genMf "github.com/mendixlabs/mxcli/modelsdk/gen/microflows"
)

// ─── State container helpers ────────────────────────────────────────

// activeIsEmpty reports whether a pendingErrorHandlerStateGen carries
// no live routing — used to skip empty entries when iterating the
// queue.
func (s pendingErrorHandlerStateGen) activeIsEmpty() bool {
	return s.emptyFrom == "" && s.tailFrom == "" && s.source == "" && s.skipVar == ""
}

// activePendingErrorHandlerGen snapshots the builder's "currently
// active" error-handler routing into a value type the queue can
// rewrite without cross-aliasing.
func (fb *flowBuilderGen) activePendingErrorHandlerGen() pendingErrorHandlerStateGen {
	return pendingErrorHandlerStateGen{
		emptyFrom:    fb.emptyErrorHandlerFrom,
		tailFrom:     fb.errorHandlerTailFrom,
		source:       fb.errorHandlerSource,
		skipVar:      fb.errorHandlerSkipVar,
		tailCase:     fb.errorHandlerTailCase,
		tailAnchor:   fb.errorHandlerTailAnchor,
		tailIsSource: fb.errorHandlerTailIsSource,
		returnValue:  fb.errorHandlerReturnValue,
	}
}

// setActivePendingErrorHandlerGen restores a snapshotted state into
// the builder's flat fields. Used by the queue rewrite primitives.
func (fb *flowBuilderGen) setActivePendingErrorHandlerGen(state pendingErrorHandlerStateGen) {
	fb.emptyErrorHandlerFrom = state.emptyFrom
	fb.errorHandlerTailFrom = state.tailFrom
	fb.errorHandlerSource = state.source
	fb.errorHandlerSkipVar = state.skipVar
	fb.errorHandlerTailCase = state.tailCase
	fb.errorHandlerTailAnchor = state.tailAnchor
	fb.errorHandlerTailIsSource = state.tailIsSource
	fb.errorHandlerReturnValue = state.returnValue
}

// queueActivePendingErrorHandlerGen pushes the currently-active
// routing onto the deferred queue and clears the active slot. No-op
// when the active state is empty. Called when a new error handler is
// about to take ownership of the active fields.
func (fb *flowBuilderGen) queueActivePendingErrorHandlerGen() {
	state := fb.activePendingErrorHandlerGen()
	if state.activeIsEmpty() {
		return
	}
	fb.pendingErrorHandlers = append(fb.pendingErrorHandlers, state)
	fb.setActivePendingErrorHandlerGen(pendingErrorHandlerStateGen{})
}

// rewritePendingErrorHandlersGen runs `rewrite` over every queued
// state and the active state. States that become empty after rewrite
// are dropped; the active state is replaced in place. The queue
// preserves insertion order so handlers fire in the order they were
// registered.
func (fb *flowBuilderGen) rewritePendingErrorHandlersGen(rewrite func(pendingErrorHandlerStateGen) pendingErrorHandlerStateGen) {
	queue := fb.pendingErrorHandlers[:0]
	for _, state := range fb.pendingErrorHandlers {
		state = rewrite(state)
		if !state.activeIsEmpty() {
			queue = append(queue, state)
		}
	}
	fb.pendingErrorHandlers = queue

	active := rewrite(fb.activePendingErrorHandlerGen())
	fb.setActivePendingErrorHandlerGen(active)
}

// registerEmptyCustomErrorHandlerWithSkipGen records an empty `on
// error custom` handler so the next emitted activity wires up its
// rejoin edge. When skipVar is supplied the rejoin is suppressed
// across statements that reference that variable (matches legacy
// flowBuilder.registerEmptyCustomErrorHandlerWithSkip).
func (fb *flowBuilderGen) registerEmptyCustomErrorHandlerWithSkipGen(activityID element.ID, eh *ast.ErrorHandlingClause, skipVar string) {
	if !isEmptyCustomErrorHandlerGen(eh) {
		return
	}
	fb.queueActivePendingErrorHandlerGen()
	if skipVar == "" {
		fb.emptyErrorHandlerFrom = activityID
		return
	}
	fb.errorHandlerSource = activityID
	fb.errorHandlerTailFrom = activityID
	fb.errorHandlerSkipVar = skipVar
	fb.errorHandlerTailIsSource = true
}

// ─── Per-statement / after-the-fact rejoin emission ─────────────────

// addPendingErrorHandlerFlowForStatementGen runs the queue's per-flow
// rejoin logic against a freshly-created normal flow
// (originID → destinationID) emitted on behalf of stmt. Every queued
// state is asked to either emit its rejoin into this flow's geometry
// or pass through to the next iteration. Mirrors legacy
// flowBuilder.addPendingErrorHandlerFlowForStatement.
func (fb *flowBuilderGen) addPendingErrorHandlerFlowForStatementGen(originID, destinationID element.ID, stmt ast.MicroflowStatement, futureReferencesSkipVar ...bool) {
	futureReferences := len(futureReferencesSkipVar) > 0 && futureReferencesSkipVar[0]
	fb.rewritePendingErrorHandlersGen(func(state pendingErrorHandlerStateGen) pendingErrorHandlerStateGen {
		return fb.addPendingErrorHandlerFlowForStateGen(state, originID, destinationID, stmt, futureReferences)
	})
}

// addPendingErrorHandlerFlowToGen drains every queued handler whose
// rejoin can land at destinationID — used at the end of a containing
// branch when no further statements will be emitted to anchor the
// rejoin against. Mirrors legacy
// flowBuilder.addPendingErrorHandlerFlowTo.
func (fb *flowBuilderGen) addPendingErrorHandlerFlowToGen(destinationID element.ID) {
	if destinationID == "" {
		return
	}
	fb.rewritePendingErrorHandlersGen(func(state pendingErrorHandlerStateGen) pendingErrorHandlerStateGen {
		if state.emptyFrom != "" {
			fb.addEmptyErrorHandlerRejoinFlowFromGen(state.emptyFrom, state.emptyFrom, destinationID)
			state.emptyFrom = ""
		}
		if state.source != "" && state.tailFrom != "" {
			fb.addErrorHandlerRejoinFlowForStateGen(state, state.source, destinationID)
			state.source = ""
			state.tailFrom = ""
			state.skipVar = ""
			state.tailCase = ""
			state.tailAnchor = nil
			state.tailIsSource = false
			state.returnValue = ""
		}
		return state
	})
}

// addPendingErrorHandlerFlowForStateGen is the per-state core of the
// per-statement rejoin logic. It examines the state's emptyFrom,
// tailFrom, source, skipVar markers and either emits a rejoin into
// the (originID → destinationID) flow or returns the state unchanged
// for the next iteration to consume. Mirrors legacy
// flowBuilder.addPendingErrorHandlerFlowForState semantics.
func (fb *flowBuilderGen) addPendingErrorHandlerFlowForStateGen(state pendingErrorHandlerStateGen, originID, destinationID element.ID, stmt ast.MicroflowStatement, futureReferencesSkipVar bool) pendingErrorHandlerStateGen {
	if destinationID == "" {
		return state
	}
	if state.emptyFrom != "" {
		if state.emptyFrom != originID {
			return state
		}
		fb.addEmptyErrorHandlerRejoinFlowFromGen(originID, state.emptyFrom, destinationID)
		state.emptyFrom = ""
	}
	if state.tailFrom == "" {
		return state
	}
	if state.source != "" && destinationID == state.source {
		return state
	}
	if state.skipVar != "" {
		if statementReferencesVar(stmt, state.skipVar) {
			if !fb.hasDeclaredReturnValue() {
				if derivedVar := outputDerivedVariable(stmt, state.skipVar); derivedVar != "" {
					state.skipVar = derivedVar
				}
				return state
			}
			return state
		}
		if futureReferencesSkipVar {
			return state
		}
		fb.addErrorHandlerRejoinFlowForStateGen(state, originID, destinationID)
		return pendingErrorHandlerStateGen{}
	}
	if state.source != "" && state.source == originID {
		fb.addErrorHandlerRejoinFlowForStateGen(state, originID, destinationID)
		return pendingErrorHandlerStateGen{}
	}
	return state
}

// errorHandlerTailGen carries the tail-of-handler-body information
// that the (yet-to-land) addErrorHandlerFlow returns. Mirrors legacy
// errorHandlerTail.
type errorHandlerTailGen struct {
	id         element.ID
	caseValue  string
	flowAnchor *ast.FlowAnchors
}

// ─── Rejoin emission primitives ─────────────────────────────────────

// addEmptyErrorHandlerRejoinFlowFromGen splices an ExclusiveMerge into
// an existing horizontal flow (normalOriginID → destinationID) so an
// error-handler rejoin can land at the same point. If no such flow
// exists, falls back to a direct error-handler edge. Mirrors legacy
// flowBuilder.addEmptyErrorHandlerRejoinFlowFrom.
func (fb *flowBuilderGen) addEmptyErrorHandlerRejoinFlowFromGen(normalOriginID, errorOriginID, destinationID element.ID) {
	existingIdx := -1
	for i := len(fb.flows) - 1; i >= 0; i-- {
		flow := fb.flows[i]
		if !flow.IsErrorHandler() && flow.OriginRefID() == normalOriginID && flow.DestinationRefID() == destinationID {
			existingIdx = i
			break
		}
	}
	if existingIdx == -1 {
		if mergeID := fb.findExistingRejoinMergeGen(normalOriginID, destinationID); mergeID != "" {
			fb.flows = append(fb.flows, newErrorHandlerFlowGen(errorOriginID, mergeID))
			return
		}
		fb.flows = append(fb.flows, newErrorHandlerFlowGen(errorOriginID, destinationID))
		return
	}

	existing := fb.flows[existingIdx]
	fb.flows = append(fb.flows[:existingIdx], fb.flows[existingIdx+1:]...)

	merge := genMf.NewExclusiveMerge()
	mergeID := assignFreshID(merge)
	merge.SetRelativeMiddlePoint(layoutPos(fb.posX-HorizontalSpacing/2, fb.baseY))
	merge.SetSize(layoutSize(MergeSize, MergeSize))
	fb.objects = append(fb.objects, merge)

	normalFlow := newHorizontalFlowGen(normalOriginID, mergeID)
	normalFlow.SetOriginConnectionIndex(existing.OriginConnectionIndex())
	if cv := existing.CaseValue(); cv != nil {
		normalFlow.SetCaseValue(cv)
	}
	fb.flows = append(fb.flows, normalFlow)
	fb.flows = append(fb.flows, newErrorHandlerFlowGen(errorOriginID, mergeID))

	mergeFlow := newHorizontalFlowGen(mergeID, destinationID)
	mergeFlow.SetDestinationConnectionIndex(existing.DestinationConnectionIndex())
	fb.flows = append(fb.flows, mergeFlow)
}

// addErrorHandlerRejoinFlowForStateGen is the non-empty-handler
// counterpart to addEmptyErrorHandlerRejoinFlowFromGen: the queued
// state's tail can be the source itself (treat as error edge) or a
// genuine handler-body tail (treat as upward rejoin). Mirrors legacy
// flowBuilder.addErrorHandlerRejoinFlowForState.
func (fb *flowBuilderGen) addErrorHandlerRejoinFlowForStateGen(state pendingErrorHandlerStateGen, originID, destinationID element.ID) {
	existingIdx := -1
	for i := len(fb.flows) - 1; i >= 0; i-- {
		flow := fb.flows[i]
		if !flow.IsErrorHandler() && flow.OriginRefID() == originID && flow.DestinationRefID() == destinationID {
			existingIdx = i
			break
		}
	}
	if existingIdx == -1 {
		if mergeID := fb.findExistingRejoinMergeGen(originID, destinationID); mergeID != "" {
			if state.tailIsSource {
				fb.flows = append(fb.flows, newErrorHandlerFlowGen(state.tailFrom, mergeID))
			} else {
				flow := newUpwardFlowGen(state.tailFrom, mergeID)
				applyDeferredFlowCaseGen(flow, state.tailCase, state.tailAnchor)
				fb.flows = append(fb.flows, flow)
			}
			return
		}
		if state.tailIsSource {
			fb.flows = append(fb.flows, newErrorHandlerFlowGen(state.tailFrom, destinationID))
		} else {
			flow := newHorizontalFlowGen(state.tailFrom, destinationID)
			applyDeferredFlowCaseGen(flow, state.tailCase, state.tailAnchor)
			fb.flows = append(fb.flows, flow)
		}
		return
	}

	existing := fb.flows[existingIdx]
	fb.flows = append(fb.flows[:existingIdx], fb.flows[existingIdx+1:]...)

	merge := genMf.NewExclusiveMerge()
	mergeID := assignFreshID(merge)
	merge.SetRelativeMiddlePoint(layoutPos(fb.posX-HorizontalSpacing/2, fb.baseY))
	merge.SetSize(layoutSize(MergeSize, MergeSize))
	fb.objects = append(fb.objects, merge)

	normalFlow := newHorizontalFlowGen(originID, mergeID)
	normalFlow.SetOriginConnectionIndex(existing.OriginConnectionIndex())
	if cv := existing.CaseValue(); cv != nil {
		normalFlow.SetCaseValue(cv)
	}
	fb.flows = append(fb.flows, normalFlow)
	if state.tailIsSource {
		fb.flows = append(fb.flows, newErrorHandlerFlowGen(state.tailFrom, mergeID))
	} else {
		flow := newUpwardFlowGen(state.tailFrom, mergeID)
		applyDeferredFlowCaseGen(flow, state.tailCase, state.tailAnchor)
		fb.flows = append(fb.flows, flow)
	}

	mergeFlow := newHorizontalFlowGen(mergeID, destinationID)
	mergeFlow.SetDestinationConnectionIndex(existing.DestinationConnectionIndex())
	fb.flows = append(fb.flows, mergeFlow)
}

// applyDeferredFlowCaseGen applies a deferred case value and/or
// anchor to a freshly-created flow. Companion to legacy
// applyDeferredFlowCase. Always uses EnumerationCase since gen has
// no ExpressionCase (caseValueForFlowGen documents the rationale).
func applyDeferredFlowCaseGen(flow *genMf.SequenceFlow, caseValue string, anchor *ast.FlowAnchors) {
	if flow == nil {
		return
	}
	if caseValue != "" {
		ec := genMf.NewEnumerationCase()
		assignFreshID(ec)
		ec.SetValue(caseValue)
		flow.SetCaseValue(ec)
	}
	applyUserAnchorsGen(flow, anchor, anchor)
}

// findExistingRejoinMergeGen scans fb.flows for an
// origin → ExclusiveMerge → destination pattern and returns the merge
// ID if found. Used by the rejoin emitters to splice into existing
// merges rather than create duplicates. Returns "" when no candidate
// merge is found. O(flows²) but error-handler rejoins are rare.
func (fb *flowBuilderGen) findExistingRejoinMergeGen(originID, destinationID element.ID) element.ID {
	for _, flow := range fb.flows {
		if flow.OriginRefID() != originID || flow.IsErrorHandler() {
			continue
		}
		if !fb.isExclusiveMergeGen(flow.DestinationRefID()) {
			continue
		}
		for _, mergeFlow := range fb.flows {
			if mergeFlow.OriginRefID() == flow.DestinationRefID() && mergeFlow.DestinationRefID() == destinationID && !mergeFlow.IsErrorHandler() {
				return flow.DestinationRefID()
			}
		}
	}
	return ""
}

// isExclusiveMergeGen reports whether the object identified by id is
// a *genMf.ExclusiveMerge. Linear scan on fb.objects, same complexity
// budget as the legacy variant.
func (fb *flowBuilderGen) isExclusiveMergeGen(id element.ID) bool {
	for _, obj := range fb.objects {
		if obj == nil || obj.ID() != id {
			continue
		}
		_, ok := obj.(*genMf.ExclusiveMerge)
		return ok
	}
	return false
}
