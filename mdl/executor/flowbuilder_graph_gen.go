// SPDX-License-Identifier: Apache-2.0

// Stage 3.2.3.j — gen-typed buildFlowGraphGen entry.
//
// Top-level driver that turns a microflow body (slice of AST
// statements) into a complete *genMf.MicroflowObjectCollection
// with StartEvent + per-statement activities + sequence flow chain
// + EndEvent (synthesised when the body doesn't already terminate).
//
// Returns the assembled ObjectCollection. The caller (k —
// execCreateMicroflowGen) takes the flows / annotation flows from
// fb.flows / fb.annotationFlows and copies them onto the
// Microflow's Flows array (gen Mendix BSON layout: flows live at
// the microflow level, not inside the ObjectCollection).
//
// Out of scope (deferred from legacy):
//
//   - @position annotation pre-scan (legacy shifts StartEvent left
//     of authored coordinates) — when first stmt has @position
//   - retry-loop incomingRedirect handling (already deferred from i)
//   - terminatePendingErrorHandlersAtEnd full implementation —
//     placeholder no-op here (the EH queue helpers from d are
//     wired but the terminal-rejoin emission needs more work)
//   - @anchor preservation across compound statements
//   - free-floating @annotation flush at body end
//
// The minimal viable shipped here covers the dominant fresh-author
// shape: linear chain of statements with optional final RETURN.

package executor

import (
	"github.com/mendixlabs/mxcli/mdl/ast"
	"github.com/mendixlabs/mxcli/modelsdk/element"
	genMf "github.com/mendixlabs/mxcli/modelsdk/gen/microflows"
)

// buildFlowGraphGen iterates the microflow body and emits the
// complete object graph. Returns the *genMf.MicroflowObjectCollection
// that the caller attaches to the Microflow being built.
//
// Side effect: populates fb.objects / fb.flows / fb.annotationFlows
// with the emitted elements. The caller owns moving fb.flows onto
// the Microflow's Flows array.
func (fb *flowBuilderGen) buildFlowGraphGen(stmts []ast.MicroflowStatement, returns *ast.MicroflowReturnType) *genMf.MicroflowObjectCollection {
	// Defensive init — caller may forget.
	if fb.measurer == nil {
		fb.measurer = &layoutMeasurer{varTypes: fb.varTypes}
	}
	if fb.declaredVars == nil {
		fb.declaredVars = make(map[string]string)
	}
	if fb.varTypes == nil {
		fb.varTypes = make(map[string]string)
	}
	// Set return value expression for synthesised EndEvent.
	fb.returnType = returns
	if returns != nil && returns.Variable != "" {
		fb.returnValue = "$" + returns.Variable
	}
	fb.baseY = fb.posY

	// Emit StartEvent at the current cursor.
	startEvent := genMf.NewStartEvent()
	startID := assignFreshID(startEvent)
	startEvent.SetRelativeMiddlePoint(layoutPos(fb.posX, fb.posY))
	startEvent.SetSize(layoutSize(EventSize, EventSize))
	fb.objects = append(fb.objects, startEvent)
	lastID := startID
	fb.posX += fb.spacing

	// Iterate body statements via the dispatcher (h1).
	for _, stmt := range stmts {
		activityID := fb.addStatementGen(stmt)
		if activityID == "" {
			continue
		}
		fb.applyPendingAnnotations(activityID)

		// Connect previous to current.
		flow := newHorizontalFlowGen(lastID, activityID)
		fb.flows = append(fb.flows, flow)

		// Compound statements (IF/loop/split) advertise their merge via
		// nextConnectionPoint; consume so subsequent flow originates
		// from the right place.
		if fb.nextConnectionPoint != "" {
			lastID = fb.nextConnectionPoint
			fb.nextConnectionPoint = ""
		} else {
			lastID = activityID
		}
	}

	// Synthesise final EndEvent unless the body already terminates
	// (fb.endsWithReturn set by RETURN / both-branches-return etc).
	if !fb.endsWithReturn {
		fb.posX += fb.spacing / 2
		fb.posY = fb.baseY
		endEvent := genMf.NewEndEvent()
		endID := assignFreshID(endEvent)
		endEvent.SetRelativeMiddlePoint(layoutPos(fb.posX, fb.posY))
		endEvent.SetSize(layoutSize(EventSize, EventSize))
		if fb.returnValue != "" {
			endEvent.SetReturnValue(fb.returnValue)
		}
		fb.objects = append(fb.objects, endEvent)

		// Connect last activity (or StartEvent for empty body) to EndEvent.
		fb.flows = append(fb.flows, newHorizontalFlowGen(lastID, endID))
	}

	// Wrap all emitted objects into the MicroflowObjectCollection
	// that the caller attaches to the Microflow.
	oc := genMf.NewMicroflowObjectCollection()
	assignFreshID(oc)
	for _, obj := range fb.objects {
		oc.AddObjects(obj)
	}
	return oc
}

// flowBuilderGenObjects returns the slice of element.Element objects
// emitted into fb.objects so far. Exposed for the caller (k entry)
// to inspect or copy without mutating the slice. Currently a thin
// accessor — kept for API symmetry with the buildFlowGraphGen
// return value.
func (fb *flowBuilderGen) flowBuilderGenObjects() []element.Element {
	return fb.objects
}

// resetLayoutGen clears RelativeMiddlePoint on every activity in oc so that
// Studio Pro re-computes the canvas layout on next open. This is used by the
// RESET LAYOUT microflow/nanoflow option to delegate positioning to Studio Pro
// instead of the flowbuilder's linear algorithm. Recurses into LoopedActivity.
func resetLayoutGen(oc *genMf.MicroflowObjectCollection) {
	if oc == nil {
		return
	}
	type posResetter interface {
		SetRelativeMiddlePoint(string)
	}
	for _, obj := range oc.ObjectsItems() {
		if obj == nil {
			continue
		}
		if r, ok := obj.(posResetter); ok {
			r.SetRelativeMiddlePoint("")
		}
		if looped, ok := obj.(*genMf.LoopedActivity); ok {
			if inner, ok2 := looped.ObjectCollection().(*genMf.MicroflowObjectCollection); ok2 {
				resetLayoutGen(inner)
			}
		}
	}
}
