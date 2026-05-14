// SPDX-License-Identifier: Apache-2.0

// Stage 3.2.3.i — gen-typed error-handler body emission.
//
// Counterpart of `cmd_microflows_builder_flows.go` lines 658-747:
//
//   - addErrorHandlerFlowGen — emits the EH body activities below
//     the source, connects them with chain flows, and an error edge
//     from the source activity to the first body activity.
//   - finishCustomErrorHandlerGen — top-level entry called from each
//     adder's wrap helper (genActivityWrap) when the action carries
//     an `on error custom` clause:
//       * empty clause → register handler for next normal-flow rejoin
//         (delegates to registerEmptyCustomErrorHandlerWithSkipGen)
//       * non-empty body → emit body via addErrorHandlerFlowGen, then
//         queue the merge target for the next normal-flow rejoin via
//         handleErrorHandlerMergeWithSkipGen
//
// Recursion: each statement in the body goes through addStatementGen
// (h1) so all 41 action adder kinds are reachable inside an EH body.
//
// Out of scope (deferred):
//
//   - buildRetryLoopErrorHandlerGen — the retry-loop pattern needs an
//     ExclusiveMerge spliced before the source activity so the body
//     tail loops back. The detection predicate (isRetryLoopErrorHandler)
//     is pure-AST and reusable; the wiring depends on the addStatement-
//     dispatcher recursion (now ready) but the topology is intricate
//     enough to warrant its own commit.
//
//   - branch-anchor preservation across EH body — fb.nextFlowAnchor /
//     fb.nextFlowCase carries authored @anchor / case overrides; the
//     minimal viable shipped here drops those when the EH body's last
//     activity is a compound statement. Real fixtures rarely set
//     @anchor inside EH bodies; if needed we'll extend.

package executor

import (
	"github.com/mendixlabs/mxcli/mdl/ast"
	"github.com/mendixlabs/mxcli/modelsdk/element"
)

// addErrorHandlerFlowGen builds error-handler body activities below
// the main flow, connects them with the standard error edge from the
// source activity, and returns an errorHandlerTailGen describing the
// last activity for the caller to merge back to the main flow.
//
// Returns errorHandlerTailGen{} (empty) when:
//   - the body is empty
//   - the body terminates (RAISE ERROR / RETURN); the handler
//     stops there and no merge is needed
func (fb *flowBuilderGen) addErrorHandlerFlowGen(sourceActivityID element.ID, sourceX int, errorBody []ast.MicroflowStatement) errorHandlerTailGen {
	if len(errorBody) == 0 {
		return errorHandlerTailGen{}
	}

	errorY := fb.posY + VerticalSpacing
	errorX := sourceX

	// Sub-builder for the EH body — shares varTypes/declaredVars with
	// parent (legacy parity: the EH body sees variables from the main
	// scope) and shares backend/repos/measurer/restServices.
	errBuilder := &flowBuilderGen{
		posX:           errorX,
		posY:           errorY,
		baseY:          errorY,
		spacing:        HorizontalSpacing,
		returnType:     fb.returnType,
		varTypes:       fb.varTypes,
		declaredVars:   fb.declaredVars,
		measurer:       fb.measurer,
		backend:        fb.backend,
		microflowsRepo: fb.microflowsRepo,
		nanoflowsRepo:  fb.nanoflowsRepo,
		hierarchy:      fb.hierarchy,
		restServices:   fb.restServices,
		isNanoflow:     fb.isNanoflow,
	}

	var lastErrID element.ID
	for _, stmt := range errorBody {
		actID := errBuilder.addStatementGen(stmt)
		if actID == "" {
			continue
		}
		errBuilder.applyPendingAnnotations(actID)
		if lastErrID == "" {
			// First activity — connect from source activity via the
			// error-handler edge (anchored bottom→top, IsErrorHandler).
			fb.flows = append(fb.flows, newErrorHandlerFlowGen(sourceActivityID, actID))
		} else {
			// Subsequent activities — plain horizontal chain. These
			// flows live on the EH sub-builder and get promoted to
			// the parent below.
			errBuilder.flows = append(errBuilder.flows, newHorizontalFlowGen(lastErrID, actID))
		}
		// Compound statement (nested IF/loop) advertises its merge via
		// nextConnectionPoint; consume so subsequent flow originates
		// there.
		if errBuilder.nextConnectionPoint != "" {
			lastErrID = errBuilder.nextConnectionPoint
			errBuilder.nextConnectionPoint = ""
		} else {
			lastErrID = actID
		}
	}

	// Promote sub-builder objects + flows + annotation flows to parent.
	fb.objects = append(fb.objects, errBuilder.objects...)
	fb.flows = append(fb.flows, errBuilder.flows...)
	fb.annotationFlows = append(fb.annotationFlows, errBuilder.annotationFlows...)

	if errBuilder.endsWithReturn {
		// Body terminates with RETURN / RAISE ERROR — no merge needed.
		return errorHandlerTailGen{}
	}
	return errorHandlerTailGen{id: lastErrID}
}

// handleErrorHandlerMergeWithSkipGen queues the EH body's tail to be
// rejoined into the next normal-flow edge. Skip-var semantics
// suppress the rejoin across statements that reference the variable
// (matches legacy handleErrorHandlerMergeWithSkip).
//
// No-op when the tail is empty (handler terminated with
// RETURN/RAISE ERROR).
func (fb *flowBuilderGen) handleErrorHandlerMergeWithSkipGen(tail errorHandlerTailGen, activityID element.ID, skipVar string) {
	if tail.id == "" {
		return
	}
	fb.queueActivePendingErrorHandlerGen()
	fb.errorHandlerSource = activityID
	fb.errorHandlerTailFrom = tail.id
	fb.errorHandlerSkipVar = skipVar
	fb.errorHandlerTailCase = tail.caseValue
	fb.errorHandlerTailAnchor = tail.flowAnchor
	fb.errorHandlerTailIsSource = false
	fb.errorHandlerReturnValue = fb.returnValue
}

// finishCustomErrorHandlerGen is the top-level entry called from
// each adder's wrap helper (genActivityWrap) when the action carries
// an `on error custom` (or `custom without rollback`) clause:
//
//   - nil clause → no-op
//   - empty clause body → register for next normal-flow rejoin via
//     registerEmptyCustomErrorHandlerWithSkipGen (commit d)
//   - non-empty body → emit body via addErrorHandlerFlowGen, then
//     queue the merge target via handleErrorHandlerMergeWithSkipGen
//
// outputVar is forwarded as the skip-var so the rejoin is suppressed
// across statements that reference the action's output variable
// (legacy semantics: avoid CE0108 "variable not in scope" by routing
// the error edge AFTER the variable's declaration).
func (fb *flowBuilderGen) finishCustomErrorHandlerGen(activityID element.ID, activityX int, eh *ast.ErrorHandlingClause, outputVar string) {
	if eh == nil {
		return
	}
	if len(eh.Body) > 0 {
		// TODO Stage 3.2.3.i+: retry-loop pattern detection (legacy
		// isRetryLoopErrorHandler + buildRetryLoopErrorHandler).
		// Fresh-author MDL rarely uses the retry pattern; describer
		// → exec roundtrip will need this when the pattern surfaces
		// in fixtures.
		tail := fb.addErrorHandlerFlowGen(activityID, activityX, eh.Body)
		fb.handleErrorHandlerMergeWithSkipGen(tail, activityID, outputVar)
		return
	}
	fb.registerEmptyCustomErrorHandlerWithSkipGen(activityID, eh, outputVar)
}
