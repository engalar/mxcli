// SPDX-License-Identifier: Apache-2.0

// Stage 3.2.6.4: pure-AST helpers extracted from the deleted
// cmd_microflows_builder*.go family. None of these functions touch
// sdk/microflows types — they operate on AST or constants — so they
// are kept as plain package-level helpers consumed by the gen
// builders (`flowbuilder_*_gen.go`) and a few other callers.

package executor

import (
	"fmt"
	"strings"

	"github.com/mendixlabs/mxcli/mdl/ast"
)

// defaultLogNodeExpression is the quoted Mendix expression used for the
// log node name when an authored LogStmt does not specify one.
const defaultLogNodeExpression = "'Application'"

// errorExampleDeclareVariable returns an example for declaring a
// variable. Surfaced in error messages from the gen builder when a
// reference targets an undeclared variable.
func errorExampleDeclareVariable(varName string) string {
	cleanName := varName
	if len(varName) > 0 && varName[0] == '$' {
		cleanName = varName[1:]
	}
	return fmt.Sprintf(`    declare $%s Boolean = true;  -- or String, Integer, Decimal, DateTime
    ...
    set $%s = false;`, cleanName, cleanName)
}

// enumSplitCaseValues normalises an EnumSplitCase to a flat string
// slice (handling both Value and Values shapes). Returns nil when the
// case carries no values at all.
func enumSplitCaseValues(c ast.EnumSplitCase) []string {
	if len(c.Values) > 0 {
		return append([]string(nil), c.Values...)
	}
	if c.Value != "" {
		return []string{c.Value}
	}
	return nil
}

// listOperationFieldName extracts the bare attribute / association
// name from a list-operation FIELD expression. Returns ("", false) for
// shapes that aren't a simple identifier or qualified name.
func listOperationFieldName(expr ast.Expression) (string, bool) {
	switch e := expr.(type) {
	case *ast.IdentifierExpr:
		return e.Name, true
	case *ast.QualifiedNameExpr:
		return e.QualifiedName.String(), true
	default:
		return "", false
	}
}

// retrieveXPathConstraint normalises an XPath expression for the
// `database retrieve ... where <expr>` constraint:
//   - converts 3-part enum refs to single-quoted literals;
//   - wraps the constraint in `[...]` if it isn't already.
func retrieveXPathConstraint(expr ast.Expression) string {
	xpath := normalizeXPathEnumRefs(expressionToXPath(expr))
	trimmed := strings.TrimSpace(xpath)
	if strings.HasPrefix(trimmed, "[") && strings.HasSuffix(trimmed, "]") {
		return trimmed
	}
	return "[" + xpath + "]"
}

// getStatementAnnotations extracts the *ast.ActivityAnnotations from
// any microflow statement. Returns nil when the statement has no
// annotations field. Pure switch over AST types.
func getStatementAnnotations(stmt ast.MicroflowStatement) *ast.ActivityAnnotations {
	switch s := stmt.(type) {
	case *ast.DeclareStmt:
		return s.Annotations
	case *ast.InheritanceSplitStmt:
		return s.Annotations
	case *ast.CastObjectStmt:
		return s.Annotations
	case *ast.MfSetStmt:
		return s.Annotations
	case *ast.ReturnStmt:
		return s.Annotations
	case *ast.RaiseErrorStmt:
		return s.Annotations
	case *ast.CreateObjectStmt:
		return s.Annotations
	case *ast.ChangeObjectStmt:
		return s.Annotations
	case *ast.MfCommitStmt:
		return s.Annotations
	case *ast.DeleteObjectStmt:
		return s.Annotations
	case *ast.RollbackStmt:
		return s.Annotations
	case *ast.RetrieveStmt:
		return s.Annotations
	case *ast.IfStmt:
		return s.Annotations
	case *ast.EnumSplitStmt:
		return s.Annotations
	case *ast.LoopStmt:
		return s.Annotations
	case *ast.WhileStmt:
		return s.Annotations
	case *ast.LogStmt:
		return s.Annotations
	case *ast.CallMicroflowStmt:
		return s.Annotations
	case *ast.CallNanoflowStmt:
		return s.Annotations
	case *ast.CallJavaActionStmt:
		return s.Annotations
	case *ast.CallJavaScriptActionStmt:
		return s.Annotations
	case *ast.CallWebServiceStmt:
		return s.Annotations
	case *ast.ExecuteDatabaseQueryStmt:
		return s.Annotations
	case *ast.CallExternalActionStmt:
		return s.Annotations
	case *ast.BreakStmt:
		return s.Annotations
	case *ast.ContinueStmt:
		return s.Annotations
	case *ast.ListOperationStmt:
		return s.Annotations
	case *ast.AggregateListStmt:
		return s.Annotations
	case *ast.CreateListStmt:
		return s.Annotations
	case *ast.AddToListStmt:
		return s.Annotations
	case *ast.RemoveFromListStmt:
		return s.Annotations
	case *ast.ShowPageStmt:
		return s.Annotations
	case *ast.ClosePageStmt:
		return s.Annotations
	case *ast.ShowHomePageStmt:
		return s.Annotations
	case *ast.SynchronizeStmt:
		return s.Annotations
	case *ast.ShowMessageStmt:
		return s.Annotations
	case *ast.DownloadFileStmt:
		return s.Annotations
	case *ast.ValidationFeedbackStmt:
		return s.Annotations
	case *ast.RestCallStmt:
		return s.Annotations
	case *ast.CallWorkflowStmt:
		return s.Annotations
	case *ast.GetWorkflowDataStmt:
		return s.Annotations
	case *ast.GetWorkflowsStmt:
		return s.Annotations
	case *ast.GetWorkflowActivityRecordsStmt:
		return s.Annotations
	case *ast.WorkflowOperationStmt:
		return s.Annotations
	case *ast.SetTaskOutcomeStmt:
		return s.Annotations
	case *ast.OpenUserTaskStmt:
		return s.Annotations
	case *ast.NotifyWorkflowStmt:
		return s.Annotations
	case *ast.OpenWorkflowStmt:
		return s.Annotations
	case *ast.LockWorkflowStmt:
		return s.Annotations
	case *ast.UnlockWorkflowStmt:
		return s.Annotations
	case *ast.GenerateJumpToStmt:
		return s.Annotations
	case *ast.ApplyJumpToStmt:
		return s.Annotations
	default:
		return nil
	}
}

// lastStmtIsReturn reports whether the last statement in stmts is a
// guaranteed terminator (return / raise error / break / continue, or
// an IfStmt / EnumSplitStmt / InheritanceSplitStmt where every branch
// terminates, or a `while true` with a terminating body). Used by the
// gen IF builder to detect when a continuation flow is needed.
func lastStmtIsReturn(stmts []ast.MicroflowStatement) bool {
	if len(stmts) == 0 {
		return false
	}
	return isTerminalStmt(stmts[len(stmts)-1])
}

func isTerminalStmt(stmt ast.MicroflowStatement) bool {
	switch s := stmt.(type) {
	case *ast.ReturnStmt:
		return true
	case *ast.RaiseErrorStmt:
		return true
	case *ast.BreakStmt:
		return true
	case *ast.ContinueStmt:
		return true
	case *ast.IfStmt:
		if len(s.ElseBody) == 0 {
			return false
		}
		return lastStmtIsReturn(s.ThenBody) && lastStmtIsReturn(s.ElseBody)
	case *ast.WhileStmt:
		return isManualWhileTrueCandidate(s)
	case *ast.EnumSplitStmt:
		if len(s.Cases) == 0 {
			return false
		}
		if len(s.ElseBody) > 0 && !lastStmtIsReturn(s.ElseBody) {
			return false
		}
		for _, c := range s.Cases {
			if !lastStmtIsReturn(c.Body) {
				return false
			}
		}
		return true
	case *ast.InheritanceSplitStmt:
		if len(s.Cases) == 0 || len(s.ElseBody) == 0 || !lastStmtIsReturn(s.ElseBody) {
			return false
		}
		for _, c := range s.Cases {
			if !lastStmtIsReturn(c.Body) {
				return false
			}
		}
		return true
	default:
		return false
	}
}

// containsTerminalStmt walks stmts (and recurses into IF / loop bodies)
// looking for a `return` or `raise error` — used to detect bodies that
// may exit early even when not the last statement.
func containsTerminalStmt(stmts []ast.MicroflowStatement) bool {
	for _, stmt := range stmts {
		switch s := stmt.(type) {
		case *ast.ReturnStmt, *ast.RaiseErrorStmt:
			return true
		case *ast.IfStmt:
			if containsTerminalStmt(s.ThenBody) || containsTerminalStmt(s.ElseBody) {
				return true
			}
		case *ast.LoopStmt:
			if containsTerminalStmt(s.Body) {
				return true
			}
		case *ast.WhileStmt:
			if containsTerminalStmt(s.Body) {
				return true
			}
		}
	}
	return false
}

// isManualWhileTrueCandidate reports whether a `while true` body
// contains a continue / terminator that warrants emitting it as a
// manual back-edge instead of the standard LoopedActivity.
func isManualWhileTrueCandidate(s *ast.WhileStmt) bool {
	if s == nil || containsBreakForCurrentLoop(s.Body) || (!containsContinueForCurrentLoop(s.Body) && !containsTerminalStmt(s.Body)) {
		return false
	}
	lit, ok := s.Condition.(*ast.LiteralExpr)
	if !ok || lit.Kind != ast.LiteralBoolean {
		return false
	}
	value, ok := lit.Value.(bool)
	return ok && value
}

func containsBreakForCurrentLoop(stmts []ast.MicroflowStatement) bool {
	for _, stmt := range stmts {
		switch s := stmt.(type) {
		case *ast.BreakStmt:
			return true
		case *ast.IfStmt:
			if containsBreakForCurrentLoop(s.ThenBody) || containsBreakForCurrentLoop(s.ElseBody) {
				return true
			}
		case *ast.LoopStmt, *ast.WhileStmt:
			// A break inside a nested loop exits that nested loop, not this
			// manual while-true back-edge.
			continue
		}
	}
	return false
}

func containsContinueForCurrentLoop(stmts []ast.MicroflowStatement) bool {
	for _, stmt := range stmts {
		switch s := stmt.(type) {
		case *ast.ContinueStmt:
			return true
		case *ast.IfStmt:
			if containsContinueForCurrentLoop(s.ThenBody) || containsContinueForCurrentLoop(s.ElseBody) {
				return true
			}
		case *ast.LoopStmt, *ast.WhileStmt:
			continue
		}
	}
	return false
}
