// SPDX-License-Identifier: Apache-2.0

// Stage 3.2.3.h2 — gen-typed addIfStatementGen.
//
// Minimal-viable IF adder: covers the dominant fresh-author shapes
//
//   - then-only `if X then ... end if;`
//   - then + else
//   - both branches return → no merge
//   - then returns + else continues → only else flows into merge
//   - else returns + then continues → only then flows into merge
//
// Out of scope for h2 (tracked as TODOs):
//
//   - retry-loop pattern detection / wiring
//   - per-branch @anchor overrides (TrueBranchAnchor / FalseBranchAnchor)
//   - @position annotation on the split itself (handled by dispatcher
//     wrapper; per-branch annotations are dropped for now)
//   - empty-custom-error-handler routing into ELSE branch
//     (`pendingElseErrorOrigin` legacy logic)
//   - case-on-bool dispatch (boolean cases use simple flow labels;
//     bool-as-enum case dispatch lands when EnumSplit ships in h4)
//
// The legacy addIfStatement is ~500 LoC because it weaves several
// flow-anchor / pending-error-handler / merge-skipping interactions
// together. The minimal version below clocks in at ~200 LoC and
// covers fresh-author MDL paths; describe→exec roundtrip of complex
// edge cases will need extensions but the dispatcher recursion loop
// (the structural barrier h3/h4 need) is fully exercised.

package executor

import (
	"github.com/mendixlabs/mxcli/mdl/ast"
	"github.com/mendixlabs/mxcli/modelsdk/element"
	genMf "github.com/mendixlabs/mxcli/modelsdk/gen/microflows"
)

// addIfStatementGen emits an ExclusiveSplit + branch activities +
// (optional) ExclusiveMerge for an `if X then ... [else ...] end if;`
// statement.
//
// Returns the split's ID (the IF's "entry point"). When both branches
// terminate, fb.endsWithReturn is set so the parent flow-graph driver
// knows there's no continuation. When at least one branch flows
// through, fb.nextConnectionPoint is set to the merge ID so the next
// statement's inbound flow originates from the merge instead of the
// split.
func (fb *flowBuilderGen) addIfStatementGen(s *ast.IfStmt) element.ID {
	if fb.measurer == nil {
		fb.measurer = &layoutMeasurer{varTypes: fb.varTypes}
	}

	// Branch geometry — measure widths so the merge lands beyond the
	// longest branch.
	thenBounds := fb.measurer.measureStatements(s.ThenBody)
	elseBounds := fb.measurer.measureStatements(s.ElseBody)
	branchWidth := thenBounds.Width
	if elseBounds.Width > branchWidth {
		branchWidth = elseBounds.Width
	}
	if branchWidth == 0 {
		branchWidth = HorizontalSpacing / 2
	}

	thenReturns := lastStmtIsReturn(s.ThenBody)
	hasElseBody := s.HasElse || len(s.ElseBody) > 0
	elseReturns := hasElseBody && lastStmtIsReturn(s.ElseBody)
	bothReturn := hasElseBody && thenReturns && elseReturns

	splitX := fb.posX
	centerY := fb.posY

	// ── Build the split + condition ──
	caption := fb.exprToString(s.Condition)
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

	// Merge position (only created lazily when a branch flows through).
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

	savedEndsWithReturn := fb.endsWithReturn
	allBranchesReturn := bothReturn || (!hasElseBody && false)

	// ── THEN branch ──
	//
	// IF with ELSE:    TRUE  path = horizontal (main line at centerY)
	// IF without ELSE: TRUE  path = below main line (centerY + ActivityHeight + BranchGap)
	//                  FALSE path = straight to merge (main line)
	fb.posX = splitX + SplitWidth + HorizontalSpacing/2
	if hasElseBody {
		fb.posY = centerY
	} else {
		fb.posY = centerY + ActivityHeight + BranchGap
	}
	thenLast := fb.emitBranchBodyGen(s.ThenBody, splitID, "true", !hasElseBody)
	if !thenReturns {
		// Connect last-emitted activity to merge.
		if thenLast != "" {
			fb.flows = append(fb.flows, newHorizontalFlowGen(thenLast, ensureMerge()))
		} else {
			// Empty THEN body — split flows directly to merge.
			if hasElseBody {
				fb.flows = append(fb.flows, newHorizontalFlowWithCaseGen(splitID, ensureMerge(), "true"))
			} else {
				fb.flows = append(fb.flows, newDownwardFlowWithCaseGen(splitID, ensureMerge(), "true"))
			}
		}
	}

	// ── ELSE branch ──
	// Place ELSE below the THEN body's measured height so nested IFs
	// inside THEN don't overlap with the ELSE branch.
	if hasElseBody {
		thenBoundsForElse := fb.measurer.measureStatements(s.ThenBody)
		thenHeightForElse := thenBoundsForElse.Height
		if thenHeightForElse < ActivityHeight {
			thenHeightForElse = ActivityHeight
		}
		fb.posX = splitX + SplitWidth + HorizontalSpacing/2
		fb.posY = centerY + thenHeightForElse + BranchGap
		elseLast := fb.emitBranchBodyGen(s.ElseBody, splitID, "false", true)
		if !elseReturns {
			if elseLast != "" {
				fb.flows = append(fb.flows, newHorizontalFlowGen(elseLast, ensureMerge()))
			} else {
				fb.flows = append(fb.flows, newDownwardFlowWithCaseGen(splitID, ensureMerge(), "false"))
			}
		}
	} else {
		if thenReturns {
			// Then terminates, no else: skip the merge entirely.
			// The false branch flows directly from split to the next activity.
			// Set nextConnectionPoint/Case so buildFlowGraphGen wires the
			// split→nextActivity flow with the correct "false" case label.
			fb.posX = mergeX
			fb.posY = centerY
			fb.nextConnectionPoint = splitID
			fb.nextConnectionCase = "false"
		} else {
			// No else, then continues — split's "false" branch flows to merge.
			fb.flows = append(fb.flows, newHorizontalFlowWithCaseGen(splitID, ensureMerge(), "false"))
		}
	}

	// ── Restore main cursor for next statement ──
	// Use MergeSize/2 + ActivityWidth/2 + BranchGap so the first activity
	// after the merge has a BranchGap edge-to-edge gap from the merge node.
	if merge != nil {
		fb.posX = mergeX + MergeSize/2 + ActivityWidth/2 + BranchGap
		fb.posY = centerY
		fb.nextConnectionPoint = mergeID
	}
	fb.endsWithReturn = savedEndsWithReturn
	if allBranchesReturn {
		fb.endsWithReturn = true
	}

	return splitID
}

// emitBranchBodyGen iterates a branch body, dispatching each statement
// through addStatementGen and threading the chained activities. The
// first activity gets a labelled flow from the split (case "true" or
// "false"); subsequent activities get plain horizontal flows.
//
// isBelow must be true when the branch is placed below the split's Y
// position — this causes the split→first-activity flow to use a
// downward anchor (AnchorBottom→AnchorLeft) instead of a horizontal
// one (AnchorRight→AnchorLeft).
//
// Returns the ID of the last emitted activity, or "" when the body
// is empty (caller decides whether to wire split→merge directly).
func (fb *flowBuilderGen) emitBranchBodyGen(body []ast.MicroflowStatement, splitID element.ID, caseValue string, isBelow bool) element.ID {
	var lastID element.ID
	for _, stmt := range body {
		actID := fb.addStatementGen(stmt)
		if actID == "" {
			continue
		}
		if lastID == "" {
			// First activity in branch — connect from split with case label.
			if isBelow {
				fb.flows = append(fb.flows, newDownwardFlowWithCaseGen(splitID, actID, caseValue))
			} else {
				fb.flows = append(fb.flows, newHorizontalFlowWithCaseGen(splitID, actID, caseValue))
			}
		} else {
			// Subsequent activities — plain horizontal chain.
			fb.flows = append(fb.flows, newHorizontalFlowGen(lastID, actID))
		}
		// Compound statements (nested IF/loop) advertise their merge via
		// nextConnectionPoint; consume so subsequent flows originate from
		// the right place.
		if fb.nextConnectionPoint != "" {
			lastID = fb.nextConnectionPoint
			fb.nextConnectionPoint = ""
		} else {
			lastID = actID
		}
	}
	return lastID
}
