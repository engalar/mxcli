// SPDX-License-Identifier: Apache-2.0

package visitor

import (
	"github.com/mendixlabs/mxcli/mdl/ast"
	"github.com/mendixlabs/mxcli/mdl/grammar/parser"
)

// ExitCreateWorkflowStatement handles CREATE WORKFLOW statements.
func (b *Builder) ExitCreateWorkflowStatement(ctx *parser.CreateWorkflowStatementContext) {
	names := ctx.AllQualifiedName()
	if len(names) == 0 {
		return
	}

	stmt := &ast.CreateWorkflowStmt{
		Name: buildQualifiedName(names[0]),
	}

	// Parse PARAMETER $Var: Entity
	if ctx.PARAMETER() != nil && ctx.VARIABLE() != nil {
		stmt.ParameterVar = ctx.VARIABLE().GetText()
		// The parameter entity is the second qualified name
		if len(names) > 1 {
			stmt.ParameterEntity = buildQualifiedName(names[1])
		}
	}

	// Parse OVERVIEW PAGE QualifiedName
	overviewPageIdx := -1
	if ctx.OVERVIEW() != nil && ctx.PAGE() != nil {
		// Find the overview page qualified name
		// It's either names[1] or names[2] depending on whether PARAMETER was present
		startIdx := 1
		if ctx.PARAMETER() != nil {
			startIdx = 2
		}
		if len(names) > startIdx {
			stmt.OverviewPage = buildQualifiedName(names[startIdx])
			overviewPageIdx = startIdx
		}
	}
	_ = overviewPageIdx

	// Parse DUE DATE 'expression'
	if ctx.DUE() != nil && ctx.STRING_LITERAL() != nil {
		stmt.DueDate = unquoteString(ctx.STRING_LITERAL().GetText())
	}

	// Parse CREATE OR MODIFY
	createStmt := findParentCreateStatement(ctx)
	if createStmt != nil {
		if createStmt.OR() != nil && (createStmt.MODIFY() != nil || createStmt.REPLACE() != nil) {
			stmt.CreateOrModify = true
		}
	}
	stmt.Documentation = findDocCommentText(ctx)

	// Parse body
	if body := ctx.WorkflowBody(); body != nil {
		stmt.Activities = buildWorkflowBody(body)
	}

	b.statements = append(b.statements, stmt)
}

// buildWorkflowBody converts a workflow body context to activity nodes.
func buildWorkflowBody(ctx parser.IWorkflowBodyContext) []ast.WorkflowActivityNode {
	if ctx == nil {
		return nil
	}
	bodyCtx := ctx.(*parser.WorkflowBodyContext)
	var activities []ast.WorkflowActivityNode

	for _, actCtx := range bodyCtx.AllWorkflowActivityStmt() {
		act := buildWorkflowActivityStmt(actCtx)
		if act != nil {
			activities = append(activities, act)
		}
	}

	return activities
}

// buildWorkflowActivityStmt dispatches to the appropriate builder.
func buildWorkflowActivityStmt(ctx parser.IWorkflowActivityStmtContext) ast.WorkflowActivityNode {
	if ctx == nil {
		return nil
	}
	actCtx := ctx.(*parser.WorkflowActivityStmtContext)

	if ut := actCtx.WorkflowUserTaskStmt(); ut != nil {
		return buildWorkflowUserTask(ut)
	}
	if cm := actCtx.WorkflowCallMicroflowStmt(); cm != nil {
		return buildWorkflowCallMicroflow(cm)
	}
	if cw := actCtx.WorkflowCallWorkflowStmt(); cw != nil {
		return buildWorkflowCallWorkflow(cw)
	}
	if d := actCtx.WorkflowDecisionStmt(); d != nil {
		return buildWorkflowDecision(d)
	}
	if ps := actCtx.WorkflowParallelSplitStmt(); ps != nil {
		return buildWorkflowParallelSplit(ps)
	}
	if jt := actCtx.WorkflowJumpToStmt(); jt != nil {
		return buildWorkflowJumpTo(jt)
	}
	if wt := actCtx.WorkflowWaitForTimerStmt(); wt != nil {
		return buildWorkflowWaitForTimer(wt)
	}
	if wn := actCtx.WorkflowWaitForNotificationStmt(); wn != nil {
		return buildWorkflowWaitForNotification(wn)
	}
	return nil
}

// buildWorkflowUserTask builds a WorkflowUserTaskNode from the grammar context.
func buildWorkflowUserTask(ctx parser.IWorkflowUserTaskStmtContext) *ast.WorkflowUserTaskNode {
	utCtx := ctx.(*parser.WorkflowUserTaskStmtContext)

	node := &ast.WorkflowUserTaskNode{
		Name: utCtx.IDENTIFIER().GetText(),
	}

	// Caption is the first STRING_LITERAL
	allStrings := utCtx.AllSTRING_LITERAL()
	if len(allStrings) > 0 {
		node.Caption = unquoteString(allStrings[0].GetText())
	}

	// Qualified names: PAGE, TARGETING MICROFLOW, ENTITY (in order)
	names := utCtx.AllQualifiedName()
	nameIdx := 0

	if utCtx.PAGE() != nil && nameIdx < len(names) {
		node.Page = buildQualifiedName(names[nameIdx])
		nameIdx++
	}

	if utCtx.MICROFLOW() != nil && nameIdx < len(names) {
		node.Targeting.Kind = "microflow"
		node.Targeting.Microflow = buildQualifiedName(names[nameIdx])
		nameIdx++
	}

	if utCtx.XPATH() != nil && len(allStrings) > 1 {
		node.Targeting.Kind = "xpath"
		node.Targeting.XPath = unquoteString(allStrings[1].GetText())
	}

	if utCtx.ENTITY() != nil && nameIdx < len(names) {
		node.Entity = buildQualifiedName(names[nameIdx])
	}

	// Outcomes
	for _, outcomeCtx := range utCtx.AllWorkflowUserTaskOutcome() {
		outcome := buildWorkflowUserTaskOutcome(outcomeCtx)
		node.Outcomes = append(node.Outcomes, outcome)
	}

	return node
}

// buildWorkflowUserTaskOutcome builds a WorkflowUserTaskOutcomeNode.
func buildWorkflowUserTaskOutcome(ctx parser.IWorkflowUserTaskOutcomeContext) ast.WorkflowUserTaskOutcomeNode {
	outCtx := ctx.(*parser.WorkflowUserTaskOutcomeContext)
	outcome := ast.WorkflowUserTaskOutcomeNode{
		Caption: unquoteString(outCtx.STRING_LITERAL().GetText()),
	}
	if body := outCtx.WorkflowBody(); body != nil {
		outcome.Activities = buildWorkflowBody(body)
	}
	return outcome
}

// buildWorkflowCallMicroflow builds a WorkflowCallMicroflowNode.
func buildWorkflowCallMicroflow(ctx parser.IWorkflowCallMicroflowStmtContext) *ast.WorkflowCallMicroflowNode {
	cmCtx := ctx.(*parser.WorkflowCallMicroflowStmtContext)
	node := &ast.WorkflowCallMicroflowNode{
		Microflow: buildQualifiedName(cmCtx.QualifiedName()),
	}

	if cmCtx.COMMENT() != nil && cmCtx.STRING_LITERAL() != nil {
		node.Caption = unquoteString(cmCtx.STRING_LITERAL().GetText())
	}

	for _, outcomeCtx := range cmCtx.AllWorkflowConditionOutcome() {
		outcome := buildWorkflowConditionOutcome(outcomeCtx)
		node.Outcomes = append(node.Outcomes, outcome)
	}

	return node
}

// buildWorkflowCallWorkflow builds a WorkflowCallWorkflowNode.
func buildWorkflowCallWorkflow(ctx parser.IWorkflowCallWorkflowStmtContext) *ast.WorkflowCallWorkflowNode {
	cwCtx := ctx.(*parser.WorkflowCallWorkflowStmtContext)
	node := &ast.WorkflowCallWorkflowNode{
		Workflow: buildQualifiedName(cwCtx.QualifiedName()),
	}

	if cwCtx.COMMENT() != nil && cwCtx.STRING_LITERAL() != nil {
		node.Caption = unquoteString(cwCtx.STRING_LITERAL().GetText())
	}

	return node
}

// buildWorkflowDecision builds a WorkflowDecisionNode.
func buildWorkflowDecision(ctx parser.IWorkflowDecisionStmtContext) *ast.WorkflowDecisionNode {
	dCtx := ctx.(*parser.WorkflowDecisionStmtContext)
	node := &ast.WorkflowDecisionNode{}

	allStrings := dCtx.AllSTRING_LITERAL()
	stringIdx := 0

	// First STRING_LITERAL is the expression (if present and COMMENT is not present or expression comes first)
	if len(allStrings) > 0 && dCtx.COMMENT() == nil {
		// All strings are expression
		node.Expression = unquoteString(allStrings[0].GetText())
		stringIdx = 1
	} else if len(allStrings) > 0 && dCtx.COMMENT() != nil {
		// Distinguish expression from comment
		if len(allStrings) >= 2 {
			node.Expression = unquoteString(allStrings[0].GetText())
			node.Caption = unquoteString(allStrings[1].GetText())
		} else {
			// Only one string with COMMENT - it's the caption
			node.Caption = unquoteString(allStrings[0].GetText())
		}
		stringIdx = len(allStrings)
	}
	_ = stringIdx

	for _, outcomeCtx := range dCtx.AllWorkflowConditionOutcome() {
		outcome := buildWorkflowConditionOutcome(outcomeCtx)
		node.Outcomes = append(node.Outcomes, outcome)
	}

	return node
}

// buildWorkflowConditionOutcome builds a WorkflowConditionOutcomeNode.
func buildWorkflowConditionOutcome(ctx parser.IWorkflowConditionOutcomeContext) ast.WorkflowConditionOutcomeNode {
	coCtx := ctx.(*parser.WorkflowConditionOutcomeContext)
	outcome := ast.WorkflowConditionOutcomeNode{}

	if coCtx.TRUE() != nil {
		outcome.Value = "True"
	} else if coCtx.FALSE() != nil {
		outcome.Value = "False"
	} else if coCtx.DEFAULT() != nil {
		outcome.Value = "Default"
	} else if coCtx.STRING_LITERAL() != nil {
		outcome.Value = unquoteString(coCtx.STRING_LITERAL().GetText())
	}

	if body := coCtx.WorkflowBody(); body != nil {
		outcome.Activities = buildWorkflowBody(body)
	}

	return outcome
}

// buildWorkflowParallelSplit builds a WorkflowParallelSplitNode.
func buildWorkflowParallelSplit(ctx parser.IWorkflowParallelSplitStmtContext) *ast.WorkflowParallelSplitNode {
	psCtx := ctx.(*parser.WorkflowParallelSplitStmtContext)
	node := &ast.WorkflowParallelSplitNode{}

	if psCtx.COMMENT() != nil && psCtx.STRING_LITERAL() != nil {
		node.Caption = unquoteString(psCtx.STRING_LITERAL().GetText())
	}

	for _, pathCtx := range psCtx.AllWorkflowParallelPath() {
		path := buildWorkflowParallelPath(pathCtx)
		node.Paths = append(node.Paths, path)
	}

	return node
}

// buildWorkflowParallelPath builds a WorkflowParallelPathNode.
func buildWorkflowParallelPath(ctx parser.IWorkflowParallelPathContext) ast.WorkflowParallelPathNode {
	ppCtx := ctx.(*parser.WorkflowParallelPathContext)
	path := ast.WorkflowParallelPathNode{}

	if ppCtx.NUMBER_LITERAL() != nil {
		path.PathNumber = parseInt(ppCtx.NUMBER_LITERAL().GetText())
	}

	if body := ppCtx.WorkflowBody(); body != nil {
		path.Activities = buildWorkflowBody(body)
	}

	return path
}

// buildWorkflowJumpTo builds a WorkflowJumpToNode.
func buildWorkflowJumpTo(ctx parser.IWorkflowJumpToStmtContext) *ast.WorkflowJumpToNode {
	jtCtx := ctx.(*parser.WorkflowJumpToStmtContext)
	node := &ast.WorkflowJumpToNode{
		Target: jtCtx.IDENTIFIER().GetText(),
	}

	if jtCtx.COMMENT() != nil && jtCtx.STRING_LITERAL() != nil {
		node.Caption = unquoteString(jtCtx.STRING_LITERAL().GetText())
	}

	return node
}

// buildWorkflowWaitForTimer builds a WorkflowWaitForTimerNode.
func buildWorkflowWaitForTimer(ctx parser.IWorkflowWaitForTimerStmtContext) *ast.WorkflowWaitForTimerNode {
	wtCtx := ctx.(*parser.WorkflowWaitForTimerStmtContext)
	node := &ast.WorkflowWaitForTimerNode{}

	allStrings := wtCtx.AllSTRING_LITERAL()
	if len(allStrings) > 0 && wtCtx.COMMENT() == nil {
		node.DelayExpression = unquoteString(allStrings[0].GetText())
	} else if len(allStrings) >= 2 && wtCtx.COMMENT() != nil {
		node.DelayExpression = unquoteString(allStrings[0].GetText())
		node.Caption = unquoteString(allStrings[1].GetText())
	} else if len(allStrings) == 1 && wtCtx.COMMENT() != nil {
		node.Caption = unquoteString(allStrings[0].GetText())
	}

	return node
}

// buildWorkflowWaitForNotification builds a WorkflowWaitForNotificationNode.
func buildWorkflowWaitForNotification(ctx parser.IWorkflowWaitForNotificationStmtContext) *ast.WorkflowWaitForNotificationNode {
	wnCtx := ctx.(*parser.WorkflowWaitForNotificationStmtContext)
	node := &ast.WorkflowWaitForNotificationNode{}

	if wnCtx.COMMENT() != nil && wnCtx.STRING_LITERAL() != nil {
		node.Caption = unquoteString(wnCtx.STRING_LITERAL().GetText())
	}

	return node
}

// parseInt parses a string as an integer, returning 0 on failure.
func parseInt(s string) int {
	n := 0
	for _, c := range s {
		if c >= '0' && c <= '9' {
			n = n*10 + int(c-'0')
		}
	}
	return n
}
