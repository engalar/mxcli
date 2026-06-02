// SPDX-License-Identifier: Apache-2.0

// Stage 3.2.6.4: standalone semantic validator for microflow / nanoflow
// bodies, ported from cmd_microflows_builder_validate.go (deleted with
// the rest of the legacy `flowBuilder` family).
//
// Used by `mxcli check` (validate.go) to catch CE0111 duplicate-variable
// errors and undeclared-variable usage without needing a connected
// project. Reference resolution (callee microflow / nanoflow return
// types, attribute existence on entities) is NOT performed here — that
// happens later in validateMicroflowReferences (which has the catalog
// available). Callee output variables are recorded as "Unknown" so
// downstream statements pass the declared-check.

package executor

import (
	"fmt"
	"strings"

	"github.com/mendixlabs/mxcli/mdl/ast"
)

// ValidateMicroflowBody validates a microflow body for semantic errors
// without needing a connected backend. Returns the list of problems as
// human-readable strings (empty when the body is clean).
func ValidateMicroflowBody(s *ast.CreateMicroflowStmt) []string {
	var returnVar string
	if s.ReturnType != nil {
		returnVar = s.ReturnType.Variable
	}
	return validateFlowBody(s.Parameters, returnVar, s.Body)
}

// ValidateNanoflowBody is the nanoflow analogue of ValidateMicroflowBody.
func ValidateNanoflowBody(s *ast.CreateNanoflowStmt) []string {
	var returnVar string
	if s.ReturnType != nil {
		returnVar = s.ReturnType.Variable
	}
	return validateFlowBody(s.Parameters, returnVar, s.Body)
}

// validateFlowBody walks parameters and body statements, returning any
// validation errors detected. Same algorithm shape as the legacy
// flowBuilder validator — one error message per problem.
func validateFlowBody(params []ast.MicroflowParam, returnVar string, body []ast.MicroflowStatement) []string {
	v := newFlowValidator()

	// Register the named return variable (returns Type as $Var) so body
	// statements that assign or read it pass the declared-variable check.
	// Do NOT add it to flatOutputVarNames — the return variable may be
	// initialised inside the body (declare $Var …, retrieve … into $Var, etc.)
	// without being a duplicate declaration. flatOutputVarNames is only used to
	// catch genuinely duplicate output-variable declarations (CE0111), and the
	// return variable is expected to be written exactly once by the body.
	if returnVar != "" {
		v.declaredVars[returnVar] = "Unknown"
	}

	for _, p := range params {
		if ref := paramEntityRef(p.Type); ref != nil {
			if ref.Module == "" {
				v.errors = append(v.errors, fmt.Sprintf(
					"parameter '$%s': entity type '%s' is missing module prefix (use 'Module.%s')",
					p.Name, ref.Name, ref.Name))
				continue
			}
			entityQN := ref.Module + "." + ref.Name
			if p.Type.Kind == ast.TypeListOf {
				v.varTypes[p.Name] = "List of " + entityQN
			} else {
				v.varTypes[p.Name] = entityQN
			}
			v.flatOutputVarNames[p.Name] = true
		} else {
			v.declaredVars[p.Name] = p.Type.Kind.String()
			v.flatOutputVarNames[p.Name] = true
		}
	}
	if len(v.errors) > 0 {
		return v.errors
	}

	v.validateStatements(body)
	return v.errors
}

// flowValidator carries the type-tracking maps + error sink used by
// the recursive walker. Mirrors the shape of the legacy flowBuilder
// struct's validator-relevant fields, minus the backend / hierarchy /
// ID-generation state.
type flowValidator struct {
	varTypes           map[string]string
	declaredVars       map[string]string
	flatOutputVarNames map[string]bool // shared across all scoped copies — not cloned
	errors             []string
}

func newFlowValidator() *flowValidator {
	return &flowValidator{
		varTypes:           make(map[string]string),
		declaredVars:       make(map[string]string),
		flatOutputVarNames: make(map[string]bool),
	}
}

func (v *flowValidator) addError(format string, args ...any) {
	v.errors = append(v.errors, fmt.Sprintf(format, args...))
}

func (v *flowValidator) addErrorWithExample(message, example string) {
	v.errors = append(v.errors, fmt.Sprintf("%s\n\n  Example:\n%s", message, example))
}

func (v *flowValidator) isVariableDeclared(varName string) bool {
	name := varName
	if len(name) > 0 && name[0] == '$' {
		name = name[1:]
	}
	for i, c := range name {
		if c == '/' {
			name = name[:i]
			break
		}
	}
	if _, ok := v.varTypes[name]; ok {
		return true
	}
	if _, ok := v.declaredVars[name]; ok {
		return true
	}
	return false
}

func (v *flowValidator) validateOutputVariable(varName, statement string) {
	if varName == "" {
		return
	}
	name := strippedVarName(varName)
	if v.flatOutputVarNames[name] {
		v.addError("duplicate variable name '$%s' — %s output variable is already declared in this microflow (CE0111)", name, statement)
		return
	}
	v.flatOutputVarNames[name] = true
}

func (v *flowValidator) validateScopedStatements(stmts []ast.MicroflowStatement) {
	scoped := *v
	scoped.varTypes = cloneStringMap(v.varTypes)
	scoped.declaredVars = cloneStringMap(v.declaredVars)
	scoped.validateStatements(stmts)
	v.errors = scoped.errors
}

func (v *flowValidator) validateStatements(stmts []ast.MicroflowStatement) {
	for _, stmt := range stmts {
		v.validateStatement(stmt)
	}
}

func (v *flowValidator) validateStatement(stmt ast.MicroflowStatement) {
	switch s := stmt.(type) {
	case *ast.DeclareStmt:
		name := s.Variable
		if v.flatOutputVarNames[name] {
			v.addError("duplicate variable name '$%s' — variable is already declared in this microflow (CE0111)", name)
		} else {
			v.flatOutputVarNames[name] = true
		}
		if ref := paramEntityRef(s.Type); ref != nil {
			v.varTypes[s.Variable] = ref.Module + "." + ref.Name
		} else {
			v.declaredVars[s.Variable] = s.Type.Kind.String()
		}

	case *ast.MfSetStmt:
		name := strippedVarName(s.Target)
		if !v.isVariableDeclared(s.Target) {
			v.addErrorWithExample(
				fmt.Sprintf("variable '%s' is not declared", s.Target),
				errorExampleDeclareVariable(s.Target))
		} else if listType, ok := v.varTypes[name]; ok && strings.HasPrefix(listType, "List of ") {
			v.addError("cannot use 'set' on list variable '$%s' — use 'add ... to $%s' or re-assign via create list (CE7247)", name, name)
		}

	case *ast.IfStmt:
		v.validateScopedStatements(s.ThenBody)
		if len(s.ElseBody) > 0 {
			v.validateScopedStatements(s.ElseBody)
		}

	case *ast.EnumSplitStmt:
		if count := enumSplitBranchCount(s); count > maxEnumSplitBranches {
			v.addError("enum split has %d branches; at most %d branches are supported", count, maxEnumSplitBranches)
		}
		for _, c := range s.Cases {
			v.validateScopedStatements(c.Body)
		}
		if len(s.ElseBody) > 0 {
			v.validateScopedStatements(s.ElseBody)
		}

	case *ast.InheritanceSplitStmt:
		for _, c := range s.Cases {
			v.validateScopedStatements(c.Body)
		}
		if len(s.ElseBody) > 0 {
			v.validateScopedStatements(s.ElseBody)
		}

	case *ast.LoopStmt:
		if s.ListVariable != "" {
			if listType, ok := v.varTypes[s.ListVariable]; ok {
				if len(listType) > 8 && listType[:8] == "List of " {
					v.varTypes[s.LoopVariable] = listType[8:]
				}
			}
		}
		v.validateStatements(s.Body)

	case *ast.CreateObjectStmt:
		v.validateOutputVariable(s.Variable, "create")
		if s.Variable != "" && s.EntityType.Module != "" {
			v.varTypes[s.Variable] = s.EntityType.Module + "." + s.EntityType.Name
		}
		if s.ErrorHandling != nil && len(s.ErrorHandling.Body) > 0 {
			v.validateStatements(s.ErrorHandling.Body)
		}

	case *ast.ChangeObjectStmt:
		// ChangeObjectStmt has no ErrorHandling field — nothing to recurse into.
		_ = s

	case *ast.CallMicroflowStmt:
		if s.OutputVariable != "" {
			v.validateOutputVariable(s.OutputVariable, "call microflow")
			v.declaredVars[s.OutputVariable] = "Unknown"
		}
		if s.ErrorHandling != nil && len(s.ErrorHandling.Body) > 0 {
			v.validateStatements(s.ErrorHandling.Body)
		}

	case *ast.CallNanoflowStmt:
		if s.OutputVariable != "" {
			v.validateOutputVariable(s.OutputVariable, "call nanoflow")
			v.declaredVars[s.OutputVariable] = "Unknown"
		}
		if s.ErrorHandling != nil && len(s.ErrorHandling.Body) > 0 {
			v.validateStatements(s.ErrorHandling.Body)
		}

	case *ast.CallJavaActionStmt:
		if s.OutputVariable != "" {
			v.validateOutputVariable(s.OutputVariable, "call java action")
			v.declaredVars[s.OutputVariable] = "Unknown"
		}
		if s.ErrorHandling != nil && len(s.ErrorHandling.Body) > 0 {
			v.validateStatements(s.ErrorHandling.Body)
		}

	case *ast.DownloadFileStmt:
		if s.ErrorHandling != nil && len(s.ErrorHandling.Body) > 0 {
			v.validateStatements(s.ErrorHandling.Body)
		}

	case *ast.CallJavaScriptActionStmt:
		if s.OutputVariable != "" {
			v.validateOutputVariable(s.OutputVariable, "call javascript action")
			v.declaredVars[s.OutputVariable] = "Unknown"
		}
		if s.ErrorHandling != nil && len(s.ErrorHandling.Body) > 0 {
			v.validateStatements(s.ErrorHandling.Body)
		}

	case *ast.CallWebServiceStmt:
		if s.OutputVariable != "" {
			v.validateOutputVariable(s.OutputVariable, "call web service")
			v.declaredVars[s.OutputVariable] = "Unknown"
		}
		if s.ErrorHandling != nil && len(s.ErrorHandling.Body) > 0 {
			v.validateStatements(s.ErrorHandling.Body)
		}

	case *ast.ExecuteDatabaseQueryStmt:
		if s.OutputVariable != "" {
			v.validateOutputVariable(s.OutputVariable, "execute database query")
			v.declaredVars[s.OutputVariable] = "Unknown"
		}
		if s.ErrorHandling != nil && len(s.ErrorHandling.Body) > 0 {
			v.validateStatements(s.ErrorHandling.Body)
		}

	case *ast.CallExternalActionStmt:
		if s.OutputVariable != "" {
			v.validateOutputVariable(s.OutputVariable, "call external action")
			v.declaredVars[s.OutputVariable] = "Unknown"
		}
		if s.ErrorHandling != nil && len(s.ErrorHandling.Body) > 0 {
			v.validateStatements(s.ErrorHandling.Body)
		}

	case *ast.RestCallStmt:
		if s.OutputVariable != "" {
			v.validateOutputVariable(s.OutputVariable, "rest call")
			switch s.Result.Type {
			case ast.RestResultString:
				v.declaredVars[s.OutputVariable] = "String"
			case ast.RestResultResponse:
				v.declaredVars[s.OutputVariable] = "System.HttpResponse"
			case ast.RestResultMapping:
				if s.Result.ResultEntity.Module != "" {
					v.varTypes[s.OutputVariable] = s.Result.ResultEntity.Module + "." + s.Result.ResultEntity.Name
				} else {
					v.declaredVars[s.OutputVariable] = "Unknown"
				}
			default:
				v.declaredVars[s.OutputVariable] = "String"
			}
		}
		if s.ErrorHandling != nil && len(s.ErrorHandling.Body) > 0 {
			v.validateStatements(s.ErrorHandling.Body)
		}

	case *ast.SendRestRequestStmt:
		if s.OutputVariable != "" {
			v.validateOutputVariable(s.OutputVariable, "send rest request")
			v.declaredVars[s.OutputVariable] = "Unknown"
		}
		if s.ErrorHandling != nil && len(s.ErrorHandling.Body) > 0 {
			v.validateStatements(s.ErrorHandling.Body)
		}

	case *ast.MfCommitStmt:
		if s.ErrorHandling != nil && len(s.ErrorHandling.Body) > 0 {
			v.validateStatements(s.ErrorHandling.Body)
		}

	case *ast.DeleteObjectStmt:
		if s.ErrorHandling != nil && len(s.ErrorHandling.Body) > 0 {
			v.validateStatements(s.ErrorHandling.Body)
		}

	case *ast.RetrieveStmt:
		v.validateOutputVariable(s.Variable, "retrieve")
		if s.Variable != "" && s.Source.Module != "" {
			if s.StartVariable != "" {
				v.varTypes[s.Variable] = "List of " + s.Source.Module + "." + s.Source.Name
			} else if s.Limit == "1" {
				v.varTypes[s.Variable] = s.Source.Module + "." + s.Source.Name
			} else {
				v.varTypes[s.Variable] = "List of " + s.Source.Module + "." + s.Source.Name
			}
		}
		if s.ErrorHandling != nil && len(s.ErrorHandling.Body) > 0 {
			v.validateStatements(s.ErrorHandling.Body)
		}

	case *ast.CreateListStmt:
		v.validateOutputVariable(s.Variable, "create list")
		if s.Variable != "" && s.EntityType.Module != "" {
			v.varTypes[s.Variable] = "List of " + s.EntityType.Module + "." + s.EntityType.Name
		}

	case *ast.ListOperationStmt:
		v.validateOutputVariable(s.OutputVariable, "list operation")
		if s.OutputVariable != "" {
			v.declaredVars[s.OutputVariable] = "Unknown"
		}

	case *ast.AggregateListStmt:
		v.validateOutputVariable(s.OutputVariable, "aggregate list")
		if s.OutputVariable != "" {
			v.declaredVars[s.OutputVariable] = "Unknown"
		}

	case *ast.ImportFromMappingStmt:
		v.validateOutputVariable(s.OutputVariable, "import mapping")
		if s.OutputVariable != "" {
			v.declaredVars[s.OutputVariable] = "Unknown"
		}
		if s.ErrorHandling != nil && len(s.ErrorHandling.Body) > 0 {
			v.validateStatements(s.ErrorHandling.Body)
		}

	case *ast.ExportToMappingStmt:
		v.validateOutputVariable(s.OutputVariable, "export mapping")
		if s.OutputVariable != "" {
			v.declaredVars[s.OutputVariable] = "String"
		}
		if s.ErrorHandling != nil && len(s.ErrorHandling.Body) > 0 {
			v.validateStatements(s.ErrorHandling.Body)
		}

	case *ast.TransformJsonStmt:
		v.validateOutputVariable(s.OutputVariable, "transform json")
		if s.OutputVariable != "" {
			v.declaredVars[s.OutputVariable] = "String"
		}
		if s.ErrorHandling != nil && len(s.ErrorHandling.Body) > 0 {
			v.validateStatements(s.ErrorHandling.Body)
		}

		// Other statement types (ReturnStmt, RaiseErrorStmt, BreakStmt,
		// ContinueStmt, RollbackStmt, LogStmt, ShowPageStmt, etc.)
		// don't declare or reference variables so no validation needed
		// at this phase.
	}
}

// cloneStringMap returns a shallow copy of in. Returns nil when in is
// nil. Used to scope varTypes / declaredVars per-branch in the
// validator (and elsewhere in the gen builder).
func cloneStringMap(in map[string]string) map[string]string {
	if in == nil {
		return nil
	}
	out := make(map[string]string, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}

// enumSplitBranchCount returns the number of distinct branches an
// EnumSplitStmt produces (one per case + 1 if there's an else body).
// Pure AST.
func enumSplitBranchCount(s *ast.EnumSplitStmt) int {
	if s == nil {
		return 0
	}
	count := len(s.Cases)
	if len(s.ElseBody) > 0 {
		count++
	}
	return count
}

// suggestLoopVarName returns a human-readable rename hint for a
// conflicting loop variable. Tries to derive a descriptive name from
// the list element type; falls back to appending "2".
func suggestLoopVarName(conflictName, listVarName string, varTypes map[string]string) string {
	if listVarName != "" {
		if listType, ok := varTypes[listVarName]; ok {
			if len(listType) > 8 && listType[:8] == "List of " {
				after := listType[8:]
				lastDot := -1
				for i := 0; i < len(after); i++ {
					if after[i] == '.' {
						lastDot = i
					}
				}
				entity := after
				if lastDot >= 0 {
					entity = after[lastDot+1:]
				}
				return fmt.Sprintf("rename '$%s' to '$%sItem' (descriptive name from list element type)", conflictName, entity)
			}
		}
	}
	return fmt.Sprintf("rename '$%s' to '$%s2' or a more descriptive name", conflictName, conflictName)
}

// statementReferencesVar reports whether stmt directly references
// $varName (not via nested bodies). Used by gen flow builders for
// scoping decisions. Pure AST shape walk — only the easy cases that
// the gen builder checks (declare, set, return, change/create
// member-expression rewrites). Returns false for any unhandled type.
func statementReferencesVar(stmt ast.MicroflowStatement, varName string) bool {
	target := varName
	if len(target) > 0 && target[0] == '$' {
		target = target[1:]
	}
	switch s := stmt.(type) {
	case *ast.MfSetStmt:
		return strippedVarName(s.Target) == target
	case *ast.ReturnStmt:
		return exprReferencesVar(s.Value, target)
	case *ast.ChangeObjectStmt:
		return strippedVarName(s.Variable) == target
	case *ast.CreateObjectStmt:
		return s.Variable == target
	case *ast.DeleteObjectStmt:
		return strippedVarName(s.Variable) == target
	case *ast.MfCommitStmt:
		return strippedVarName(s.Variable) == target
	case *ast.LogStmt:
		return exprReferencesVar(s.Message, target)
	}
	return false
}

func strippedVarName(v string) string {
	if len(v) > 0 && v[0] == '$' {
		return v[1:]
	}
	return v
}

func exprReferencesVar(expr ast.Expression, varName string) bool {
	if expr == nil {
		return false
	}
	switch e := expr.(type) {
	case *ast.VariableExpr:
		return e.Name == varName
	case *ast.AttributePathExpr:
		return e.Variable == varName
	case *ast.BinaryExpr:
		return exprReferencesVar(e.Left, varName) || exprReferencesVar(e.Right, varName)
	case *ast.UnaryExpr:
		return exprReferencesVar(e.Operand, varName)
	case *ast.FunctionCallExpr:
		for _, arg := range e.Arguments {
			if exprReferencesVar(arg, varName) {
				return true
			}
		}
	case *ast.ParenExpr:
		return exprReferencesVar(e.Inner, varName)
	}
	return false
}

// outputDerivedVariable returns the destination variable of a
// statement that produces output derived from sourceVar (used by the
// gen builder to detect inferred type propagation chains). Returns
// "" for statements whose output is unrelated to sourceVar.
func outputDerivedVariable(stmt ast.MicroflowStatement, sourceVar string) string {
	switch s := stmt.(type) {
	case *ast.ListOperationStmt:
		if s.InputVariable == sourceVar {
			return s.OutputVariable
		}
	case *ast.RetrieveStmt:
		if s.StartVariable == sourceVar {
			return s.Variable
		}
	}
	return ""
}

// maxEnumSplitBranches caps the number of branches an EnumSplitStmt
// may have. Mirrors the legacy `len(splitCaseOrderAnchors)` constant
// (the order anchor table is also gone with the builder family — the
// underlying limit comes from Mendix's BSON serialization which only
// has slot anchors for 16 branches).
const maxEnumSplitBranches = 16
