// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/mendixlabs/mxcli/mdl/ast"
	"github.com/mendixlabs/mxcli/mdl/exprcheck/adapters"
	"github.com/mendixlabs/mxcli/mdl/linter"
)

var (
	reUppercaseAND    = regexp.MustCompile(`\bAND\b`)
	reQuotedAssocName = regexp.MustCompile(`"[^"]+"[.]"[^"]+"`)
	reCurrentUserPath = regexp.MustCompile(`\[%CurrentUser%\]/[A-Za-z][A-Za-z0-9_]*\.[A-Za-z]`)
)

// ValidateMicroflow checks a microflow for common issues that don't require a project connection.
// Returns a list of structured violations with rule IDs.
func ValidateMicroflow(stmt *ast.CreateMicroflowStmt) []linter.Violation {
	v := &microflowValidator{
		mfName:     stmt.Name.String(),
		returnType: stmt.ReturnType,
	}
	// Validate parameter names and entity references.
	for _, p := range stmt.Parameters {
		// Reject parameter names that include the '$' prefix in the declaration.
		// The '$' is reference-site syntax only; including it in the declaration
		// produces a parameter literally named "$Foo" and breaks every later
		// reference like $Foo, while masking the real problem with cryptic
		// "unknown variable" errors deep in microflow validation.
		if strings.HasPrefix(p.Name, "$") {
			bare := strings.TrimPrefix(p.Name, "$")
			v.addViolation("MDL012", linter.SeverityError,
				fmt.Sprintf("parameter %q must not include '$' prefix in declaration", p.Name),
				fmt.Sprintf("declare as %q; reference it inside the microflow body as $%s", bare, bare))
		}
		// Reject bare entity names without module prefix.
		if p.Type.EntityRef != nil && p.Type.EntityRef.Module == "" {
			v.addViolation("MDL008", linter.SeverityError,
				fmt.Sprintf("parameter '$%s': entity type '%s' is missing module prefix",
					p.Name, p.Type.EntityRef.Name),
				fmt.Sprintf("Use a qualified name like 'Module.%s' or 'System.%s'",
					p.Type.EntityRef.Name, p.Type.EntityRef.Name))
		}
	}
	v.validate(stmt.Body)
	res := adapters.NewCheckAdapter(nil).CheckMicroflow(stmt)
	v.violations = append(v.violations, res.AsViolations()...)
	return v.violations
}

// microflowValidator holds state for validating a single microflow.
type microflowValidator struct {
	mfName        string
	returnType    *ast.MicroflowReturnType // nil = void
	violations    []linter.Violation
	loopDepth     int             // Track nesting depth inside loops
	emptyListVars map[string]bool // List variables declared empty and never populated
}

func (v *microflowValidator) addViolation(ruleID string, severity linter.Severity, message, suggestion string) {
	v.violations = append(v.violations, linter.Violation{
		RuleID:   ruleID,
		Severity: severity,
		Message:  message,
		Location: linter.Location{
			DocumentType: "microflow",
			DocumentName: v.mfName,
		},
		Suggestion: suggestion,
	})
}

// validate runs all checks on the microflow body.
func (v *microflowValidator) validate(body []ast.MicroflowStatement) {
	// Walk the body for per-statement checks (validation feedback, return value checks)
	v.emptyListVars = make(map[string]bool)
	v.walkBody(body)

	// Check 5: missing RETURN on non-void microflow paths
	if v.returnType != nil && v.returnType.Type.Kind != ast.TypeVoid {
		if !bodyReturns(body) {
			v.addViolation("MDL003", linter.SeverityError,
				fmt.Sprintf("microflow returns %s but not all code paths have a return statement",
					returnTypeString(v.returnType)),
				"Add return statements to all code paths")
		}
	}

	// MDL017: RETURNS Integer + /id in body → return type should be Long
	if v.returnType != nil && v.returnType.Type.Kind == ast.TypeInteger {
		if bodyContainsIdPath(body) {
			v.addViolation("MDL017", linter.SeverityError,
				fmt.Sprintf("microflow returns Integer but body accesses '/id' which returns Long — this causes CE0117 at runtime."),
				"Change RETURNS Integer to RETURNS Long (Mendix object IDs are Long). Alternatively use a Java Action like GetObjectId.")
		}
	}

	// Check 3: variable scope — detect variables declared inside branches but used after
	v.checkBranchScoping(body)
}

// walkBody recursively walks microflow body statements looking for per-statement issues.
func (v *microflowValidator) walkBody(body []ast.MicroflowStatement) {
	for _, s := range body {
		switch stmt := s.(type) {
		case *ast.ValidationFeedbackStmt:
			if isEmptyMessage(stmt.Message) {
				v.addViolation("MDL007", linter.SeverityWarning,
					"validation feedback has empty message template. "+
						"Mendix requires a non-empty feedback message (CE0091).",
					"Add a message template to the validation feedback action")
			}
		case *ast.ReturnStmt:
			v.checkReturn(stmt)
			v.checkIdAccess(stmt.Value)
			v.checkCurrentUserPath(stmt.Value)
		case *ast.IfStmt:
			v.walkBody(stmt.ThenBody)
			v.walkBody(stmt.ElseBody)
		case *ast.EnumSplitStmt:
			// Mendix enumeration splits map to exclusive splits with one outgoing
			// flow per enum value. Multiple values per branch and a default (else)
			// flow are not supported — Studio Pro will reject both with CE errors.
			if len(stmt.ElseBody) > 0 {
				v.addViolation("MDL008", linter.SeverityError,
					fmt.Sprintf("case statement on '$%s' has an else branch; "+
						"Mendix enumeration splits do not support a default case. "+
						"Add an explicit when branch for every enum value instead.",
						stmt.Variable),
					"Add an explicit when branch for every enum value instead of using else")
			}
			for _, c := range stmt.Cases {
				if len(c.Values) > 1 {
					v.addViolation("MDL009", linter.SeverityError,
						fmt.Sprintf("case statement on '$%s': when branch lists %d values (%s); "+
							"Mendix enumeration splits require exactly one value per branch.",
							stmt.Variable, len(c.Values), strings.Join(c.Values, ", ")),
						"Split into separate when branches, one per enum value")
				}
				v.walkBody(c.Body)
			}
			v.walkBody(stmt.ElseBody)
		case *ast.InheritanceSplitStmt:
			for _, c := range stmt.Cases {
				v.walkBody(c.Body)
			}
			v.walkBody(stmt.ElseBody)
		case *ast.MfSetStmt:
			v.checkIdAccess(stmt.Value)
			v.checkCurrentUserPath(stmt.Value)
		case *ast.DeclareStmt:
			v.checkIdAccess(stmt.InitialValue)
			v.checkCurrentUserPath(stmt.InitialValue)
			// Track list variables declared as empty (candidates for the empty-list-in-loop anti-pattern)
			if stmt.Type.Kind == ast.TypeListOf {
				if isEmptyInit(stmt.InitialValue) {
					v.emptyListVars[stmt.Variable] = true
				}
			}
		case *ast.RetrieveStmt:
			v.checkWhereClause(stmt.Where)
			// RETRIEVE populates a list variable — remove from empty tracking
			delete(v.emptyListVars, stmt.Variable)
		case *ast.LoopStmt:
			// Check: nested loop anti-pattern
			if v.loopDepth > 0 {
				v.addViolation("MDL001", linter.SeverityWarning,
					"nested loop detected (loop inside a loop). "+
						"Use retrieve $Match from $List where ... limit 1 for list matching instead of nested loops (O(N^2) performance).",
					"Replace nested loop with retrieve ... where ... limit 1 for O(N) lookup")
			}
			// Check: loop over empty declared list
			if v.emptyListVars[stmt.ListVariable] {
				v.addViolation("MDL002", linter.SeverityWarning,
					fmt.Sprintf("loop iterates over '$%s' which was declared as an empty list and never populated. "+
						"Pass the list as a microflow parameter instead of creating an empty variable.",
						stmt.ListVariable),
					"Pass the list as a microflow parameter instead of creating an empty variable")
			}
			v.loopDepth++
			v.walkBody(stmt.Body)
			v.loopDepth--
		}
		// Check error handling inside loops
		if eh := stmtErrorHandling(s); eh != nil {
			v.checkErrorHandlingInLoop(s, eh)
			// Also walk ON ERROR bodies
			if len(eh.Body) > 0 {
				v.walkBody(eh.Body)
			}
		}
	}
}

// checkErrorHandlingInLoop warns if custom error handling is used inside a loop.
// Mendix requires error handling to be 'Rollback' inside looped activities (CE0644, CE6035).
func (v *microflowValidator) checkErrorHandlingInLoop(stmt ast.MicroflowStatement, eh *ast.ErrorHandlingClause) {
	if v.loopDepth == 0 {
		return // Not inside a loop
	}

	// Only Rollback is allowed inside loops
	if eh.Type != ast.ErrorHandlingRollback && eh.Type != "" {
		activityName := stmtActivityName(stmt)
		v.addViolation("MDL006", linter.SeverityWarning,
			fmt.Sprintf("%s has error handling type '%s' inside a loop. "+
				"Mendix requires error handling to be 'Rollback' inside looped activities (CE0644).",
				activityName, eh.Type),
			"Extract the activity with custom error handling into a submicroflow")
	}
}

// stmtActivityName returns a human-readable name for a statement type.
func stmtActivityName(stmt ast.MicroflowStatement) string {
	switch stmt.(type) {
	case *ast.CreateObjectStmt:
		return "create"
	case *ast.DeleteObjectStmt:
		return "delete"
	case *ast.MfCommitStmt:
		return "commit"
	case *ast.RetrieveStmt:
		return "retrieve"
	case *ast.CallMicroflowStmt:
		return "call microflow"
	case *ast.CallNanoflowStmt:
		return "call nanoflow"
	case *ast.CallJavaActionStmt:
		return "call java action"
	case *ast.CallJavaScriptActionStmt:
		return "call javascript action"
	case *ast.CallWebServiceStmt:
		return "call web service"
	case *ast.ExecuteDatabaseQueryStmt:
		return "execute database query"
	default:
		return "Activity"
	}
}

// checkReturn validates a RETURN statement against the microflow's return type.
func (v *microflowValidator) checkReturn(stmt *ast.ReturnStmt) {
	isVoid := v.returnType == nil || v.returnType.Type.Kind == ast.TypeVoid
	hasValue := stmt.Value != nil

	// Check 1: RETURN with no value when microflow has a return type
	if !isVoid && !hasValue {
		v.addViolation("MDL004", linter.SeverityError,
			fmt.Sprintf("return requires a value because microflow returns %s",
				returnTypeString(v.returnType)),
			fmt.Sprintf("Add a return value of type %s", returnTypeString(v.returnType)))
		return
	}

	// Check 2: RETURN with value when microflow returns Void
	if isVoid && hasValue {
		// Allow RETURN empty; on void microflows (it's a no-op)
		if lit, ok := stmt.Value.(*ast.LiteralExpr); ok {
			if lit.Kind == ast.LiteralEmpty || lit.Kind == ast.LiteralNull {
				return
			}
		}
		v.addViolation("MDL004", linter.SeverityError,
			"return has a value but microflow does not declare a return type",
			"Remove the return value or add a return type to the microflow")
		return
	}

	// Check 4: literal RETURN from entity-typed microflow
	if !isVoid && hasValue {
		retKind := v.returnType.Type.Kind
		if retKind == ast.TypeEntity || retKind == ast.TypeListOf {
			if isScalarLiteral(stmt.Value) {
				v.addViolation("MDL004", linter.SeverityError,
					fmt.Sprintf("return has a %s literal but microflow returns %s",
						literalKindName(stmt.Value), returnTypeString(v.returnType)),
					fmt.Sprintf("Return an object of type %s instead of a scalar literal", returnTypeString(v.returnType)))
			}
		}
	}
}

// isScalarLiteral returns true if the expression is a string, integer, boolean, or decimal literal.
func isScalarLiteral(expr ast.Expression) bool {
	lit, ok := expr.(*ast.LiteralExpr)
	if !ok {
		return false
	}
	switch lit.Kind {
	case ast.LiteralString, ast.LiteralInteger, ast.LiteralDecimal, ast.LiteralBoolean:
		return true
	}
	return false
}

// literalKindName returns a human-readable name for a literal expression's kind.
func literalKindName(expr ast.Expression) string {
	lit, ok := expr.(*ast.LiteralExpr)
	if !ok {
		return "unknown"
	}
	switch lit.Kind {
	case ast.LiteralString:
		return "String"
	case ast.LiteralInteger:
		return "Integer"
	case ast.LiteralDecimal:
		return "Decimal"
	case ast.LiteralBoolean:
		return "Boolean"
	default:
		return "unknown"
	}
}

// returnTypeString formats a MicroflowReturnType for display in messages.
func returnTypeString(rt *ast.MicroflowReturnType) string {
	if rt == nil {
		return "Void"
	}
	switch rt.Type.Kind {
	case ast.TypeEntity:
		if rt.Type.EntityRef != nil {
			return rt.Type.EntityRef.String()
		}
		return "Entity"
	case ast.TypeListOf:
		if rt.Type.EntityRef != nil {
			return "List of " + rt.Type.EntityRef.String()
		}
		return "List"
	default:
		return rt.Type.Kind.String()
	}
}

// bodyReturns returns true if all execution paths in the body end with a RETURN.
func bodyReturns(stmts []ast.MicroflowStatement) bool {
	if len(stmts) == 0 {
		return false
	}
	// Check from the last statement backwards for a RETURN or exhaustive IF/ELSE
	last := stmts[len(stmts)-1]
	switch s := last.(type) {
	case *ast.ReturnStmt:
		return true
	case *ast.IfStmt:
		// Both branches must return, and ELSE must be present
		return len(s.ElseBody) > 0 && bodyReturns(s.ThenBody) && bodyReturns(s.ElseBody)
	case *ast.WhileStmt:
		return isUnconditionalTrueWhile(s) && !containsBreakForCurrentLoop(s.Body)
	case *ast.EnumSplitStmt:
		// else is not supported by Mendix; treat the split as exhaustive if
		// every explicit case ends with a return. Unhandled enum values fall
		// through to the next statement, so callers should add a return after
		// end case when the split may not cover all values.
		if len(s.Cases) == 0 {
			return false
		}
		for _, c := range s.Cases {
			if !bodyReturns(c.Body) {
				return false
			}
		}
		return true
	case *ast.InheritanceSplitStmt:
		if len(s.Cases) == 0 || len(s.ElseBody) == 0 || !bodyReturns(s.ElseBody) {
			return false
		}
		for _, c := range s.Cases {
			if !bodyReturns(c.Body) {
				return false
			}
		}
		return true
	}
	return false
}

func isUnconditionalTrueWhile(s *ast.WhileStmt) bool {
	if s == nil {
		return false
	}
	lit, ok := s.Condition.(*ast.LiteralExpr)
	if !ok || lit.Kind != ast.LiteralBoolean {
		return false
	}
	value, ok := lit.Value.(bool)
	return ok && value
}

// checkBranchScoping detects variables declared inside IF/ELSE branches that are
// referenced in subsequent statements at the same level.
func (v *microflowValidator) checkBranchScoping(body []ast.MicroflowStatement) {
	// Collect variables that are only declared inside branches
	branchVars := make(map[string]string) // varName -> "IF branch" / "ELSE branch" / "ON ERROR body"

	for i, s := range body {
		switch stmt := s.(type) {
		case *ast.IfStmt:
			// Collect vars declared in THEN branch
			for varName := range collectDeclaredVars(stmt.ThenBody) {
				branchVars[varName] = "if branch"
			}
			// Collect vars declared in ELSE branch
			for varName := range collectDeclaredVars(stmt.ElseBody) {
				branchVars[varName] = "else branch"
			}
			// Recurse into branches for nested scoping checks
			v.checkBranchScoping(stmt.ThenBody)
			v.checkBranchScoping(stmt.ElseBody)
		case *ast.EnumSplitStmt:
			for _, c := range stmt.Cases {
				for varName := range collectDeclaredVars(c.Body) {
					branchVars[varName] = "enum split branch"
				}
				v.checkBranchScoping(c.Body)
			}
			for varName := range collectDeclaredVars(stmt.ElseBody) {
				branchVars[varName] = "enum split else branch"
			}
			v.checkBranchScoping(stmt.ElseBody)
		case *ast.InheritanceSplitStmt:
			for _, c := range stmt.Cases {
				for varName := range collectDeclaredVars(c.Body) {
					branchVars[varName] = "split type branch"
				}
				v.checkBranchScoping(c.Body)
			}
			for varName := range collectDeclaredVars(stmt.ElseBody) {
				branchVars[varName] = "split type else branch"
			}
			v.checkBranchScoping(stmt.ElseBody)
		case *ast.LoopStmt:
			v.checkBranchScoping(stmt.Body)
		}

		// Check ON ERROR bodies
		if eh := stmtErrorHandling(s); eh != nil && len(eh.Body) > 0 {
			for varName := range collectDeclaredVars(eh.Body) {
				branchVars[varName] = "on error body"
			}
			v.checkBranchScoping(eh.Body)
		}

		// After processing this statement, check if subsequent statements reference branch vars
		if len(branchVars) > 0 {
			for _, subsequent := range body[i+1:] {
				for _, refVar := range referencedVars(subsequent) {
					if scope, ok := branchVars[refVar]; ok {
						v.addViolation("MDL005", linter.SeverityWarning,
							fmt.Sprintf("variable '$%s' is declared inside %s but used outside",
								refVar, scope),
							fmt.Sprintf("Declare '$%s' before the if/else block", refVar))
						// Remove to avoid duplicate warnings
						delete(branchVars, refVar)
					}
				}
			}
		}
	}
}

// collectDeclaredVars returns the set of variable names declared in a body.
func collectDeclaredVars(body []ast.MicroflowStatement) map[string]bool {
	vars := make(map[string]bool)
	for _, s := range body {
		switch stmt := s.(type) {
		case *ast.DeclareStmt:
			vars[stmt.Variable] = true
		case *ast.CreateObjectStmt:
			if stmt.Variable != "" {
				vars[stmt.Variable] = true
			}
		case *ast.RetrieveStmt:
			if stmt.Variable != "" {
				vars[stmt.Variable] = true
			}
		case *ast.CallMicroflowStmt:
			if stmt.OutputVariable != "" {
				vars[stmt.OutputVariable] = true
			}
		case *ast.CallNanoflowStmt:
			if stmt.OutputVariable != "" {
				vars[stmt.OutputVariable] = true
			}
		case *ast.CallJavaActionStmt:
			if stmt.OutputVariable != "" {
				vars[stmt.OutputVariable] = true
			}
		case *ast.CallJavaScriptActionStmt:
			if stmt.OutputVariable != "" {
				vars[stmt.OutputVariable] = true
			}
		case *ast.ExecuteDatabaseQueryStmt:
			if stmt.OutputVariable != "" {
				vars[stmt.OutputVariable] = true
			}
		case *ast.ListOperationStmt:
			if stmt.OutputVariable != "" {
				vars[stmt.OutputVariable] = true
			}
		case *ast.AggregateListStmt:
			if stmt.OutputVariable != "" {
				vars[stmt.OutputVariable] = true
			}
		case *ast.CreateListStmt:
			if stmt.Variable != "" {
				vars[stmt.Variable] = true
			}
		case *ast.EnumSplitStmt:
		case *ast.CastObjectStmt:
			if stmt.OutputVariable != "" {
				vars[stmt.OutputVariable] = true
			}
		case *ast.InheritanceSplitStmt:
			for _, c := range stmt.Cases {
				for varName := range collectDeclaredVars(c.Body) {
					vars[varName] = true
				}
			}
			for varName := range collectDeclaredVars(stmt.ElseBody) {
				vars[varName] = true
			}
		}
	}
	return vars
}

// referencedVars returns the variable names referenced in a statement (SET targets, RETURN values, etc.).
func referencedVars(stmt ast.MicroflowStatement) []string {
	var refs []string
	switch s := stmt.(type) {
	case *ast.MfSetStmt:
		// SET $Var = expr — the target variable is a reference
		refs = append(refs, extractVarName(s.Target))
		refs = append(refs, exprVarRefs(s.Value)...)
	case *ast.ReturnStmt:
		if s.Value != nil {
			refs = append(refs, exprVarRefs(s.Value)...)
		}
	case *ast.ChangeObjectStmt:
		refs = append(refs, s.Variable)
	case *ast.MfCommitStmt:
		refs = append(refs, s.Variable)
	case *ast.DeleteObjectStmt:
		refs = append(refs, s.Variable)
	case *ast.AddToListStmt:
		if s.Value != nil {
			refs = append(refs, exprVarRefs(s.Value)...)
		} else {
			refs = append(refs, s.Item)
		}
		refs = append(refs, s.List)
	case *ast.RemoveFromListStmt:
		refs = append(refs, s.Item, s.List)
	case *ast.LogStmt:
		refs = append(refs, exprVarRefs(s.Node)...)
		refs = append(refs, exprVarRefs(s.Message)...)
	case *ast.EnumSplitStmt:
		refs = append(refs, extractVarName(s.Variable))
	case *ast.CastObjectStmt:
		if s.ObjectVariable != "" {
			refs = append(refs, s.ObjectVariable)
		}
	case *ast.InheritanceSplitStmt:
		refs = append(refs, s.Variable)
		for _, c := range s.Cases {
			for _, nested := range c.Body {
				refs = append(refs, referencedVars(nested)...)
			}
		}
		for _, nested := range s.ElseBody {
			refs = append(refs, referencedVars(nested)...)
		}
	}
	return refs
}

// extractVarName extracts the base variable name from a target that may include
// a $ prefix or attribute path (e.g., "$Var/Attr" → "Var").
func extractVarName(target string) string {
	name := strings.TrimPrefix(target, "$")
	if before, _, ok := strings.Cut(name, "/"); ok {
		return before
	}
	return name
}

// exprVarRefs extracts variable names referenced in an expression.
func exprVarRefs(expr ast.Expression) []string {
	if expr == nil {
		return nil
	}
	var refs []string
	switch e := expr.(type) {
	case *ast.VariableExpr:
		refs = append(refs, e.Name)
	case *ast.AttributePathExpr:
		refs = append(refs, e.Variable)
	case *ast.BinaryExpr:
		refs = append(refs, exprVarRefs(e.Left)...)
		refs = append(refs, exprVarRefs(e.Right)...)
	case *ast.UnaryExpr:
		refs = append(refs, exprVarRefs(e.Operand)...)
	case *ast.FunctionCallExpr:
		for _, arg := range e.Arguments {
			refs = append(refs, exprVarRefs(arg)...)
		}
	case *ast.ParenExpr:
		refs = append(refs, exprVarRefs(e.Inner)...)
	case *ast.IfThenElseExpr:
		refs = append(refs, exprVarRefs(e.Condition)...)
		refs = append(refs, exprVarRefs(e.ThenExpr)...)
		refs = append(refs, exprVarRefs(e.ElseExpr)...)
	case *ast.SourceExpr:
		refs = append(refs, exprVarRefs(e.Expression)...)
	}
	return refs
}

// stmtErrorHandling returns the ErrorHandlingClause for statements that support it.
func stmtErrorHandling(stmt ast.MicroflowStatement) *ast.ErrorHandlingClause {
	switch s := stmt.(type) {
	case *ast.CreateObjectStmt:
		return s.ErrorHandling
	case *ast.DeleteObjectStmt:
		return s.ErrorHandling
	case *ast.MfCommitStmt:
		return s.ErrorHandling
	case *ast.RetrieveStmt:
		return s.ErrorHandling
	case *ast.CallMicroflowStmt:
		return s.ErrorHandling
	case *ast.CallNanoflowStmt:
		return s.ErrorHandling
	case *ast.CallJavaActionStmt:
		return s.ErrorHandling
	case *ast.DownloadFileStmt:
		return s.ErrorHandling
	case *ast.CallJavaScriptActionStmt:
		return s.ErrorHandling
	case *ast.CallWebServiceStmt:
		return s.ErrorHandling
	case *ast.ExecuteDatabaseQueryStmt:
		return s.ErrorHandling
	}
	return nil
}

// isEmptyInit checks if a variable initializer is empty/nil (used to detect "DECLARE $List List of ... = empty").
func isEmptyInit(expr ast.Expression) bool {
	if expr == nil {
		return true
	}
	if lit, ok := expr.(*ast.LiteralExpr); ok {
		return lit.Kind == ast.LiteralEmpty || lit.Kind == ast.LiteralNull
	}
	return false
}

// checkWhereClause applies MDL014 and MDL015 to a RETRIEVE WHERE expression.
func (v *microflowValidator) checkWhereClause(where ast.Expression) {
	if where == nil {
		return
	}
	src := sourceOf(where)
	if src == "" {
		return
	}
	if reUppercaseAND.MatchString(src) {
		v.addViolation("MDL014", linter.SeverityError,
			"WHERE clause uses uppercase 'AND' — Mendix requires lowercase 'and' (CE0109).",
			"Replace 'AND' with 'and'. Prefer: WHERE (Condition1) and (Condition2).")
	}
	if reQuotedAssocName.MatchString(src) {
		v.addViolation("MDL015", linter.SeverityError,
			"WHERE clause uses a quoted association name (e.g. \"Module\".\"AssocName\") — Mendix requires dotted notation without quotes (CE0109).",
			"Replace \"Module\".\"AssocName\" with Module.AssocName.")
	}
}

// checkCurrentUserPath reports MDL016 when [%CurrentUser%]/Module.Type/attr
// is used instead of the direct [%CurrentUser%]/attr path.
func (v *microflowValidator) checkCurrentUserPath(expr ast.Expression) {
	if expr == nil {
		return
	}
	src := sourceOf(expr)
	if reCurrentUserPath.MatchString(src) {
		v.addViolation("MDL016", linter.SeverityError,
			"'[%CurrentUser%]/Module.Type/attr' traverses an intermediate entity type — Mendix resolves CurrentUser directly to System.User, so the intermediate step is invalid (CE0117).",
			"Use '[%CurrentUser%]/attr' directly, e.g. [%CurrentUser%]/Name.")
	}
}

// sourceOf returns the Source string of a SourceExpr, or empty string otherwise.
func sourceOf(expr ast.Expression) string {
	if se, ok := expr.(*ast.SourceExpr); ok {
		return se.Source
	}
	return ""
}

// bodyContainsIdPath reports whether any statement in the body (recursively)
// accesses /id through a SourceExpr — used to detect MDL017.
func bodyContainsIdPath(body []ast.MicroflowStatement) bool {
	for _, s := range body {
		if stmtSourceContainsIdPath(s) {
			return true
		}
	}
	return false
}

func stmtSourceContainsIdPath(stmt ast.MicroflowStatement) bool {
	switch s := stmt.(type) {
	case *ast.MfSetStmt:
		return containsIdPath(sourceOf(s.Value))
	case *ast.ReturnStmt:
		return containsIdPath(sourceOf(s.Value))
	case *ast.DeclareStmt:
		return containsIdPath(sourceOf(s.InitialValue))
	case *ast.IfStmt:
		return bodyContainsIdPath(s.ThenBody) || bodyContainsIdPath(s.ElseBody)
	case *ast.LoopStmt:
		return bodyContainsIdPath(s.Body)
	case *ast.EnumSplitStmt:
		for _, c := range s.Cases {
			if bodyContainsIdPath(c.Body) {
				return true
			}
		}
		return bodyContainsIdPath(s.ElseBody)
	}
	return false
}

// checkIdAccess walks an expression and reports MDL013 if $Var/id is found.
// $Object/id is illegal in Mendix microflow expressions; use an AutoNumber
// attribute or return the object directly. See hint E012 for fix options.
func (v *microflowValidator) checkIdAccess(expr ast.Expression) {
	if expr == nil {
		return
	}
	switch e := expr.(type) {
	case *ast.SourceExpr:
		// Source is the canonical string form; Expression is its parsed representation.
		// Check the Source string only — recursing into Expression would double-report
		// the same issue (both paths detect the same /id access).
		if containsIdPath(e.Source) {
			v.addViolation("MDL013", linter.SeverityError,
				fmt.Sprintf("expression contains '/id' which is illegal in Mendix microflow expressions — 'id' is a reserved system attribute (only valid in XPath constraints). Found in: %q", e.Source),
				"Run 'mxcli hint E012' for fix options (return the object, or add an AutoNumber attribute).")
		}
	case *ast.AttributePathExpr:
		for _, seg := range e.Path {
			if seg == "id" {
				v.addViolation("MDL013", linter.SeverityError,
					fmt.Sprintf("'$%s/id' is illegal in Mendix microflow expressions — 'id' is a reserved system attribute (only valid in XPath constraints).", e.Variable),
					"Run 'mxcli hint E012' for fix options (return the object, or add an AutoNumber attribute).")
				return
			}
		}
	case *ast.BinaryExpr:
		v.checkIdAccess(e.Left)
		v.checkIdAccess(e.Right)
	case *ast.UnaryExpr:
		v.checkIdAccess(e.Operand)
	case *ast.FunctionCallExpr:
		for _, arg := range e.Arguments {
			v.checkIdAccess(arg)
		}
	case *ast.ParenExpr:
		v.checkIdAccess(e.Inner)
	case *ast.IfThenElseExpr:
		v.checkIdAccess(e.Condition)
		v.checkIdAccess(e.ThenExpr)
		v.checkIdAccess(e.ElseExpr)
	}
}

// containsIdPath reports whether a raw expression string accesses /id.
// Matches $Var/id, $Var/Assoc/id, etc. Avoids false positives on attribute
// names that contain "id" as a substring (e.g. $Obj/UserId).
func containsIdPath(src string) bool {
	for i := 0; i+2 < len(src); i++ {
		if src[i] == '/' && src[i+1] == 'i' && src[i+2] == 'd' {
			// must be end-of-string or followed by non-identifier char
			end := i + 3
			if end >= len(src) || !isIdentChar(src[end]) {
				return true
			}
		}
	}
	return false
}


// isEmptyMessage checks if a message expression is empty or nil.
func isEmptyMessage(expr ast.Expression) bool {
	if expr == nil {
		return true
	}
	if lit, ok := expr.(*ast.LiteralExpr); ok {
		if lit.Kind == ast.LiteralString {
			if s, ok := lit.Value.(string); ok && s == "" {
				return true
			}
		}
		if lit.Kind == ast.LiteralEmpty || lit.Kind == ast.LiteralNull {
			return true
		}
	}
	return false
}
