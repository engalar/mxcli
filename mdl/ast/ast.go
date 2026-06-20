// SPDX-License-Identifier: Apache-2.0

// Package ast defines the Abstract Syntax Tree nodes for MDL (Mendix Definition Language).
// This package contains types for the domain model subset: entities, attributes,
// associations, enumerations, and view entities.
package ast

// Statement represents any MDL statement that can be executed.
type Statement interface {
	isStatement()
	TypeName() string
}

// Position represents a location in the domain model canvas.
type Position struct {
	X int
	Y int
}

// QualifiedName represents a module-qualified name like "Module.Entity".
type QualifiedName struct {
	Module string
	Name   string
}

func (q QualifiedName) String() string {
	if q.Module == "" {
		return q.Name
	}
	return q.Module + "." + q.Name
}

// ============================================================================
// Program
// ============================================================================

// Program represents a complete MDL program (sequence of statements).
type Program struct {
	Statements []Statement
}

// ============================================================================
// Move Statement
// ============================================================================

// DocumentType represents the type of document being moved.
type DocumentType string

const (
	DocumentTypePage               DocumentType = "PAGE"
	DocumentTypeMicroflow          DocumentType = "MICROFLOW"
	DocumentTypeSnippet            DocumentType = "SNIPPET"
	DocumentTypeNanoflow           DocumentType = "NANOFLOW"
	DocumentTypeEntity             DocumentType = "ENTITY"
	DocumentTypeEnumeration        DocumentType = "ENUMERATION"
	DocumentTypeConstant           DocumentType = "CONSTANT"
	DocumentTypeDatabaseConnection DocumentType = "DATABASE CONNECTION"
	DocumentTypeJavaAction         DocumentType = "JAVA ACTION"
	DocumentTypeJavaScriptAction   DocumentType = "JAVASCRIPT ACTION"
	DocumentTypeLayout             DocumentType = "LAYOUT"
	DocumentTypeWorkflow           DocumentType = "WORKFLOW"
)

// MoveStmt represents: MOVE PAGE/MICROFLOW/SNIPPET/NANOFLOW/ENTITY/ENUMERATION Module.Name TO FOLDER 'path' IN Module
type MoveStmt struct {
	DocumentType DocumentType  // PAGE, MICROFLOW, SNIPPET, NANOFLOW, ENTITY, ENUMERATION
	Name         QualifiedName // Source document qualified name
	Folder       string        // Target folder path (empty = module root)
	TargetModule string        // Target module name (empty = same module)
}

func (s *MoveStmt) isStatement() {}
func (s *MoveStmt) TypeName() string { return "Move" }
