// SPDX-License-Identifier: Apache-2.0

// Stage 3.2.3.h3 — gen-typed Loop + While adders.
//
// Both wrap a `*genMf.LoopedActivity` with a different LoopSource:
//
//   Loop  — `loop $i in $List ... end loop;`
//           LoopSource = *IterableList{ListVariableName, VariableName}
//   While — `while X ... end while;`
//           LoopSource = *WhileLoopCondition{WhileExpression}
//
// Body activities live in the loop's nested ObjectCollection. Internal
// flows go to the parent's flows list (top-level), not inside the
// loop's ObjectCollection — matches Mendix BSON layout where Studio
// Pro reads all flows at the microflow level.
//
// Out of scope for h3 (tracked as TODOs):
//
//   - manual while-true pattern detection (legacy
//     addManualWhileTrueStatement) — fresh-author MDL doesn't author
//     this directly; describer→exec roundtrip needs it later
//   - per-loop @anchor(iterator: ...) annotation surface (legacy
//     parses but doesn't emit a SequenceFlow either, so deferred)
//   - error-handler interaction in the body — the simplified
//     emitBranchBodyGen reuse handles plain bodies; complex EH inside
//     loops will need the same extension as IF (h2)

package executor

import (
	"strings"

	"github.com/mendixlabs/mxcli/mdl/ast"
	"github.com/mendixlabs/mxcli/modelsdk/element"
	genMf "github.com/mendixlabs/mxcli/modelsdk/gen/microflows"
)

// addLoopStatementGen emits a `loop $i in $List ... end loop;`
// LoopedActivity with body activities in a nested ObjectCollection.
// Returns the loop activity's ID. Reports a CE0111 validation error
// when the loop variable name collides with an existing var.
func (fb *flowBuilderGen) addLoopStatementGen(s *ast.LoopStmt) element.ID {
	if fb.measurer == nil {
		fb.measurer = &layoutMeasurer{varTypes: fb.varTypes}
	}

	// Duplicate-loop-var guard before mutating varTypes.
	if _, exists := fb.varTypes[s.LoopVariable]; exists {
		hint := suggestLoopVarName(s.LoopVariable, s.ListVariable, fb.varTypes)
		fb.addError("loop variable '$%s' is already declared in this scope (CE0111)\nHint: %s",
			s.LoopVariable, hint)
		return ""
	}

	// Register loop variable with element type derived from list.
	if fb.varTypes != nil {
		listType := fb.varTypes[s.ListVariable]
		if after, ok := strings.CutPrefix(listType, "List of "); ok {
			fb.varTypes[s.LoopVariable] = after
		}
	}

	src := genMf.NewIterableList()
	assignFreshID(src)
	src.SetListVariableName(s.ListVariable)
	src.SetVariableName(s.LoopVariable)

	return fb.emitLoopedActivityGen(s.Body, src)
}

// addWhileStatementGen emits a `while X ... end while;` LoopedActivity
// with the WhileLoopCondition source carrying the condition expression.
func (fb *flowBuilderGen) addWhileStatementGen(s *ast.WhileStmt) element.ID {
	if fb.measurer == nil {
		fb.measurer = &layoutMeasurer{varTypes: fb.varTypes}
	}

	expr := fb.exprToString(s.Condition)

	src := genMf.NewWhileLoopCondition()
	assignFreshID(src)
	src.SetWhileExpression(expr)

	return fb.emitLoopedActivityGen(s.Body, src)
}

// emitLoopedActivityGen is the shared LoopedActivity emission helper.
// Builds the body inside a child flowBuilderGen (so var scopes stay
// loop-local), wraps the body activities into a MicroflowObjectCollection,
// and pushes internal flows up to the parent (top-level placement).
func (fb *flowBuilderGen) emitLoopedActivityGen(body []ast.MicroflowStatement, src element.Element) element.ID {
	bodyBounds := fb.measurer.measureStatements(body)

	loopWidth := bodyBounds.Width + 2*LoopPadding + IteratorSpace
	if loopWidth < MinLoopWidth {
		loopWidth = MinLoopWidth
	}
	loopHeight := bodyBounds.Height + 2*LoopPadding
	if loopHeight < MinLoopHeight {
		loopHeight = MinLoopHeight
	}

	innerStartX := LoopPadding + IteratorSpace
	innerStartY := loopHeight / 2

	loopLeftX := fb.posX
	loopCenterX := loopLeftX + loopWidth/2

	// Child builder for the body — clones varTypes/declaredVars so
	// loop-local declarations don't leak. Other flowBuilderGen state
	// (backend, hierarchy, repos, measurer) is shared.
	loopBuilder := &flowBuilderGen{
		posX:           innerStartX,
		posY:           innerStartY,
		baseY:          innerStartY,
		spacing:        HorizontalSpacing,
		varTypes:       cloneStringMap(fb.varTypes),
		declaredVars:   cloneStringMap(fb.declaredVars),
		measurer:       fb.measurer,
		backend:        fb.backend,
		microflowsRepo: fb.microflowsRepo,
		nanoflowsRepo:  fb.nanoflowsRepo,
		hierarchy:      fb.hierarchy,
		restServices:   fb.restServices,
		isNanoflow:     fb.isNanoflow,
	}

	// Process body statements + thread chain flows.
	var lastBodyID element.ID
	for _, stmt := range body {
		actID := loopBuilder.addStatementGen(stmt)
		if actID == "" {
			continue
		}
		if lastBodyID != "" {
			loopBuilder.flows = append(loopBuilder.flows, newHorizontalFlowGen(lastBodyID, actID))
		}
		if loopBuilder.nextConnectionPoint != "" {
			lastBodyID = loopBuilder.nextConnectionPoint
			loopBuilder.nextConnectionPoint = ""
		} else {
			lastBodyID = actID
		}
	}

	// Build the LoopedActivity.
	oc := genMf.NewMicroflowObjectCollection()
	assignFreshID(oc)
	for _, obj := range loopBuilder.objects {
		oc.AddObjects(obj)
	}

	loop := genMf.NewLoopedActivity()
	loopID := assignFreshID(loop)
	loop.SetRelativeMiddlePoint(layoutPos(loopCenterX, fb.posY))
	loop.SetSize(layoutSize(loopWidth, loopHeight))
	loop.SetLoopSource(src)
	loop.SetObjectCollection(oc)
	loop.SetErrorHandlingType(fb.ehTypeGen(nil))

	fb.objects = append(fb.objects, loop)

	// Promote internal flows + annotation flows to the parent (top-level).
	fb.flows = append(fb.flows, loopBuilder.flows...)
	fb.annotationFlows = append(fb.annotationFlows, loopBuilder.annotationFlows...)

	fb.posX = loopLeftX + loopWidth + HorizontalSpacing

	return loopID
}
