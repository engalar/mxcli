// SPDX-License-Identifier: Apache-2.0

package ast

// ============================================================================
// Java Action Statements
// ============================================================================

// JavaActionParam represents a parameter in a Java action definition.
type JavaActionParam struct {
	Name       string   // Parameter name
	Type       DataType // Parameter type
	IsRequired bool     // NOT NULL constraint
}

// CreateJavaActionStmt represents:
//
//	CREATE JAVA ACTION Module.Name(
//	  EntityType: ENTITY <pEntity> NOT NULL,
//	  Source: pEntity NOT NULL
//	) RETURNS type
//	EXPOSED AS 'caption' IN 'category'
//	AS $$ ... $$;
type CreateJavaActionStmt struct {
	Name            QualifiedName     // Qualified name (Module.ActionName)
	Parameters      []JavaActionParam // Input parameters
	ReturnType      DataType          // Return type (can be nil for void)
	JavaCode        string            // The executeAction() body
	ExtraCode       string            // Optional extra code section
	Imports         []string          // Optional additional imports
	Documentation   string            // Optional documentation comment
	TypeParameters  []string          // Type parameter names (e.g., ["pEntity"])
	ExposedCaption  string            // EXPOSED AS 'caption'
	ExposedCategory string            // IN 'category'
	CreateOrModify  bool              // true for CREATE OR MODIFY / CREATE OR REPLACE
}

func (s *CreateJavaActionStmt) isStatement()     {}
func (s *CreateJavaActionStmt) TypeName() string { return "CreateJavaAction" }

// DropJavaActionStmt represents: DROP JAVA ACTION Module.Name
type DropJavaActionStmt struct {
	Name QualifiedName
}

func (s *DropJavaActionStmt) isStatement()     {}
func (s *DropJavaActionStmt) TypeName() string { return "DropJavaAction" }

// CreateJavaScriptActionStmt represents CREATE [OR MODIFY] JAVASCRIPT ACTION statements.
type CreateJavaScriptActionStmt struct {
	Name           QualifiedName
	Parameters     []JavaActionParam
	ReturnType     DataType
	Platform       string
	Folder         string
	Imports        string
	ExtraCode      string
	UserCode       string
	Documentation  string
	CreateOrModify bool
}

func (s *CreateJavaScriptActionStmt) isStatement()     {}
func (s *CreateJavaScriptActionStmt) TypeName() string { return "CreateJavaScriptAction" }
