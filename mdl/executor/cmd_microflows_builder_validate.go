// SPDX-License-Identifier: Apache-2.0

// Package executor - Microflow flow graph: semantic validation
package executor

import (
	"fmt"

	"github.com/mendixlabs/mxcli/mdl/ast"
)

// ValidateMicroflowBody validates the microflow body for semantic errors without building objects.
// This is used by the check command to validate scripts without executing them.
func ValidateMicroflowBody(s *ast.CreateMicroflowStmt) []string {
	// Build variable maps from parameters
	varTypes := make(map[string]string)
	declaredVars := make(map[string]string)

	for _, p := range s.Parameters {
		if p.Type.EntityRef != nil {
			entityQN := p.Type.EntityRef.Module + "." + p.Type.EntityRef.Name
			if p.Type.Kind == ast.TypeListOf {
				varTypes[p.Name] = "List of " + entityQN
			} else {
				varTypes[p.Name] = entityQN
			}
		} else {
			// Primitive type parameters
			declaredVars[p.Name] = p.Type.Kind.String()
		}
	}

	// Create a validation-only flow builder
	fb := &flowBuilder{
		varTypes:     varTypes,
		declaredVars: declaredVars,
		errors:       []string{},
	}

	// Validate the body statements
	fb.validateStatements(s.Body)

	return fb.errors
}

// validateStatements recursively validates statements for semantic errors.
func (fb *flowBuilder) validateStatements(stmts []ast.MicroflowStatement) {
	for _, stmt := range stmts {
		fb.validateStatement(stmt)
	}
}

// validateStatement validates a single statement for semantic errors.
func (fb *flowBuilder) validateStatement(stmt ast.MicroflowStatement) {
	switch s := stmt.(type) {
	case *ast.DeclareStmt:
		// Check for duplicate variable declaration
		if fb.isVariableDeclared(s.Variable) {
			fb.addError("duplicate variable name '$%s' — variable is already declared (CE0111)", s.Variable)
		}
		// Register the variable as declared
		if s.Type.EntityRef != nil {
			// Entity type declaration
			fb.varTypes[s.Variable] = s.Type.EntityRef.Module + "." + s.Type.EntityRef.Name
		} else {
			// Primitive type declaration
			fb.declaredVars[s.Variable] = s.Type.Kind.String()
		}

	case *ast.MfSetStmt:
		// Validate that the variable has been declared
		if !fb.isVariableDeclared(s.Target) {
			fb.addErrorWithExample(
				fmt.Sprintf("variable '%s' is not declared", s.Target),
				errorExampleDeclareVariable(s.Target))
		}

	case *ast.IfStmt:
		// Validate then branch
		fb.validateStatements(s.ThenBody)
		// Validate else branch if present
		if len(s.ElseBody) > 0 {
			fb.validateStatements(s.ElseBody)
		}

	case *ast.LoopStmt:
		// Register loop variable (derived from list type)
		if s.ListVariable != "" {
			// Try to get the list type from varTypes
			if listType, ok := fb.varTypes[s.ListVariable]; ok {
				// "List of Module.Entity" -> "Module.Entity"
				if len(listType) > 8 && listType[:8] == "List of " {
					fb.varTypes[s.LoopVariable] = listType[8:]
				}
			}
		}
		// Validate loop body
		fb.validateStatements(s.Body)

	case *ast.CreateObjectStmt:
		// Check for duplicate variable — CREATE implicitly declares the variable
		if s.Variable != "" && fb.isVariableDeclared(s.Variable) {
			fb.addError("duplicate variable name '$%s' — CREATE implicitly declares the variable, remove the preceding DECLARE (CE0111)", s.Variable)
		}
		// Register created variable as entity type
		if s.Variable != "" && s.EntityType.Module != "" {
			fb.varTypes[s.Variable] = s.EntityType.Module + "." + s.EntityType.Name
		}
		// Validate error handler body if present
		if s.ErrorHandling != nil && len(s.ErrorHandling.Body) > 0 {
			fb.validateStatements(s.ErrorHandling.Body)
		}

	case *ast.CallMicroflowStmt:
		// Register result variable if assigned
		if s.OutputVariable != "" {
			// We don't know the return type, so just mark it as declared
			fb.declaredVars[s.OutputVariable] = "Unknown"
		}
		// Validate error handler body if present
		if s.ErrorHandling != nil && len(s.ErrorHandling.Body) > 0 {
			fb.validateStatements(s.ErrorHandling.Body)
		}

	case *ast.CallJavaActionStmt:
		// Register result variable if assigned
		if s.OutputVariable != "" {
			fb.declaredVars[s.OutputVariable] = "Unknown"
		}
		// Validate error handler body if present
		if s.ErrorHandling != nil && len(s.ErrorHandling.Body) > 0 {
			fb.validateStatements(s.ErrorHandling.Body)
		}

	case *ast.ExecuteDatabaseQueryStmt:
		if s.OutputVariable != "" {
			fb.declaredVars[s.OutputVariable] = "Unknown"
		}
		if s.ErrorHandling != nil && len(s.ErrorHandling.Body) > 0 {
			fb.validateStatements(s.ErrorHandling.Body)
		}

	case *ast.CallExternalActionStmt:
		// Register result variable if assigned
		if s.OutputVariable != "" {
			fb.declaredVars[s.OutputVariable] = "Unknown"
		}
		// Validate error handler body if present
		if s.ErrorHandling != nil && len(s.ErrorHandling.Body) > 0 {
			fb.validateStatements(s.ErrorHandling.Body)
		}

	case *ast.RestCallStmt:
		// Register result variable if assigned
		if s.OutputVariable != "" {
			// Type depends on result handling
			switch s.Result.Type {
			case ast.RestResultString:
				fb.declaredVars[s.OutputVariable] = "String"
			case ast.RestResultResponse:
				fb.declaredVars[s.OutputVariable] = "System.HttpResponse"
			case ast.RestResultMapping:
				if s.Result.ResultEntity.Module != "" {
					fb.varTypes[s.OutputVariable] = s.Result.ResultEntity.Module + "." + s.Result.ResultEntity.Name
				} else {
					fb.declaredVars[s.OutputVariable] = "Unknown"
				}
			default:
				fb.declaredVars[s.OutputVariable] = "String"
			}
		}
		// Validate error handler body if present
		if s.ErrorHandling != nil && len(s.ErrorHandling.Body) > 0 {
			fb.validateStatements(s.ErrorHandling.Body)
		}

	case *ast.SendRestRequestStmt:
		// Register output variable if assigned
		if s.OutputVariable != "" {
			fb.declaredVars[s.OutputVariable] = "Unknown" // Type depends on operation response mapping
		}
		// Validate error handler body if present
		if s.ErrorHandling != nil && len(s.ErrorHandling.Body) > 0 {
			fb.validateStatements(s.ErrorHandling.Body)
		}

	case *ast.MfCommitStmt:
		// Validate error handler body if present
		if s.ErrorHandling != nil && len(s.ErrorHandling.Body) > 0 {
			fb.validateStatements(s.ErrorHandling.Body)
		}

	case *ast.DeleteObjectStmt:
		// Validate error handler body if present
		if s.ErrorHandling != nil && len(s.ErrorHandling.Body) > 0 {
			fb.validateStatements(s.ErrorHandling.Body)
		}

	case *ast.RollbackStmt:
		// No error handling to validate

	case *ast.RetrieveStmt:
		// Check for duplicate variable — RETRIEVE implicitly declares the variable
		if s.Variable != "" && fb.isVariableDeclared(s.Variable) {
			fb.addError("duplicate variable name '$%s' — RETRIEVE implicitly declares the variable, remove the preceding DECLARE (CE0111)", s.Variable)
		}
		// Register retrieved variable
		if s.Variable != "" && s.Source.Module != "" {
			if s.StartVariable != "" {
				// Association retrieve always returns a list
				fb.varTypes[s.Variable] = "List of " + s.Source.Module + "." + s.Source.Name
			} else if s.Limit == "1" {
				fb.varTypes[s.Variable] = s.Source.Module + "." + s.Source.Name
			} else {
				fb.varTypes[s.Variable] = "List of " + s.Source.Module + "." + s.Source.Name
			}
		}
		// Validate error handler body if present
		if s.ErrorHandling != nil && len(s.ErrorHandling.Body) > 0 {
			fb.validateStatements(s.ErrorHandling.Body)
		}

		// Other statement types don't need validation for variable declarations
	}
}
