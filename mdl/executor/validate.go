// SPDX-License-Identifier: Apache-2.0

// Package executor - MDL script validation (reference checking without execution).
package executor

import (
	"fmt"
	"strings"

	"github.com/mendixlabs/mxcli/mdl/ast"
)

// scriptContext holds objects defined within a script for reference validation.
type scriptContext struct {
	modules      map[string]bool // Modules created in the script
	entities     map[string]bool // Entities created (Module.Entity)
	enumerations map[string]bool // Enumerations created (Module.Enum)
	microflows   map[string]bool // Microflows created (Module.Microflow)
	pages        map[string]bool // Pages created (Module.Page)
	snippets     map[string]bool // Snippets created (Module.Snippet)
}

// newScriptContext creates a new script context.
func newScriptContext() *scriptContext {
	return &scriptContext{
		modules:      make(map[string]bool),
		entities:     make(map[string]bool),
		enumerations: make(map[string]bool),
		microflows:   make(map[string]bool),
		pages:        make(map[string]bool),
		snippets:     make(map[string]bool),
	}
}

// collectDefinitions scans a program and collects all objects that will be created.
func (sc *scriptContext) collectDefinitions(prog *ast.Program) {
	for _, stmt := range prog.Statements {
		switch s := stmt.(type) {
		case *ast.CreateModuleStmt:
			sc.modules[s.Name] = true
		case *ast.CreateEntityStmt:
			if s.Name.Module != "" {
				sc.entities[s.Name.String()] = true
			}
		case *ast.CreateViewEntityStmt:
			if s.Name.Module != "" {
				sc.entities[s.Name.String()] = true
			}
		case *ast.CreateExternalEntityStmt:
			if s.Name.Module != "" {
				sc.entities[s.Name.String()] = true
			}
		case *ast.CreateEnumerationStmt:
			if s.Name.Module != "" {
				sc.enumerations[s.Name.String()] = true
			}
		case *ast.CreateMicroflowStmt:
			if s.Name.Module != "" {
				sc.microflows[s.Name.String()] = true
			}
		case *ast.CreatePageStmtV3:
			if s.Name.Module != "" {
				sc.pages[s.Name.String()] = true
			}
		case *ast.CreateSnippetStmtV3:
			if s.Name.Module != "" {
				sc.snippets[s.Name.String()] = true
			}
		}
	}
}

// ValidateProgram validates all statements in a program, skipping references
// to objects that are defined within the script itself.
func (e *Executor) ValidateProgram(prog *ast.Program) []error {
	if e.reader == nil {
		return []error{fmt.Errorf("not connected to a project")}
	}

	// Collect all objects defined in the script
	sc := newScriptContext()
	sc.collectDefinitions(prog)

	// Validate each statement
	var errors []error
	for i, stmt := range prog.Statements {
		if err := e.validateWithContext(stmt, sc); err != nil {
			errors = append(errors, fmt.Errorf("statement %d: %w", i+1, err))
		}
	}
	return errors
}

// validateWithContext validates a statement, considering objects defined in the script.
func (e *Executor) validateWithContext(stmt ast.Statement, sc *scriptContext) error {
	switch s := stmt.(type) {
	// Statements that reference modules
	case *ast.CreateEntityStmt:
		if s.Name.Module != "" && !sc.modules[s.Name.Module] {
			if _, err := e.findModule(s.Name.Module); err != nil {
				return fmt.Errorf("module not found: %s", s.Name.Module)
			}
		}
		// Validate enumeration references in attributes
		for _, attr := range s.Attributes {
			if attr.Type.Kind == ast.TypeEnumeration && attr.Type.EnumRef != nil {
				enumRef := attr.Type.EnumRef
				// Check for missing module (common mistake - bare type name)
				if enumRef.Module == "" {
					return fmt.Errorf("attribute '%s': enumeration reference '%s' is missing module prefix. "+
						"Did you mean to use a built-in type like DateTime instead of DateAndTime?",
						attr.Name, enumRef.Name)
				}
				// Check if enumeration exists (in project or script)
				enumQN := enumRef.String()
				if !sc.enumerations[enumQN] {
					if !e.enumerationExists(enumQN) {
						return fmt.Errorf("attribute '%s': enumeration not found: %s", attr.Name, enumQN)
					}
				}
			}
		}
	case *ast.CreateAssociationStmt:
		if s.Name.Module != "" && !sc.modules[s.Name.Module] {
			if _, err := e.findModule(s.Name.Module); err != nil {
				return fmt.Errorf("module not found: %s", s.Name.Module)
			}
		}
		// Check parent and child entity references
		if s.Parent.Module != "" && !sc.modules[s.Parent.Module] {
			if _, err := e.findModule(s.Parent.Module); err != nil {
				return fmt.Errorf("parent entity module not found: %s", s.Parent.Module)
			}
		}
		if s.Child.Module != "" && !sc.modules[s.Child.Module] {
			if _, err := e.findModule(s.Child.Module); err != nil {
				return fmt.Errorf("child entity module not found: %s", s.Child.Module)
			}
		}
	case *ast.CreateImageCollectionStmt:
		if s.Name.Module != "" && !sc.modules[s.Name.Module] {
			if _, err := e.findModule(s.Name.Module); err != nil {
				return fmt.Errorf("module not found: %s", s.Name.Module)
			}
		}
	case *ast.DropImageCollectionStmt:
		if s.Name.Module != "" && !sc.modules[s.Name.Module] {
			if _, err := e.findModule(s.Name.Module); err != nil {
				return fmt.Errorf("module not found: %s", s.Name.Module)
			}
		}
	case *ast.CreateEnumerationStmt:
		if s.Name.Module != "" && !sc.modules[s.Name.Module] {
			if _, err := e.findModule(s.Name.Module); err != nil {
				return fmt.Errorf("module not found: %s", s.Name.Module)
			}
		}
	case *ast.CreateMicroflowStmt:
		if s.Name.Module != "" && !sc.modules[s.Name.Module] {
			if _, err := e.findModule(s.Name.Module); err != nil {
				return fmt.Errorf("module not found: %s", s.Name.Module)
			}
		}
		// Validate microflow body for semantic errors (e.g., undeclared variables)
		if validationErrors := ValidateMicroflowBody(s); len(validationErrors) > 0 {
			return fmt.Errorf("microflow '%s' has validation errors:\n  - %s",
				s.Name.String(), strings.Join(validationErrors, "\n  - "))
		}
		// Validate references inside microflow body (pages, microflows, java actions, entities)
		if refErrors := e.validateMicroflowReferences(s, sc); len(refErrors) > 0 {
			return fmt.Errorf("microflow '%s' has reference errors:\n  - %s",
				s.Name.String(), strings.Join(refErrors, "\n  - "))
		}
	case *ast.CreatePageStmtV3:
		if s.Name.Module != "" && !sc.modules[s.Name.Module] {
			if _, err := e.findModule(s.Name.Module); err != nil {
				return fmt.Errorf("module not found: %s", s.Name.Module)
			}
		}
		// Validate widget references (DataSource, Action, Snippet)
		if refErrors := e.validateWidgetReferences(s.Widgets, sc); len(refErrors) > 0 {
			return fmt.Errorf("page '%s' has reference errors:\n  - %s",
				s.Name.String(), strings.Join(refErrors, "\n  - "))
		}
	case *ast.CreateSnippetStmtV3:
		if s.Name.Module != "" && !sc.modules[s.Name.Module] {
			if _, err := e.findModule(s.Name.Module); err != nil {
				return fmt.Errorf("module not found: %s", s.Name.Module)
			}
		}
		// Validate widget references (DataSource, Action, Snippet)
		if refErrors := e.validateWidgetReferences(s.Widgets, sc); len(refErrors) > 0 {
			return fmt.Errorf("snippet '%s' has reference errors:\n  - %s",
				s.Name.String(), strings.Join(refErrors, "\n  - "))
		}
	case *ast.CreateViewEntityStmt:
		if s.Name.Module != "" && !sc.modules[s.Name.Module] {
			if _, err := e.findModule(s.Name.Module); err != nil {
				return fmt.Errorf("module not found: %s", s.Name.Module)
			}
		}
		// Validate OQL types match declared attribute types
		if typeErrors := e.ValidateViewEntityTypes(s); len(typeErrors) > 0 {
			return fmt.Errorf("view entity '%s' has type mismatches:\n  - %s",
				s.Name.String(), strings.Join(typeErrors, "\n  - "))
		}
	case *ast.AlterEntityStmt:
		if s.Name.Module != "" && !sc.modules[s.Name.Module] {
			if _, err := e.findModule(s.Name.Module); err != nil {
				return fmt.Errorf("module not found: %s", s.Name.Module)
			}
		}
		// Validate enumeration references in ADD ATTRIBUTE
		if s.Operation == ast.AlterEntityAddAttribute && s.Attribute != nil {
			attr := s.Attribute
			if attr.Type.Kind == ast.TypeEnumeration && attr.Type.EnumRef != nil {
				enumRef := attr.Type.EnumRef
				if enumRef.Module == "" {
					return fmt.Errorf("attribute '%s': enumeration reference '%s' is missing module prefix",
						attr.Name, enumRef.Name)
				}
				enumQN := enumRef.String()
				if !sc.enumerations[enumQN] {
					if !e.enumerationExists(enumQN) {
						return fmt.Errorf("attribute '%s': enumeration not found: %s", attr.Name, enumQN)
					}
				}
			}
		}
	case *ast.DropEntityStmt:
		if s.Name.Module != "" && !sc.modules[s.Name.Module] {
			if _, err := e.findModule(s.Name.Module); err != nil {
				return fmt.Errorf("module not found: %s", s.Name.Module)
			}
		}
	case *ast.DropModuleStmt:
		// For DROP, check if module exists in project OR will be created in script
		if !sc.modules[s.Name] {
			if _, err := e.findModule(s.Name); err != nil {
				return fmt.Errorf("module not found: %s", s.Name)
			}
		}

	// Query statements - no validation needed for basic ones
	case *ast.ShowStmt, *ast.DescribeStmt, *ast.SelectStmt:
		// These are read-only and will fail gracefully at execution
		return nil

	// Connection/session statements - no validation needed
	case *ast.ConnectStmt, *ast.DisconnectStmt, *ast.StatusStmt,
		*ast.SetStmt, *ast.HelpStmt, *ast.ExitStmt, *ast.ExecuteScriptStmt,
		*ast.UpdateStmt, *ast.RefreshStmt, *ast.RefreshCatalogStmt,
		*ast.SearchStmt:
		return nil

	default:
		// For unhandled statement types, skip validation
		return nil
	}

	return nil
}

// Validate checks if a statement's references are valid without executing it.
// This requires being connected to a project.
// Note: For validating entire programs with proper handling of script-defined objects,
// use ValidateProgram instead.
func (e *Executor) Validate(stmt ast.Statement) error {
	// Use validateWithContext with an empty script context for single statements
	return e.validateWithContext(stmt, newScriptContext())
}

// ----------------------------------------------------------------------------
// Microflow Body Reference Validation
// ----------------------------------------------------------------------------

// validateMicroflowReferences validates that all qualified name references in a
// microflow body (pages, microflows, java actions, entities) point to existing objects.
func (e *Executor) validateMicroflowReferences(s *ast.CreateMicroflowStmt, sc *scriptContext) []string {
	if e.reader == nil || len(s.Body) == 0 {
		return nil
	}

	// Collect all references from the microflow body
	refs := &microflowRefCollector{}
	refs.collectFromStatements(s.Body)

	if refs.empty() {
		return nil
	}

	var errors []string

	if len(refs.pages) > 0 {
		known := e.buildPageQualifiedNames()
		for _, ref := range refs.pages {
			if !known[ref] && !sc.pages[ref] {
				errors = append(errors, fmt.Sprintf("page not found: %s (referenced by SHOW PAGE)", ref))
			}
		}
	}

	if len(refs.microflows) > 0 {
		known := e.buildMicroflowQualifiedNames()
		for _, ref := range refs.microflows {
			if !known[ref] && !sc.microflows[ref] {
				errors = append(errors, fmt.Sprintf("microflow not found: %s (referenced by CALL MICROFLOW)", ref))
			}
		}
	}

	if len(refs.javaActions) > 0 {
		known := e.buildJavaActionQualifiedNames()
		for _, ref := range refs.javaActions {
			if !known[ref] {
				errors = append(errors, fmt.Sprintf("java action not found: %s (referenced by CALL JAVA ACTION)", ref))
			}
		}
	}

	if len(refs.entities) > 0 {
		known := e.buildEntityQualifiedNames()
		for _, ref := range refs.entities {
			if !known[ref.name] && !sc.entities[ref.name] {
				errors = append(errors, fmt.Sprintf("entity not found: %s (referenced by %s)", ref.name, ref.source))
			}
		}
	}

	return errors
}

// microflowRefCollector collects qualified name references from microflow statements.
type microflowRefCollector struct {
	pages       []string
	microflows  []string
	javaActions []string
	entities    []entityRef
}

// entityRef tracks an entity reference along with the statement that referenced it.
type entityRef struct {
	name   string
	source string // e.g., "CREATE", "RETRIEVE", "CREATE LIST OF"
}

func (c *microflowRefCollector) empty() bool {
	return len(c.pages) == 0 && len(c.microflows) == 0 &&
		len(c.javaActions) == 0 && len(c.entities) == 0
}

func (c *microflowRefCollector) collectFromStatements(stmts []ast.MicroflowStatement) {
	for _, stmt := range stmts {
		switch s := stmt.(type) {
		case *ast.ShowPageStmt:
			if s.PageName.Module != "" {
				c.pages = append(c.pages, s.PageName.String())
			}
		case *ast.CallMicroflowStmt:
			if s.MicroflowName.Module != "" {
				c.microflows = append(c.microflows, s.MicroflowName.String())
			}
		case *ast.CallJavaActionStmt:
			if s.ActionName.Module != "" {
				c.javaActions = append(c.javaActions, s.ActionName.String())
			}
		case *ast.CreateObjectStmt:
			if s.EntityType.Module != "" {
				c.entities = append(c.entities, entityRef{name: s.EntityType.String(), source: "CREATE"})
			}
		case *ast.RetrieveStmt:
			if s.StartVariable != "" {
				// Association retrieve — Source is an association name, not an entity; skip entity validation
			} else if s.Source.Module != "" {
				c.entities = append(c.entities, entityRef{name: s.Source.String(), source: "RETRIEVE"})
			}
		case *ast.CreateListStmt:
			if s.EntityType.Module != "" {
				c.entities = append(c.entities, entityRef{name: s.EntityType.String(), source: "CREATE LIST OF"})
			}
		case *ast.IfStmt:
			c.collectFromStatements(s.ThenBody)
			c.collectFromStatements(s.ElseBody)
		case *ast.LoopStmt:
			c.collectFromStatements(s.Body)
		}
		// Recurse into error handler bodies
		if eh := getErrorHandlerBody(stmt); eh != nil {
			c.collectFromStatements(eh)
		}
	}
}

// getErrorHandlerBody returns the custom error handler body if present, or nil.
func getErrorHandlerBody(stmt ast.MicroflowStatement) []ast.MicroflowStatement {
	switch s := stmt.(type) {
	case *ast.CreateObjectStmt:
		if s.ErrorHandling != nil && s.ErrorHandling.Body != nil {
			return s.ErrorHandling.Body
		}
	case *ast.RetrieveStmt:
		if s.ErrorHandling != nil && s.ErrorHandling.Body != nil {
			return s.ErrorHandling.Body
		}
	case *ast.CallMicroflowStmt:
		if s.ErrorHandling != nil && s.ErrorHandling.Body != nil {
			return s.ErrorHandling.Body
		}
	case *ast.CallJavaActionStmt:
		if s.ErrorHandling != nil && s.ErrorHandling.Body != nil {
			return s.ErrorHandling.Body
		}
	case *ast.ExecuteDatabaseQueryStmt:
		if s.ErrorHandling != nil && s.ErrorHandling.Body != nil {
			return s.ErrorHandling.Body
		}
	}
	return nil
}
