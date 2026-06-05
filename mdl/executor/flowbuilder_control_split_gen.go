// SPDX-License-Identifier: Apache-2.0

// Stage 3.2.3.h4 — gen-typed EnumSplit + InheritanceSplit adders.
//
// Both wrap multi-branch splits. EnumSplit emits an ExclusiveSplit
// with an ExpressionSplitCondition (legacy parity); InheritanceSplit
// emits an InheritanceSplit element with the variable name.
//
// Branch wiring (mirrors h2 IF):
//
//   1. emit split element
//   2. for each case: emit branch body via emitBranchBodyGen with the
//      case value as the flow label
//   3. for each case that doesn't terminate: connect last activity
//      (or split→merge directly when body is empty) to merge
//   4. emit ExclusiveMerge only when at least one branch flows through;
//      skip when all branches return
//   5. when else body is present, emit it as the implicit "default" case
//
// Out of scope (legacy edge cases tracked as TODOs):
//
//   - per-branch @anchor overrides
//   - retry-loop pattern detection
//   - addStructuredInheritanceSplit fallback (richer geometry)
//
// addStructuredInheritanceSplit in the legacy path is the
// inheritance equivalent of the case-cases pattern; the bare
// InheritanceSplitStmt with cases routes through here directly.

package executor

import (
	"github.com/mendixlabs/mxcli/mdl/ast"
	"github.com/mendixlabs/mxcli/modelsdk/element"
	genMf "github.com/mendixlabs/mxcli/modelsdk/gen/microflows"
)

// addEnumSplitGen emits a `case $Var when X then ... else ... end case;`
// activity. Returns the split's ID.
func (fb *flowBuilderGen) addEnumSplitGen(s *ast.EnumSplitStmt) element.ID {
	if fb.measurer == nil {
		fb.measurer = &layoutMeasurer{varTypes: fb.varTypes}
	}

	splitX := fb.posX
	centerY := fb.posY
	caption := "$" + s.Variable

	cond := genMf.NewExpressionSplitCondition()
	assignFreshID(cond)
	cond.SetExpression(caption)

	split := genMf.NewExclusiveSplit()
	splitID := assignFreshID(split)
	split.SetRelativeMiddlePoint(layoutPos(splitX, centerY))
	split.SetSize(layoutSize(SplitWidth, SplitHeight))
	split.SetCaption(caption)
	split.SetSplitCondition(cond)
	split.SetErrorHandlingType(fb.ehTypeGen(nil))
	fb.objects = append(fb.objects, split)

	hasElse := len(s.ElseBody) > 0

	// Measure branch widths for merge placement.
	branchWidth := 0
	for _, c := range s.Cases {
		w := fb.measurer.measureStatements(c.Body).Width
		if w > branchWidth {
			branchWidth = w
		}
	}
	if hasElse {
		w := fb.measurer.measureStatements(s.ElseBody).Width
		if w > branchWidth {
			branchWidth = w
		}
	}
	if branchWidth == 0 {
		branchWidth = HorizontalSpacing / 2
	}

	mergeX := splitX + SplitWidth + HorizontalSpacing/2 + branchWidth + HorizontalSpacing/2

	var merge *genMf.ExclusiveMerge
	var mergeID element.ID
	ensureMerge := func() element.ID {
		if merge == nil {
			merge = genMf.NewExclusiveMerge()
			mergeID = assignFreshID(merge)
			merge.SetRelativeMiddlePoint(layoutPos(mergeX, centerY))
			merge.SetSize(layoutSize(MergeSize, MergeSize))
			fb.objects = append(fb.objects, merge)
		}
		return mergeID
	}

	allBranchesReturn := len(s.Cases) > 0 || hasElse
	branchY := centerY

	// Emit each case branch. First case (i=0) is on the main line
	// (horizontal flow from split); subsequent cases fan out below
	// (downward flow from split).
	for i, c := range s.Cases {
		fb.posX = splitX + SplitWidth + HorizontalSpacing/2
		fb.posY = branchY + i*VerticalSpacing
		caseLast := fb.emitBranchBodyGen(c.Body, splitID, c.Value, i > 0)
		caseReturns := lastStmtIsReturn(c.Body)
		if !caseReturns {
			allBranchesReturn = false
			if caseLast != "" {
				fb.flows = append(fb.flows, newHorizontalFlowGen(caseLast, ensureMerge()))
			} else {
				fb.flows = append(fb.flows, newHorizontalFlowWithEnumCaseGen(splitID, ensureMerge(), c.Value))
			}
		}
	}

	// Emit else branch. It sits below all named cases so isBelow=true
	// whenever there is at least one named case above it.
	if hasElse {
		fb.posX = splitX + SplitWidth + HorizontalSpacing/2
		fb.posY = branchY + len(s.Cases)*VerticalSpacing
		// Else uses an empty case label — gen describer treats unlabelled as default.
		elseLast := fb.emitBranchBodyGen(s.ElseBody, splitID, "", len(s.Cases) > 0)
		elseReturns := lastStmtIsReturn(s.ElseBody)
		if !elseReturns {
			allBranchesReturn = false
			if elseLast != "" {
				fb.flows = append(fb.flows, newHorizontalFlowGen(elseLast, ensureMerge()))
			} else {
				fb.flows = append(fb.flows, newHorizontalFlowGen(splitID, ensureMerge()))
			}
		}
	}

	if merge != nil {
		fb.posX = mergeX + MergeSize/2 + ActivityWidth/2 + BranchGap
		fb.posY = centerY
		fb.nextConnectionPoint = mergeID
	}
	if allBranchesReturn {
		fb.endsWithReturn = true
	}

	return splitID
}

// addInheritanceSplitGen emits either a bare `InheritanceSplit` element
// (legacy fast-path for no-cases / no-else) or a multi-branch
// inheritance split (cases each labelled with their entity QN).
//
// Bare path (no cases, no else): emit minimal InheritanceSplit at the
// current cursor and advance posX by spacing — matches the legacy
// addInheritanceSplit fast-path.
func (fb *flowBuilderGen) addInheritanceSplitGen(s *ast.InheritanceSplitStmt) element.ID {
	if len(s.Cases) == 0 && len(s.ElseBody) == 0 {
		split := genMf.NewInheritanceSplit()
		splitID := assignFreshID(split)
		split.SetRelativeMiddlePoint(layoutPos(fb.posX, fb.posY))
		split.SetSize(layoutSize(ActivityWidth, ActivityHeight))
		split.SetSplitVariableName(s.Variable)
		fb.objects = append(fb.objects, split)
		fb.posX += fb.spacing
		return splitID
	}
	return fb.addStructuredInheritanceSplitGen(s)
}

// addStructuredInheritanceSplitGen emits an inheritance split with
// case branches. Mirrors the EnumSplit shape but uses InheritanceSplit
// + InheritanceCase for branch labels.
func (fb *flowBuilderGen) addStructuredInheritanceSplitGen(s *ast.InheritanceSplitStmt) element.ID {
	if fb.measurer == nil {
		fb.measurer = &layoutMeasurer{varTypes: fb.varTypes}
	}

	splitX := fb.posX
	centerY := fb.posY

	split := genMf.NewInheritanceSplit()
	splitID := assignFreshID(split)
	split.SetRelativeMiddlePoint(layoutPos(splitX, centerY))
	split.SetSize(layoutSize(SplitWidth, SplitHeight))
	split.SetSplitVariableName(s.Variable)
	fb.objects = append(fb.objects, split)

	hasElse := len(s.ElseBody) > 0

	branchWidth := 0
	for _, c := range s.Cases {
		w := fb.measurer.measureStatements(c.Body).Width
		if w > branchWidth {
			branchWidth = w
		}
	}
	if hasElse {
		w := fb.measurer.measureStatements(s.ElseBody).Width
		if w > branchWidth {
			branchWidth = w
		}
	}
	if branchWidth == 0 {
		branchWidth = HorizontalSpacing / 2
	}

	mergeX := splitX + SplitWidth + HorizontalSpacing/2 + branchWidth + HorizontalSpacing/2

	var merge *genMf.ExclusiveMerge
	var mergeID element.ID
	ensureMerge := func() element.ID {
		if merge == nil {
			merge = genMf.NewExclusiveMerge()
			mergeID = assignFreshID(merge)
			merge.SetRelativeMiddlePoint(layoutPos(mergeX, centerY))
			merge.SetSize(layoutSize(MergeSize, MergeSize))
			fb.objects = append(fb.objects, merge)
		}
		return mergeID
	}

	allBranchesReturn := len(s.Cases) > 0 || hasElse
	branchY := centerY

	for i, c := range s.Cases {
		entityQN := c.Entity.Module + "." + c.Entity.Name
		fb.posX = splitX + SplitWidth + HorizontalSpacing/2
		fb.posY = branchY + i*VerticalSpacing
		caseLast := fb.emitBranchBodyInheritanceGen(c.Body, splitID, entityQN)
		caseReturns := lastStmtIsReturn(c.Body)
		if !caseReturns {
			allBranchesReturn = false
			if caseLast != "" {
				fb.flows = append(fb.flows, newHorizontalFlowGen(caseLast, ensureMerge()))
			} else {
				fb.flows = append(fb.flows, newHorizontalFlowWithInheritanceCaseGen(splitID, ensureMerge(), entityQN))
			}
		}
	}

	if hasElse {
		fb.posX = splitX + SplitWidth + HorizontalSpacing/2
		fb.posY = branchY + len(s.Cases)*VerticalSpacing
		// Else branch must use InheritanceCase Value="" (empty QN) — NOT NoCase.
		// Mendix's Object Type Decision requires InheritanceCase for all outgoing flows,
		// including the fall-through (else). An empty Value="" matches all unspecified
		// subtypes (System.User, Administration.Account, null/empty). Using NoCase here
		// causes CE0089 "(empty) not configured" and CE0090 for known subtypes.
		elseLast := fb.emitBranchBodyInheritanceGen(s.ElseBody, splitID, "")
		elseReturns := lastStmtIsReturn(s.ElseBody)
		if !elseReturns {
			allBranchesReturn = false
			if elseLast != "" {
				fb.flows = append(fb.flows, newHorizontalFlowGen(elseLast, ensureMerge()))
			} else {
				fb.flows = append(fb.flows, newHorizontalFlowWithInheritanceCaseGen(splitID, ensureMerge(), ""))
			}
		}
	}

	if merge != nil {
		fb.posX = mergeX + MergeSize/2 + ActivityWidth/2 + BranchGap
		fb.posY = centerY
		fb.nextConnectionPoint = mergeID
	}
	if allBranchesReturn {
		fb.endsWithReturn = true
	}

	return splitID
}

// emitBranchBodyInheritanceGen is the inheritance-split-flavoured
// branch emitter. Same shape as emitBranchBodyGen (h2) but the
// split→firstActivity flow carries an InheritanceCase label.
func (fb *flowBuilderGen) emitBranchBodyInheritanceGen(body []ast.MicroflowStatement, splitID element.ID, entityQN string) element.ID {
	var lastID element.ID
	for _, stmt := range body {
		actID := fb.addStatementGen(stmt)
		if actID == "" {
			continue
		}
		if lastID == "" {
			fb.flows = append(fb.flows, newHorizontalFlowWithInheritanceCaseGen(splitID, actID, entityQN))
		} else {
			fb.flows = append(fb.flows, newHorizontalFlowGen(lastID, actID))
		}
		if fb.nextConnectionPoint != "" {
			lastID = fb.nextConnectionPoint
			fb.nextConnectionPoint = ""
		} else {
			lastID = actID
		}
	}
	return lastID
}
